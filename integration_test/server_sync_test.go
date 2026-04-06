// Package integration_test 包含 server + cloud storage + sync 的端到端集成测试。
package integration_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"ttl-cli/api"
	"ttl-cli/db"
	"ttl-cli/models"
	ttlsync "ttl-cli/sync"
)

// setupServerWithLocalDB 启动一个 httptest.Server 作为远程后端，
// 后端使用独立的临时 bbolt 数据库。返回 server URL 和清理函数。
func setupServerWithLocalDB(t *testing.T) (serverURL string, cleanup func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "ttl-server-*")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}

	dbPath := filepath.Join(tmpDir, "server.db")
	confPath := filepath.Join(tmpDir, "server.ini")
	confContent := fmt.Sprintf("db_path = %s\n", dbPath)
	if err := os.WriteFile(confPath, []byte(confContent), 0644); err != nil {
		_ = os.RemoveAll(tmpDir)
		t.Fatalf("写入配置文件失败: %v", err)
	}

	// 初始化全局存储（被 api handlers 使用）
	if err := db.InitDB("local", "", "", 0, confPath); err != nil {
		_ = os.RemoveAll(tmpDir)
		t.Fatalf("初始化 server 存储失败: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/resources", api.ResourcesHandler)
	mux.HandleFunc("/api/v1/resources/", api.ResourceHandler)
	mux.HandleFunc("/api/v1/audit/stats", api.AuditStatsHandler)
	mux.HandleFunc("/api/v1/history", api.HistoryHandler)

	srv := httptest.NewServer(mux)

	return srv.URL, func() {
		srv.Close()
		_ = db.CloseDB()
		_ = os.RemoveAll(tmpDir)
	}
}

// setupLocalDB 创建独立的本地存储实例（不影响全局 db.Stor）。
func setupLocalDB(t *testing.T) (storage *db.LocalStorage, cleanup func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "ttl-local-*")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}

	dbPath := filepath.Join(tmpDir, "local.db")
	ls := db.NewLocalStorage()
	ls.SetDBPath(dbPath)

	if err := ls.Init(); err != nil {
		_ = os.RemoveAll(tmpDir)
		t.Fatalf("初始化本地存储失败: %v", err)
	}

	return ls, func() {
		_ = ls.Close()
		_ = os.RemoveAll(tmpDir)
	}
}

// ==================== Server + CloudStorage 端到端测试 ====================

// TestServerAndCloudStorage_CRUD 通过真实 HTTP server 验证 CloudStorage 的完整 CRUD 流程
func TestServerAndCloudStorage_CRUD(t *testing.T) {
	serverURL, cleanup := setupServerWithLocalDB(t)
	defer cleanup()

	cs := db.NewCloudStorage(serverURL, "", 30)
	_ = cs.Init()
	defer cs.Close()

	key := models.ValJsonKey{Key: "server-test", Type: models.ORIGIN}
	val := models.ValJson{Val: "hello-server", Tag: []string{}}

	// 1. 初始为空
	resources, err := cs.GetAllResources()
	if err != nil {
		t.Fatalf("GetAllResources 失败: %v", err)
	}
	if len(resources) != 0 {
		t.Fatalf("初始资源应为空，实际: %d", len(resources))
	}

	// 2. 创建
	if err := cs.SaveResource(key, val); err != nil {
		t.Fatalf("SaveResource 失败: %v", err)
	}

	// 3. 查询
	resources, err = cs.GetAllResources()
	if err != nil {
		t.Fatalf("GetAllResources 失败: %v", err)
	}
	if len(resources) != 1 {
		t.Fatalf("期望 1 个资源，实际: %d", len(resources))
	}
	if resources[key].Val != "hello-server" {
		t.Errorf("期望 value=hello-server，实际: %s", resources[key].Val)
	}

	// 4. 更新
	if err := cs.UpdateResource(key, models.ValJson{Val: "updated", Tag: []string{}}); err != nil {
		t.Fatalf("UpdateResource 失败: %v", err)
	}
	resources, _ = cs.GetAllResources()
	if resources[key].Val != "updated" {
		t.Errorf("更新后期望 value=updated，实际: %s", resources[key].Val)
	}

	// 5. 删除
	if err := cs.DeleteResource(key); err != nil {
		t.Fatalf("DeleteResource 失败: %v", err)
	}
	resources, _ = cs.GetAllResources()
	if len(resources) != 0 {
		t.Errorf("删除后期望 0 个资源，实际: %d", len(resources))
	}
}

