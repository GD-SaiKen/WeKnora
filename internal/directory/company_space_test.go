package directory

import (
	"context"
	"testing"
)

type memOrg struct {
	role map[uint64]string
}

func newMemOrg() *memOrg {
	return &memOrg{role: map[uint64]string{}}
}

func (m *memOrg) HasMember(_ context.Context, _ string, tenantID uint64) (bool, error) {
	_, ok := m.role[tenantID]
	return ok, nil
}

func (m *memOrg) MemberRole(_ context.Context, _ string, tenantID uint64) (string, error) {
	return m.role[tenantID], nil
}

func (m *memOrg) AddMember(_ context.Context, _ string, tenantID uint64, _ string, role string) error {
	m.role[tenantID] = role
	return nil
}

func (m *memOrg) RemoveMember(_ context.Context, _ string, tenantID uint64) error {
	delete(m.role, tenantID)
	return nil
}

func TestEnsureCompanyMemberAddsEditorWithoutOverwriting(t *testing.T) {
	store := newMemOrg()
	if err := EnsureMember(context.Background(), store, "org-1", 7, "user-1"); err != nil {
		t.Fatal(err)
	}
	if store.role[7] != OrgRoleEditor {
		t.Fatalf("role %s", store.role[7])
	}
	store.role[7] = "viewer"
	if err := EnsureMember(context.Background(), store, "org-1", 7, "user-1"); err != nil {
		t.Fatal(err)
	}
	if store.role[7] != "viewer" {
		t.Fatal("existing role was overwritten")
	}
}

func TestEnsureMemberNoOpWhenOrgEmpty(t *testing.T) {
	store := newMemOrg()
	if err := EnsureMember(context.Background(), store, "", 7, "u"); err != nil {
		t.Fatal(err)
	}
	if len(store.role) != 0 {
		t.Fatal("wrote member")
	}
}
