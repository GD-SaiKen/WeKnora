package repository

import (
	"context"
	"errors"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"gorm.io/gorm"
)

type sharedMCPRepository struct {
	db *gorm.DB
}

// NewSharedMCPRepository creates a GORM-backed shared MCP repository.
func NewSharedMCPRepository(db *gorm.DB) interfaces.SharedMCPRepository {
	return &sharedMCPRepository{db: db}
}

func (r *sharedMCPRepository) GetConfig(ctx context.Context, tenantID uint64) (*types.SharedMCPConfig, error) {
	var row types.SharedMCPConfig
	err := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *sharedMCPRepository) UpsertConfig(ctx context.Context, cfg *types.SharedMCPConfig) error {
	return r.db.WithContext(ctx).Save(cfg).Error
}

func (r *sharedMCPRepository) CreateClient(ctx context.Context, client *types.SharedMCPClient) error {
	return r.db.WithContext(ctx).Create(client).Error
}

func (r *sharedMCPRepository) GetClient(ctx context.Context, id string) (*types.SharedMCPClient, error) {
	var row types.SharedMCPClient
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *sharedMCPRepository) CreateAuthCode(ctx context.Context, row *types.SharedMCPAuthCode) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *sharedMCPRepository) ConsumeAuthCode(ctx context.Context, codeHash string, now time.Time) (*types.SharedMCPAuthCode, error) {
	var row types.SharedMCPAuthCode
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("code_hash = ?", codeHash).First(&row).Error; err != nil {
			return err
		}
		if row.UsedAt != nil || !row.ExpiresAt.After(now) {
			return gorm.ErrRecordNotFound
		}
		row.UsedAt = &now
		return tx.Save(&row).Error
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *sharedMCPRepository) CreateGrant(ctx context.Context, grant *types.SharedMCPGrant) error {
	return r.db.WithContext(ctx).Create(grant).Error
}

func (r *sharedMCPRepository) GetGrant(ctx context.Context, id string) (*types.SharedMCPGrant, error) {
	var row types.SharedMCPGrant
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *sharedMCPRepository) UpdateGrantRefresh(
	ctx context.Context,
	grantID, refreshHash string,
	refreshExpiresAt time.Time,
) error {
	return r.db.WithContext(ctx).Model(&types.SharedMCPGrant{}).
		Where("id = ?", grantID).
		Updates(map[string]interface{}{
			"refresh_token_hash": refreshHash,
			"refresh_expires_at": refreshExpiresAt,
			"updated_at":         time.Now(),
		}).Error
}

func (r *sharedMCPRepository) RevokeGrant(ctx context.Context, grantID string, now time.Time) error {
	return r.db.WithContext(ctx).Model(&types.SharedMCPGrant{}).
		Where("id = ?", grantID).
		Update("revoked_at", now).Error
}

func (r *sharedMCPRepository) ListGrantsByUser(ctx context.Context, userID string) ([]*types.SharedMCPGrant, error) {
	var rows []*types.SharedMCPGrant
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&rows).Error
	return rows, err
}

func (r *sharedMCPRepository) ListEnabledTenantIDs(ctx context.Context) ([]uint64, error) {
	var ids []uint64
	err := r.db.WithContext(ctx).Model(&types.SharedMCPConfig{}).
		Where("enabled = ?", true).
		Pluck("tenant_id", &ids).Error
	return ids, err
}
