package directory

import (
	"context"
	"errors"
)

// ErrShellInUse means the directory shell account already has data and cannot merge.
var ErrShellInUse = errors.New("directory shell account is in use")

// ShellUsage describes whether a directory-created account can be retired.
type ShellUsage struct {
	KnowledgeBaseCount      int
	OtherOrgMembershipCount int
	ExtraTenantMemberCount  int
	ShellPersonalTenantID   uint64
}

// BindStore performs account merge for mismatched emails.
type BindStore interface {
	LookupShellUserIDByAIHub(ctx context.Context, aiHubUserID string) (string, error)
	LoadShellUsage(ctx context.Context, shellUserID string, companyOrgID string) (ShellUsage, error)
	TransferIdentity(ctx context.Context, oldUserID, shellUserID, companyEmail string) error
	DeactivateUser(ctx context.Context, userID string) error
	RemoveCompanyMembership(ctx context.Context, shellPersonalTenantID uint64, companyOrgID string) error
}

// BindInput names the merge participants.
type BindInput struct {
	OldUserID    string
	AIHubUserID  string
	CompanyOrgID string
}

// BindResult returns the surviving account id.
type BindResult struct {
	OldUserID string
}

// Bind moves a directory identity from an unused shell onto an existing local account.
func Bind(ctx context.Context, store BindStore, input BindInput) (BindResult, error) {
	shellUserID, err := store.LookupShellUserIDByAIHub(ctx, input.AIHubUserID)
	if err != nil {
		return BindResult{}, err
	}
	if shellUserID == "" || shellUserID == input.OldUserID {
		return BindResult{OldUserID: input.OldUserID}, nil
	}
	usage, err := store.LoadShellUsage(ctx, shellUserID, input.CompanyOrgID)
	if err != nil {
		return BindResult{}, err
	}
	if usage.KnowledgeBaseCount > 0 || usage.OtherOrgMembershipCount > 0 || usage.ExtraTenantMemberCount > 0 {
		return BindResult{}, ErrShellInUse
	}
	if err := store.TransferIdentity(ctx, input.OldUserID, shellUserID, ""); err != nil {
		return BindResult{}, err
	}
	if usage.ShellPersonalTenantID > 0 && input.CompanyOrgID != "" {
		if err := store.RemoveCompanyMembership(ctx, usage.ShellPersonalTenantID, input.CompanyOrgID); err != nil {
			return BindResult{}, err
		}
	}
	if err := store.DeactivateUser(ctx, shellUserID); err != nil {
		return BindResult{}, err
	}
	return BindResult{OldUserID: input.OldUserID}, nil
}
