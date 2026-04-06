package db

import (
	"os"
	"path/filepath"
	"testing"
	"ttl-cli/models"
)

// ==================== UserStore ====================

func TestUserStore_AddUser(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewUserStore(filepath.Join(tmpDir, "users.json"))
	if err := store.Load(); err != nil {
		t.Fatal(err)
	}

	user, err := store.AddUser("alice", "Alice")
	if err != nil {
		t.Fatal(err)
	}
	if user.ID != "alice" || user.Name != "Alice" || !user.Active {
		t.Errorf("unexpected user: %+v", user)
	}
	if len(user.APIKey) != 64 {
		t.Errorf("API Key length should be 64, got %d", len(user.APIKey))
	}
}

func TestUserStore_AddUser_DuplicateID(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewUserStore(filepath.Join(tmpDir, "users.json"))
	_ = store.Load()

	_, _ = store.AddUser("alice", "Alice")
	_, err := store.AddUser("alice", "Alice2")
	if err == nil {
		t.Fatal("expected error for duplicate ID")
	}
}

func TestUserStore_AddUser_InvalidID(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewUserStore(filepath.Join(tmpDir, "users.json"))
	_ = store.Load()

	cases := []string{"Alice", "a b", "a!b", "verylongidthatexceedsthirtytwocharacterslimit"}
	for _, id := range cases {
		_, err := store.AddUser(id, "name")
		if err == nil {
			t.Errorf("expected error for invalid ID %q", id)
		}
	}
}

func TestUserStore_FindByAPIKey(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewUserStore(filepath.Join(tmpDir, "users.json"))
	_ = store.Load()

	user, _ := store.AddUser("alice", "Alice")

	found := store.FindByAPIKey(user.APIKey)
	if found == nil {
		t.Fatal("expected to find user by API key")
	}
	if found.ID != "alice" {
		t.Errorf("expected alice, got %s", found.ID)
	}
}

func TestUserStore_FindByAPIKey_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewUserStore(filepath.Join(tmpDir, "users.json"))
	_ = store.Load()

	found := store.FindByAPIKey("nonexistent")
	if found != nil {
		t.Fatal("expected nil for nonexistent key")
	}
}

func TestUserStore_FindByID(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewUserStore(filepath.Join(tmpDir, "users.json"))
	_ = store.Load()

	_, _ = store.AddUser("bob", "Bob")

	found := store.FindByID("bob")
	if found == nil || found.ID != "bob" {
		t.Errorf("unexpected: %v", found)
	}

	notFound := store.FindByID("nope")
	if notFound != nil {
		t.Error("expected nil")
	}
}

func TestUserStore_ListUsers(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewUserStore(filepath.Join(tmpDir, "users.json"))
	_ = store.Load()

	_, _ = store.AddUser("alice", "Alice")
	_, _ = store.AddUser("bob", "Bob")

	users := store.ListUsers()
	if len(users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(users))
	}
}

func TestUserStore_SetActive(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewUserStore(filepath.Join(tmpDir, "users.json"))
	_ = store.Load()

	_, _ = store.AddUser("alice", "Alice")

	if err := store.SetActive("alice", false); err != nil {
		t.Fatal(err)
	}
	u := store.FindByID("alice")
	if u.Active {
		t.Error("expected inactive")
	}

	if err := store.SetActive("alice", true); err != nil {
		t.Fatal(err)
	}
	u = store.FindByID("alice")
	if !u.Active {
		t.Error("expected active")
	}
}

func TestUserStore_ResetKey(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewUserStore(filepath.Join(tmpDir, "users.json"))
	_ = store.Load()

	user, _ := store.AddUser("alice", "Alice")
	oldKey := user.APIKey

	newKey, err := store.ResetKey("alice")
	if err != nil {
		t.Fatal(err)
	}
	if newKey == oldKey {
		t.Error("new key should differ from old key")
	}

	// 旧 key 失效
	if found := store.FindByAPIKey(oldKey); found != nil {
		t.Error("old key should be invalid")
	}
	// 新 key 有效
	if found := store.FindByAPIKey(newKey); found == nil {
		t.Error("new key should be valid")
	}
}

func TestUserStore_DeleteUser(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewUserStore(filepath.Join(tmpDir, "users.json"))
	_ = store.Load()

	_, _ = store.AddUser("alice", "Alice")

	if err := store.DeleteUser("alice"); err != nil {
		t.Fatal(err)
	}
	if found := store.FindByID("alice"); found != nil {
		t.Error("expected user to be deleted")
	}
}

func TestUserStore_DeleteUser_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewUserStore(filepath.Join(tmpDir, "users.json"))
	_ = store.Load()

	if err := store.DeleteUser("nope"); err == nil {
		t.Error("expected error for nonexistent user")
	}
}

func TestUserStore_Persistence(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "users.json")

	// 第一次：创建并保存
	store1 := NewUserStore(filePath)
	_ = store1.Load()
	_, _ = store1.AddUser("alice", "Alice")

	// 第二次：重新加载
	store2 := NewUserStore(filePath)
	if err := store2.Load(); err != nil {
		t.Fatal(err)
	}
	users := store2.ListUsers()
	if len(users) != 1 || users[0].ID != "alice" {
		t.Errorf("persistence failed: %+v", users)
	}
}

// ==================== TenantStorageManager ====================

func TestTenantManager_GetStorage(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewTenantStorageManager(tmpDir)

	s1, err := mgr.GetStorage("alice")
	if err != nil {
		t.Fatal(err)
	}
	if s1 == nil {
		t.Fatal("expected non-nil storage")
	}

	// 再次获取应复用同一实例
	s2, err := mgr.GetStorage("alice")
	if err != nil {
		t.Fatal(err)
	}
	if s1 != s2 {
		t.Error("expected same storage instance")
	}

	_ = mgr.CloseAll()
}

func TestTenantManager_Isolation(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewTenantStorageManager(tmpDir)
	defer mgr.CloseAll()

	sAlice, _ := mgr.GetStorage("alice")
	sBob, _ := mgr.GetStorage("bob")

	// Alice 写入数据
	key := models.ValJsonKey{Key: "isolation-test", Type: models.ORIGIN}
	val := models.ValJson{Val: "alice-data", Tag: []string{}}
	if err := sAlice.SaveResource(key, val); err != nil {
		t.Fatal(err)
	}

	// Bob 看不到 Alice 的数据
	bobResources, err := sBob.GetAllResources()
	if err != nil {
		t.Fatal(err)
	}
	if _, exists := bobResources[key]; exists {
		t.Error("Bob should not see Alice's data")
	}

	// Alice 能看到自己的数据
	aliceResources, err := sAlice.GetAllResources()
	if err != nil {
		t.Fatal(err)
	}
	if _, exists := aliceResources[key]; !exists {
		t.Error("Alice should see her own data")
	}
}

func TestTenantManager_RemoveStorage(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewTenantStorageManager(tmpDir)

	_, _ = mgr.GetStorage("alice")

	if err := mgr.RemoveStorage("alice"); err != nil {
		t.Fatal(err)
	}

	// 目录应被删除
	userDir := filepath.Join(tmpDir, "alice")
	if _, err := os.Stat(userDir); !os.IsNotExist(err) {
		t.Error("expected user dir to be removed")
	}
}

func TestTenantManager_CloseAll(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewTenantStorageManager(tmpDir)

	_, _ = mgr.GetStorage("alice")
	_, _ = mgr.GetStorage("bob")

	if err := mgr.CloseAll(); err != nil {
		t.Fatal(err)
	}
}
