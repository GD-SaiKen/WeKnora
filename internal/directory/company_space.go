package directory

import "context"

// OrgMemberStore joins personal tenants to organizations.
type OrgMemberStore interface {
	HasMember(ctx context.Context, orgID string, tenantID uint64) (bool, error)
	MemberRole(ctx context.Context, orgID string, tenantID uint64) (string, error)
	AddMember(ctx context.Context, orgID string, tenantID uint64, representativeUserID string, role string) error
	RemoveMember(ctx context.Context, orgID string, tenantID uint64) error
}

const OrgRoleEditor = "editor"

// EnsureMember adds the tenant as editor when orgID is set and not already a member.
func EnsureMember(ctx context.Context, store OrgMemberStore, orgID string, tenantID uint64, userID string) error {
	if orgID == "" || tenantID == 0 {
		return nil
	}
	ok, err := store.HasMember(ctx, orgID, tenantID)
	if err != nil {
		return err
	}
	if ok {
		return nil
	}
	return store.AddMember(ctx, orgID, tenantID, userID, OrgRoleEditor)
}

// RemoveMember drops only this organization membership.
func RemoveMember(ctx context.Context, store OrgMemberStore, orgID string, tenantID uint64) error {
	if orgID == "" || tenantID == 0 {
		return nil
	}
	return store.RemoveMember(ctx, orgID, tenantID)
}
