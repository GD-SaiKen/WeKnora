package directory

import (
	"context"
	"fmt"
	"testing"
)

type memApplyStore struct {
	users      map[string]*UserSnapshot
	identities map[string]string // aiHubUserID -> userID
	subjects   map[string]string
	emails     map[string]string
	nextID     int
}

func newMemApplyStore() *memApplyStore {
	return &memApplyStore{
		users:      map[string]*UserSnapshot{},
		identities: map[string]string{},
		subjects:   map[string]string{},
		emails:     map[string]string{},
	}
}

func (m *memApplyStore) FindUserByNormalizedEmail(_ context.Context, email string) (*UserSnapshot, error) {
	id := m.emails[NormalizeEmail(email)]
	if id == "" {
		return nil, nil
	}
	u := m.users[id]
	cp := *u
	return &cp, nil
}

func (m *memApplyStore) FindUserBySubject(_ context.Context, subject string) (*UserSnapshot, error) {
	id := m.subjects[subject]
	if id == "" {
		return nil, nil
	}
	u := m.users[id]
	cp := *u
	return &cp, nil
}

func (m *memApplyStore) FindUserByAIHubUserID(_ context.Context, aiHubUserID string) (*UserSnapshot, error) {
	id := m.identities[aiHubUserID]
	if id == "" {
		return nil, nil
	}
	u := m.users[id]
	cp := *u
	return &cp, nil
}

func (m *memApplyStore) UserIDByNormalizedEmail(_ context.Context, email string) (string, error) {
	return m.emails[NormalizeEmail(email)], nil
}

func (m *memApplyStore) BindDirectoryIdentity(_ context.Context, userID string, entry Entry) error {
	m.identities[entry.UserID] = userID
	m.subjects[entry.Subject] = userID
	m.emails[NormalizeEmail(entry.Email)] = userID
	u := m.users[userID]
	u.HasDirectoryIdentity = true
	u.AIHubUserID = entry.UserID
	return nil
}

func (m *memApplyStore) MarkUserDisabled(_ context.Context, userID string) error {
	m.users[userID].PasswordLoginEnabled = false
	return nil
}

func (m *memApplyStore) CreateDirectoryUser(_ context.Context, entry Entry) (string, uint64, error) {
	m.nextID++
	id := fmt.Sprintf("u-%d", m.nextID)
	tenantID := uint64(100 + m.nextID)
	m.users[id] = &UserSnapshot{
		ID:                   id,
		Email:                NormalizeEmail(entry.Email),
		PersonalTenantID:     tenantID,
		PasswordLoginEnabled: false,
		HasDirectoryIdentity: true,
		AIHubUserID:          entry.UserID,
	}
	m.identities[entry.UserID] = id
	m.subjects[entry.Subject] = id
	m.emails[NormalizeEmail(entry.Email)] = id
	return id, tenantID, nil
}

func (m *memApplyStore) UpdateDirectoryIdentityFields(_ context.Context, userID string, entry Entry) error {
	u := m.users[userID]
	u.Email = NormalizeEmail(entry.Email)
	m.emails[NormalizeEmail(entry.Email)] = userID
	return nil
}

func TestApplyMatchesExistingEmailWithoutSecondUser(t *testing.T) {
	store := newMemApplyStore()
	store.users["u-old"] = &UserSnapshot{
		ID: "u-old", Email: "zhou@company.com", PersonalTenantID: 7,
	}
	store.emails["zhou@company.com"] = "u-old"

	res, err := Apply(context.Background(), store, Entry{
		UserID: "hub-1", Subject: "sub-1", Email: "Zhou@company.com",
		DisplayName: "周衡", Status: "active",
	})
	if err != nil || res.Action != ActionBound || len(store.users) != 1 {
		t.Fatalf("action=%s users=%d err=%v", res.Action, len(store.users), err)
	}
	if store.identities["hub-1"] != "u-old" {
		t.Fatal("identity not attached to old user")
	}
}

func TestApplyCreatesNewUser(t *testing.T) {
	store := newMemApplyStore()
	res, err := Apply(context.Background(), store, Entry{
		UserID: "hub-2", Subject: "sub-2", Email: "new@company.com", Status: "active",
	})
	if err != nil || res.Action != ActionCreated {
		t.Fatalf("res=%+v err=%v", res, err)
	}
	u := store.users[res.UserID]
	if u.PersonalTenantID == 0 || u.PasswordLoginEnabled {
		t.Fatalf("user=%+v", u)
	}
}
