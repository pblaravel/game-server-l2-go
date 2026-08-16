package loginserver

import (
	"context"
	"sync"
	"time"
)

// AccountInfo matches Java DBAccountInfo / ACCOUNTS table.
type AccountInfo struct {
	Login       string
	PassHash    string
	AccessLevel int
	LastServer  int
	LastIP      string
	LastActive  int64
}

// GameServerRow matches Java DBGameServer / gameservers table.
type GameServerRow struct {
	ServerID int
	HexID    string
	Host     string
}

// AccountStore is the login-server account DAO (Java AccountInfoRepository).
type AccountStore interface {
	GetAccountInfo(ctx context.Context, login string) (*AccountInfo, error)
	CreateAccount(ctx context.Context, info AccountInfo) error
	UpdateAccount(ctx context.Context, info AccountInfo) error
	UpdateAccountLastServer(ctx context.Context, account string, serverID int) error
}

// GameServerStore is the login-server gameserver DAO (Java GameServerRepository).
type GameServerStore interface {
	GetAllGameServers(ctx context.Context) ([]GameServerRow, error)
	AddGameServer(ctx context.Context, gs GameServerRow) error
}

// MemoryAccountStore is used by tests and when PostgreSQL is not configured.
type MemoryAccountStore struct {
	mu   sync.RWMutex
	data map[string]AccountInfo
}

func NewMemoryAccountStore() *MemoryAccountStore {
	return &MemoryAccountStore{data: make(map[string]AccountInfo)}
}

func (s *MemoryAccountStore) GetAccountInfo(_ context.Context, login string) (*AccountInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	info, ok := s.data[login]
	if !ok {
		return nil, nil
	}
	cp := info
	return &cp, nil
}

func (s *MemoryAccountStore) CreateAccount(_ context.Context, info AccountInfo) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if info.LastActive == 0 {
		info.LastActive = time.Now().UnixMilli()
	}
	s.data[info.Login] = info
	return nil
}

func (s *MemoryAccountStore) UpdateAccount(_ context.Context, info AccountInfo) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[info.Login] = info
	return nil
}

func (s *MemoryAccountStore) UpdateAccountLastServer(_ context.Context, account string, serverID int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	info, ok := s.data[account]
	if !ok {
		return nil
	}
	info.LastServer = serverID
	s.data[account] = info
	return nil
}

type MemoryGameServerStore struct {
	mu   sync.RWMutex
	data map[int]GameServerRow
}

func NewMemoryGameServerStore() *MemoryGameServerStore {
	return &MemoryGameServerStore{data: make(map[int]GameServerRow)}
}

func (s *MemoryGameServerStore) GetAllGameServers(_ context.Context) ([]GameServerRow, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]GameServerRow, 0, len(s.data))
	for _, v := range s.data {
		out = append(out, v)
	}
	return out, nil
}

func (s *MemoryGameServerStore) AddGameServer(_ context.Context, gs GameServerRow) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[gs.ServerID] = gs
	return nil
}