// TestServerAndCloudStorage_DuplicateKey 重复 key 应返回错误
func TestServerAndCloudStorage_DuplicateKey(t *testing.T) {
	serverURL, cleanup := setupServerWithLocalDB(t)
	defer cleanup()

	cs := db.NewCloudStorage(serverURL, "", 30)
	_ = cs.Init()
	defer cs.Close()

	key := models.ValJsonKey{Key: "dup-key", Type: models.ORIGIN}
	_ = cs.SaveResource(key, models.ValJson{Val: "v1", Tag: []string{}})

	err := cs.SaveResource(key, models.ValJson{Val: "v2", Tag: []string{}})
	if err == nil {
		t.Fatal("重复 key 保存应返回错误")
	}
}

// ==================== Sync Pull/Push 端到端测试 ====================

// TestSyncPullFlow 端到端 pull 流程：本地 ← 远程
func TestSyncPullFlow(t *testing.T) {
	serverURL, serverCleanup := setupServerWithLocalDB(t)
	defer serverCleanup()

	// 向远程写入数据
	cs := db.NewCloudStorage(serverURL, "", 30)
	_ = cs.Init()
	_ = cs.SaveResource(
		models.ValJsonKey{Key: "remote-a", Type: models.ORIGIN},
		models.ValJson{Val: "val-a", Tag: []string{}},
	)
	_ = cs.SaveResource(
		models.ValJsonKey{Key: "shared", Type: models.ORIGIN},
		models.ValJson{Val: "remote-ver", Tag: []string{}},
	)

	// 创建独立本地存储
	localStorage, localCleanup := setupLocalDB(t)
	defer localCleanup()

	// 向本地写入数据
	_ = localStorage.SaveResource(
		models.ValJsonKey{Key: "local-only", Type: models.ORIGIN},
		models.ValJson{Val: "local-val", Tag: []string{}},
	)
	_ = localStorage.SaveResource(
		models.ValJsonKey{Key: "shared", Type: models.ORIGIN},
		models.ValJson{Val: "local-ver", Tag: []string{}},
	)

	// 计算差异
	localRes, _ := localStorage.GetAllResources()
	remoteRes, _ := cs.GetAllResources()
	diff := ttlsync.ComputeDiff(localRes, remoteRes)

	// 验证差异
	if diff.InSync {
		t.Fatal("不应 InSync")
	}
	if len(diff.LocalOnly) != 1 {
		t.Errorf("期望 1 个 local_only，实际: %d", len(diff.LocalOnly))
	}
	if len(diff.RemoteOnly) != 1 {
		t.Errorf("期望 1 个 remote_only，实际: %d", len(diff.RemoteOnly))
	}
	if len(diff.Conflicts) != 1 {
		t.Errorf("期望 1 个 conflict，实际: %d", len(diff.Conflicts))
	}

	// 执行 pull
	if err := ttlsync.ExecutePull(diff, localStorage, cs, false); err != nil {
		t.Fatalf("ExecutePull 失败: %v", err)
	}

	// 验证 pull 结果：本地应与远程一致
	finalLocal, _ := localStorage.GetAllResources()

	// local-only 应被删除
	localOnlyKey := models.ValJsonKey{Key: "local-only", Type: models.ORIGIN}
	if _, exists := finalLocal[localOnlyKey]; exists {
		t.Error("local-only 应被删除")
	}

	// remote-a 应被新增
	remoteAKey := models.ValJsonKey{Key: "remote-a", Type: models.ORIGIN}
	if v, exists := finalLocal[remoteAKey]; !exists {
		t.Error("remote-a 应被新增到本地")
	} else if v.Val != "val-a" {
		t.Errorf("remote-a 值不正确: %s", v.Val)
	}

	// shared 应被远程覆盖
	sharedKey := models.ValJsonKey{Key: "shared", Type: models.ORIGIN}
	if finalLocal[sharedKey].Val != "remote-ver" {
		t.Errorf("shared 应为 remote-ver，实际: %s", finalLocal[sharedKey].Val)
	}

	if len(finalLocal) != 2 {
		t.Errorf("pull 后期望 2 个资源，实际: %d", len(finalLocal))
	}
}

