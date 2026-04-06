package db

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type TenantStorageManager struct {
	dataDir  string
	storages map[string]*LocalStorage
	mu       sync.RWMutex
}

func NewTenantStorageManager(dataDir string) *TenantStorageManager {
	return &TenantStorageManager{
		dataDir:  dataDir,
		storages: make(map[string]*LocalStorage),
	}
}

func (m *TenantStorageManager) GetStorage(userID string) (Storage, error) {
	m.mu.RLock()
	if s, ok := m.storages[userID]; ok {
		m.mu.RUnlock()
		return s, nil
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()

	if s, ok := m.storages[userID]; ok {
		return s, nil
	}

	userDir := filepath.Join(m.dataDir, userID)
	if err := os.MkdirAll(userDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create user data directory: %w", err)
	}

	dbPath := filepath.Join(userDir, "data.db")
	ls := NewLocalStorage()
	ls.SetDBPath(dbPath)

	if err := ls.Init(); err != nil {
		return nil, fmt.Errorf("failed to initialize database for user %s: %w", userID, err)
	}

	m.storages[userID] = ls
	return ls, nil
}

func (m *TenantStorageManager) CloseAll() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var lastErr error
	for id, s := range m.storages {
		if err := s.Close(); err != nil {
			lastErr = fmt.Errorf("failed to close storage for user %s: %w", id, err)
		}
		delete(m.storages, id)
	}
	return lastErr
}

func (m *TenantStorageManager) RemoveStorage(userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if s, ok := m.storages[userID]; ok {
		_ = s.Close()
		delete(m.storages, userID)
	}

	userDir := filepath.Join(m.dataDir, userID)
	return os.RemoveAll(userDir)
}
