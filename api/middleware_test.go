package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"ttl-cli/db"
)

func setupMultiTenantTest(t *testing.T) (*db.UserStore, *db.TenantStorageManager, func()) {
	t.Helper()
	tmpDir := t.TempDir()

	store := db.NewUserStore(filepath.Join(tmpDir, "users.json"))
	if err := store.Load(); err != nil {
		t.Fatal(err)
	}

	tenantMgr := db.NewTenantStorageManager(filepath.Join(tmpDir, "tenants"))

	cleanup := func() {
		_ = tenantMgr.CloseAll()
	}
	return store, tenantMgr, cleanup
}

func TestMultiTenantAuth_ValidKey(t *testing.T) {
	store, tenantMgr, cleanup := setupMultiTenantTest(t)
	defer cleanup()

	user, _ := store.AddUser("alice", "Alice")

	var capturedUserID string
	var capturedStorage db.Storage

	handler := MultiTenantAuthMiddleware(store, tenantMgr, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUserID = GetUserID(r)
		capturedStorage = GetStorage(r)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+user.APIKey)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if capturedUserID != "alice" {
		t.Errorf("expected user_id=alice, got %s", capturedUserID)
	}
	if capturedStorage == nil {
		t.Error("expected non-nil storage in context")
	}
}

func TestMultiTenantAuth_InvalidKey(t *testing.T) {
	store, tenantMgr, cleanup := setupMultiTenantTest(t)
	defer cleanup()

	handler := MultiTenantAuthMiddleware(store, tenantMgr, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer invalid-key")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
	var resp Response
	_ = json.NewDecoder(w.Body).Decode(&resp)
	if resp.Code != 1 {
		t.Errorf("expected code=1, got %d", resp.Code)
	}
}

func TestMultiTenantAuth_DisabledUser(t *testing.T) {
	store, tenantMgr, cleanup := setupMultiTenantTest(t)
	defer cleanup()

	user, _ := store.AddUser("alice", "Alice")
	_ = store.SetActive("alice", false)

	handler := MultiTenantAuthMiddleware(store, tenantMgr, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+user.APIKey)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestMultiTenantAuth_NoHeader(t *testing.T) {
	store, tenantMgr, cleanup := setupMultiTenantTest(t)
	defer cleanup()

	handler := MultiTenantAuthMiddleware(store, tenantMgr, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}
