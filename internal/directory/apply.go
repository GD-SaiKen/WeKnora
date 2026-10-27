package directory

import (
	"context"
	"strings"
)

// UserSnapshot is the local user fields Apply needs.
type UserSnapshot struct {
	ID                   string
	Email                string
	PersonalTenantID     uint64
	PasswordLoginEnabled bool
	HasDirectoryIdentity bool
	AIHubUserID          string
}

// ApplyStore reads and writes users for directory reconciliation.
type ApplyStore interface {
	FindUserByNormalizedEmail(ctx context.Context, email string) (*UserSnapshot, error)
	FindUserBySubject(ctx context.Context, subject string) (*UserSnapshot, error)
	FindUserByAIHubUserID(ctx context.Context, aiHubUserID string) (*UserSnapshot, error)
	UserIDByNormalizedEmail(ctx context.Context, email string) (string, error)
	BindDirectoryIdentity(ctx context.Context, userID string, entry Entry) error
	MarkUserDisabled(ctx context.Context, userID string) error
	CreateDirectoryUser(ctx context.Context, entry Entry) (userID string, personalTenantID uint64, err error)
	UpdateDirectoryIdentityFields(ctx context.Context, userID string, entry Entry) error
}

// ApplyResult describes what happened to one directory row.
type ApplyResult struct {
	Action Action
	UserID string
	Reason FailureReason
}

// Apply reconciles one directory entry against the local store.
func Apply(ctx context.Context, store ApplyStore, entry Entry) (ApplyResult, error) {
	email := NormalizeEmail(entry.Email)
	subject := strings.TrimSpace(entry.Subject)
	if email == "" || subject == "" || strings.TrimSpace(entry.UserID) == "" {
		return ApplyResult{Action: ActionSkipped, Reason: FailureEmailTaken}, nil
	}

	inactive := entry.Tombstone || !strings.EqualFold(strings.TrimSpace(entry.Status), "active")

	if byHub, err := store.FindUserByAIHubUserID(ctx, entry.UserID); err != nil {
		return ApplyResult{}, err
	} else if byHub != nil {
		if inactive {
			if err := store.MarkUserDisabled(ctx, byHub.ID); err != nil {
				return ApplyResult{}, err
			}
			return ApplyResult{Action: ActionDisabled, UserID: byHub.ID}, nil
		}
		if err := store.UpdateDirectoryIdentityFields(ctx, byHub.ID, entry); err != nil {
			return ApplyResult{}, err
		}
		if email != NormalizeEmail(byHub.Email) {
			owner, err := store.UserIDByNormalizedEmail(ctx, email)
			if err != nil {
				return ApplyResult{}, err
			}
			if owner != "" && owner != byHub.ID {
				return ApplyResult{Action: ActionSkipped, UserID: byHub.ID, Reason: FailureEmailTaken}, nil
			}
		}
		return ApplyResult{Action: ActionBound, UserID: byHub.ID}, nil
	}

	if bySub, err := store.FindUserBySubject(ctx, subject); err != nil {
		return ApplyResult{}, err
	} else if bySub != nil && bySub.AIHubUserID != "" && bySub.AIHubUserID != entry.UserID {
		return ApplyResult{Action: ActionSkipped, Reason: FailureSubjectTaken}, nil
	}

	if byEmail, err := store.FindUserByNormalizedEmail(ctx, email); err != nil {
		return ApplyResult{}, err
	} else if byEmail != nil {
		if byEmail.HasDirectoryIdentity && byEmail.AIHubUserID != entry.UserID {
			return ApplyResult{Action: ActionSkipped, Reason: FailureSubjectTaken}, nil
		}
		if inactive {
			if err := store.MarkUserDisabled(ctx, byEmail.ID); err != nil {
				return ApplyResult{}, err
			}
			return ApplyResult{Action: ActionDisabled, UserID: byEmail.ID}, nil
		}
		if err := store.BindDirectoryIdentity(ctx, byEmail.ID, entry); err != nil {
			return ApplyResult{}, err
		}
		if err := store.UpdateDirectoryIdentityFields(ctx, byEmail.ID, entry); err != nil {
			return ApplyResult{}, err
		}
		return ApplyResult{Action: ActionBound, UserID: byEmail.ID}, nil
	}

	if inactive {
		return ApplyResult{Action: ActionSkipped}, nil
	}

	userID, _, err := store.CreateDirectoryUser(ctx, entry)
	if err != nil {
		return ApplyResult{}, err
	}
	return ApplyResult{Action: ActionCreated, UserID: userID}, nil
}
