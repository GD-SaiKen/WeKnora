package directory

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client fetches directory pages from ai-hub Platform API.
type Client struct {
	baseURL string
	client  *http.Client
}

// NewClient builds a client for {base}/platform-api/v1.
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

type pageResponse struct {
	Items      []Entry  `json:"items"`
	HasMore    bool     `json:"has_more"`
	NextCursor *string  `json:"next_cursor"`
}

// AllPages returns every entry by following cursors until has_more is false.
func (c *Client) AllPages(ctx context.Context, accessToken, applicationID string) ([]Entry, error) {
	var out []Entry
	cursor := ""
	for page := 0; page < 10_000; page++ {
		pageResp, next, err := c.fetchPage(ctx, accessToken, applicationID, cursor)
		if err != nil {
			return nil, err
		}
		out = append(out, pageResp.Items...)
		if !pageResp.HasMore {
			break
		}
		if next == "" || next == cursor {
			return nil, fmt.Errorf("员工目录游标没有向前推进")
		}
		cursor = next
	}
	return out, nil
}

func (c *Client) fetchPage(ctx context.Context, accessToken, applicationID, cursor string) (pageResponse, string, error) {
	q := url.Values{}
	q.Set("limit", "500")
	if cursor != "" {
		q.Set("cursor", cursor)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/platform-api/v1/directory/users?"+q.Encode(), nil)
	if err != nil {
		return pageResponse{}, "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("X-Application-ID", applicationID)
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return pageResponse{}, "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return pageResponse{}, "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return pageResponse{}, "", fmt.Errorf("directory users: status %d", resp.StatusCode)
	}
	var parsed apiPageResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return pageResponse{}, "", err
	}
	items := make([]Entry, 0, len(parsed.Items))
	for _, row := range parsed.Items {
		items = append(items, Entry{
			UserID:           row.UserID,
			Subject:          row.Subject,
			Email:            row.Email,
			DisplayName:      row.DisplayName,
			OrganizationName: row.OrganizationName,
			Status:           row.Status,
			Tombstone:        row.Tombstone,
		})
	}
	next := ""
	if parsed.NextCursor != nil {
		next = *parsed.NextCursor
	}
	return pageResponse{Items: items, HasMore: parsed.HasMore, NextCursor: parsed.NextCursor}, next, nil
}

type apiPageResponse struct {
	Items      []apiDirectoryUser `json:"items"`
	HasMore    bool               `json:"has_more"`
	NextCursor *string            `json:"next_cursor"`
}

type apiDirectoryUser struct {
	UserID           string `json:"user_id"`
	Subject          string `json:"subject"`
	Email            string `json:"email"`
	DisplayName      string `json:"display_name"`
	OrganizationName string `json:"organization_name"`
	Status           string `json:"status"`
	Tombstone        bool   `json:"tombstone"`
}
