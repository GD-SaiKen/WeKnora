package directory

import "strings"

// NormalizeEmail trims and lowercases emails for stable directory matching.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// Entry is one row from the ai-hub employee directory API.
type Entry struct {
	UserID           string
	Subject          string
	Email            string
	DisplayName      string
	OrganizationName string
	Status           string
	Tombstone        bool
}

// Action is the outcome of applying one directory entry locally.
type Action string

const (
	ActionBound    Action = "bound"
	ActionCreated  Action = "created"
	ActionSkipped  Action = "skipped"
	ActionDisabled Action = "disabled"
)

// FailureReason is a stable code for sync failure rows.
type FailureReason string

const (
	FailureSubjectTaken    FailureReason = "subject_taken"
	FailureEmailTaken      FailureReason = "email_taken"
	FailureLastSystemAdmin FailureReason = "last_system_admin"
)
