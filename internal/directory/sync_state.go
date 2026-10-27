package directory

import "time"

// SyncFailure is one skipped row in a run.
type SyncFailure struct {
	AIHubUserID string        `json:"ai_hub_user_id"`
	Reason      FailureReason `json:"reason"`
	Message     string        `json:"message,omitempty"`
}

// SyncState is the latest directory sync status for admins.
type SyncState struct {
	Status              string        `json:"status"`
	SynchronizedUsers   int           `json:"synchronized_users"`
	Error               string        `json:"error,omitempty"`
	Failures            []SyncFailure `json:"failures,omitempty"`
	StartedAt           time.Time     `json:"started_at,omitempty"`
	FinishedAt          *time.Time    `json:"finished_at,omitempty"`
}
