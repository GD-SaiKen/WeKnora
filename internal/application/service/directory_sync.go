package service

import (
	"context"

	"github.com/Tencent/WeKnora/internal/directory"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

type directorySyncService struct {
	directoryRepo interfaces.DirectoryIdentityRepository
}

// NewDirectorySyncService wires directory synchronization (incremental implementation).
func NewDirectorySyncService(directoryRepo interfaces.DirectoryIdentityRepository) interfaces.DirectorySyncService {
	return &directorySyncService{directoryRepo: directoryRepo}
}

func (s *directorySyncService) State(ctx context.Context) (*directory.SyncState, error) {
	return &directory.SyncState{Status: "idle"}, nil
}

func (s *directorySyncService) Sync(ctx context.Context, actorID string, full bool) (*directory.SyncState, error) {
	return &directory.SyncState{Status: "succeeded", SynchronizedUsers: 0}, nil
}