// TestSyncPushFlow 端到端 push 流程：本地 → 远程
func TestSyncPushFlow(t *testing.T) {
	serverURL, serverCleanup := setupServerWithLocalDB(t)
	defer serverCleanup()

	// 向远程写入数据
	cs := db.NewCloudStorage(serverURL, "", 30)
	_ = cs.Init()
	_ = cs.SaveResource(
		models.ValJsonKey{Key: "remote-only", Type: models.ORIGIN},
		models.ValJson{Val: "remote-val", Tag: []string{}},
	)
	_ = cs.SaveResource(
		models.ValJsonKey{Key: "shared", Type: models.ORIGIN},
		models.ValJson{Val: "remote-ver", Tag: []string{}},
	)

	// 创建独立本地存储
	localStorage, localCleanup := setupLocalDB(t)
	defer localCleanup()

	// 向本地写入数据
	_ = localStorage.SaveResource(
		models.ValJsonKey{Key: "local-a", Type: models.ORIGIN},
		models.ValJson{Val: "val-a", Tag: []string{}},
	)
	_ = localStorage.SaveResource(
		models.ValJsonKey{Key: "shared", Type: models.ORIGIN},
		models.ValJson{Val: "local-ver", Tag: []string{}},
	)

	// 计算差异
	localRes, _ := localStorage.GetAllResources()
	remoteRes, _ := cs.GetAllResources()
	diff := ttlsync.ComputeDiff(localRes, remoteRes)

	// 执行 push
	if err := ttlsync.ExecutePush(diff, localStorage, cs, false); err != nil {
		t.Fatalf("ExecutePush 失败: %v", err)
	}

	// 验证 push 结果：远程应与本地一致
	finalRemote, _ := cs.GetAllResources()

	// remote-only 应被删除
	remoteOnlyKey := models.ValJsonKey{Key: "remote-only", Type: models.ORIGIN}
	if _, exists := finalRemote[remoteOnlyKey]; exists {
		t.Error("remote-only 应被删除")
	}

	// local-a 应被推送到远程
	localAKey := models.ValJsonKey{Key: "local-a", Type: models.ORIGIN}
	if v, exists := finalRemote[localAKey]; !exists {
		t.Error("local-a 应被推送到远程")
	} else if v.Val != "val-a" {
		t.Errorf("local-a 值不正确: %s", v.Val)
	}

	// shared 应被本地覆盖
	sharedKey := models.ValJsonKey{Key: "shared", Type: models.ORIGIN}
	if finalRemote[sharedKey].Val != "local-ver" {
		t.Errorf("shared 应为 local-ver，实际: %s", finalRemote[sharedKey].Val)
	}

	if len(finalRemote) != 2 {
		t.Errorf("push 后期望 2 个资源，实际: %d", len(finalRemote))
	}
}

// TestSyncDryRun_NoChanges 验证 dry-run 模式不会修改任何数据
func TestSyncDryRun_NoChanges(t *testing.T) {
	serverURL, serverCleanup := setupServerWithLocalDB(t)
	defer serverCleanup()

	cs := db.NewCloudStorage(serverURL, "", 30)
	_ = cs.Init()
	_ = cs.SaveResource(
		models.ValJsonKey{Key: "remote-res", Type: models.ORIGIN},
		models.ValJson{Val: "rv", Tag: []string{}},
	)

	localStorage, localCleanup := setupLocalDB(t)
	defer localCleanup()
	_ = localStorage.SaveResource(
		models.ValJsonKey{Key: "local-res", Type: models.ORIGIN},
		models.ValJson{Val: "lv", Tag: []string{}},
	)

	localRes, _ := localStorage.GetAllResources()
	remoteRes, _ := cs.GetAllResources()
	diff := ttlsync.ComputeDiff(localRes, remoteRes)

	// dry-run pull
	_ = ttlsync.ExecutePull(diff, localStorage, cs, true)

	// 验证本地没有变化
	finalLocal, _ := localStorage.GetAllResources()
	if len(finalLocal) != 1 {
		t.Errorf("dry-run 后本地应仍为 1 个资源，实际: %d", len(finalLocal))
	}
	localKey := models.ValJsonKey{Key: "local-res", Type: models.ORIGIN}
	if _, exists := finalLocal[localKey]; !exists {
		t.Error("dry-run 不应删除本地资源")
	}

	// 验证远程没有变化
	finalRemote, _ := cs.GetAllResources()
	if len(finalRemote) != 1 {
		t.Errorf("dry-run 后远程应仍为 1 个资源，实际: %d", len(finalRemote))
	}
}

// TestSyncAlreadyInSync 两端数据相同时应 InSync
func TestSyncAlreadyInSync(t *testing.T) {
	serverURL, serverCleanup := setupServerWithLocalDB(t)
	defer serverCleanup()

	cs := db.NewCloudStorage(serverURL, "", 30)
	_ = cs.Init()
	_ = cs.SaveResource(
		models.ValJsonKey{Key: "same", Type: models.ORIGIN},
		models.ValJson{Val: "value", Tag: []string{}},
	)

	localStorage, localCleanup := setupLocalDB(t)
	defer localCleanup()
	_ = localStorage.SaveResource(
		models.ValJsonKey{Key: "same", Type: models.ORIGIN},
		models.ValJson{Val: "value", Tag: []string{}},
	)

	localRes, _ := localStorage.GetAllResources()
	remoteRes, _ := cs.GetAllResources()
	diff := ttlsync.ComputeDiff(localRes, remoteRes)

	if !diff.InSync {
		t.Error("两端数据相同应 InSync")
	}
}
