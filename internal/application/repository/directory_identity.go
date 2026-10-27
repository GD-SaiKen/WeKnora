package repository

import (
	"context"
	"errors"

	"github.com/Tencent/WeKnora/internal/directory"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"gorm.io/gorm"
)

type directoryIdentityRepository struct {
	db *gorm.DB
}

// NewDirectoryIdentityRepository creates a directory identity repository.
func NewDirectoryIdentityRepository(db *gorm.DB) interfaces.DirectoryIdentityRepository {
	return &directoryIdentityRepository{db: db}
}

func (r *directoryIdentityRepository) GetUserIDBySubject(ctx context.Context, subject string) (string, error) {
	var row types.UserDirectoryIdentity
	err := r.db.WithContext(ctx).Where("subject = ?", subject).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return row.UserID, nil
}

func (r *directoryIdentityRepository) GetUserIDByEmailAlias(ctx context.Context, normalizedEmail string) (string, error) {
	email := directory.NormalizeEmail(normalizedEmail)
	var row types.UserEmailAlias
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return row.UserID, nil
}

func (r *directoryIdentityRepository) GetUserIDByAIHubUserID(ctx context.Context, aiHubUserID string) (string, error) {
	var row types.UserDirectoryIdentity
	err := r.db.WithContext(ctx).Where("ai_hub_user_id = ?", aiHubUserID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return row.UserID, nil
}
