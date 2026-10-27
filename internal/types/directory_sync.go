package types

import "time"

// UserDirectoryIdentity maps a WeKnora user to ai-hub directory fields.
type UserDirectoryIdentity struct {
	UserID           string    `gorm:"column:user_id;primaryKey"`
	AIHubUserID      string    `gorm:"column:ai_hub_user_id;uniqueIndex"`
	Subject          string    `gorm:"column:subject;uniqueIndex"`
	DirectoryEmail   string    `gorm:"column:directory_email"`
	DisplayName      string    `gorm:"column:display_name"`
	OrganizationName string    `gorm:"column:organization_name"`
	UpdatedAt        time.Time `gorm:"column:updated_at"`
}

func (UserDirectoryIdentity) TableName() string { return "user_directory_identities" }

// UserEmailAlias links an extra email to a user account.
type UserEmailAlias struct {
	Email  string `gorm:"column:email;primaryKey"`
	UserID string `gorm:"column:user_id;index"`
}

func (UserEmailAlias) TableName() string { return "user_email_aliases" }

// DirectorySyncRun records one directory synchronization attempt.
type DirectorySyncRun struct {
	ID                 string     `gorm:"column:id;primaryKey"`
	Status             string     `gorm:"column:status"`
	FullSync           bool       `gorm:"column:full_sync"`
	Cursor             string     `gorm:"column:cursor"`
	SynchronizedUsers  int        `gorm:"column:synchronized_users"`
	Error              string     `gorm:"column:error"`
	FailuresJSON       string     `gorm:"column:failures"`
	StartedAt          time.Time  `gorm:"column:started_at"`
	FinishedAt         *time.Time `gorm:"column:finished_at"`
}

func (DirectorySyncRun) TableName() string { return "directory_sync_runs" }
