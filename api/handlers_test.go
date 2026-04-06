package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"ttl-cli/db"
	"ttl-cli/models"
)

func setupTempStorage(t *testing.T) func() {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "ttl-api-test-*")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	dbPath := filepath.Join(tmpDir, "test.db")
	confPath := filepath.Join(tmpDir, "test.ini")
	if err := os.WriteFile(confPath, []byte(fmt.Sprintf("db_path = %s\n", dbPath)), 0644); err != nil {
		_ = os.RemoveAll(tmpDir)
		t.Fatalf("写入临时配置文件失败: %v", err)
	}
	if err := db.InitDB("local", "", "", 0, confPath); err != nil {
		_ = os.RemoveAll(tmpDir)
		t.Fatalf("初始化临时存储失败: %v", err)
	}
	return func() {
		_ = db.CloseDB()
		_ = os.RemoveAll(tmpDir)
	}
}

// helper: 向 ResourcesHandler 发请求
func doResources(t *testing.T, method, target string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var bodyReader io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, target, bodyReader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	ResourcesHandler(w, req)
	return w
}

// helper: 向 ResourceHandler 发请求（带 path 参数）
func doResource(t *testing.T, method, target string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var bodyReader io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, target, bodyReader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	ResourceHandler(w, req)
	return w
}

func parseResponse(t *testing.T, w *httptest.ResponseRecorder) Response {
	t.Helper()
	var resp Response
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("解析响应失败: %v, body: %s", err, w.Body.String())
	}
	return resp
}

// ==================== GET /api/v1/resources ====================

func TestGetResources_Empty(t *testing.T) {
	cleanup := setupTempStorage(t)
	defer cleanup()

	w := doResources(t, http.MethodGet, "/api/v1/resources", nil)
	resp := parseResponse(t, w)
	if resp.Code != 0 {
		t.Fatalf("期望 code=0, 实际: %d, msg: %s", resp.Code, resp.Message)
	}
	data, _ := json.Marshal(resp.Data)
	if string(data) != "[]" {
		t.Errorf("期望空数组，实际: %s", data)
	}
}

func TestGetResources_WithData(t *testing.T) {
	cleanup := setupTempStorage(t)
	defer cleanup()

	// 直接通过 db 插入数据
	_ = db.SaveResource(models.ValJsonKey{Key: "site", Type: models.ORIGIN}, models.ValJson{Val: "v1", Tag: []string{}})

	w := doResources(t, http.MethodGet, "/api/v1/resources", nil)
	resp := parseResponse(t, w)
	if resp.Code != 0 {
		t.Fatalf("code=%d, msg=%s", resp.Code, resp.Message)
	}
	data, _ := json.Marshal(resp.Data)
	if !bytes.Contains(data, []byte("site")) {
		t.Errorf("响应中缺少 site: %s", data)
	}
}

func TestGetResources_Search(t *testing.T) {
	cleanup := setupTempStorage(t)
	defer cleanup()

	_ = db.SaveResource(models.ValJsonKey{Key: "github", Type: models.ORIGIN}, models.ValJson{Val: "https://github.com", Tag: []string{}})
	_ = db.SaveResource(models.ValJsonKey{Key: "gitlab", Type: models.ORIGIN}, models.ValJson{Val: "https://gitlab.com", Tag: []string{}})

	w := doResources(t, http.MethodGet, "/api/v1/resources?q=github", nil)
	resp := parseResponse(t, w)
	data, _ := json.Marshal(resp.Data)
	if !bytes.Contains(data, []byte("github")) {
		t.Errorf("搜索结果应包含 github: %s", data)
	}
	if bytes.Contains(data, []byte("gitlab")) {
		t.Errorf("搜索结果不应包含 gitlab: %s", data)
	}
}

// ==================== POST /api/v1/resources ====================

func TestPostResource_Success(t *testing.T) {
	cleanup := setupTempStorage(t)
	defer cleanup()

	w := doResources(t, http.MethodPost, "/api/v1/resources", CreateResourceRequest{Key: "mykey", Value: "myval"})
	if w.Code != http.StatusCreated {
		t.Fatalf("期望 201, 实际: %d", w.Code)
	}
	resp := parseResponse(t, w)
	if resp.Code != 0 {
		t.Fatalf("code=%d, msg=%s", resp.Code, resp.Message)
	}
}

func TestPostResource_Duplicate(t *testing.T) {
	cleanup := setupTempStorage(t)
	defer cleanup()

	doResources(t, http.MethodPost, "/api/v1/resources", CreateResourceRequest{Key: "dup", Value: "v1"})
	w := doResources(t, http.MethodPost, "/api/v1/resources", CreateResourceRequest{Key: "dup", Value: "v2"})
	resp := parseResponse(t, w)
	if resp.Code != 1 {
		t.Fatalf("期望 code=1(业务错误), 实际: %d", resp.Code)
	}
}

// ==================== PUT /api/v1/resources/{key} ====================

func TestPutResource_Success(t *testing.T) {
	cleanup := setupTempStorage(t)
	defer cleanup()

	_ = db.SaveResource(models.ValJsonKey{Key: "k1", Type: models.ORIGIN}, models.ValJson{Val: "old", Tag: []string{"t1"}})

	w := doResource(t, http.MethodPut, "/api/v1/resources/k1", UpdateResourceRequest{Value: "new"})
	resp := parseResponse(t, w)
	if resp.Code != 0 {
		t.Fatalf("code=%d, msg=%s", resp.Code, resp.Message)
	}
	// 验证 tag 保留
	data, _ := json.Marshal(resp.Data)
	if !bytes.Contains(data, []byte("t1")) {
		t.Errorf("tag 丢失: %s", data)
	}
}

func TestPutResource_NotFound(t *testing.T) {
	cleanup := setupTempStorage(t)
	defer cleanup()

	w := doResource(t, http.MethodPut, "/api/v1/resources/nope", UpdateResourceRequest{Value: "v"})
	if w.Code != http.StatusNotFound {
		t.Fatalf("期望 404, 实际: %d", w.Code)
	}
}

// ==================== DELETE /api/v1/resources/{key} ====================

func TestDeleteResource_Success(t *testing.T) {
	cleanup := setupTempStorage(t)
	defer cleanup()

	_ = db.SaveResource(models.ValJsonKey{Key: "del-me", Type: models.ORIGIN}, models.ValJson{Val: "bye", Tag: []string{}})

	w := doResource(t, http.MethodDelete, "/api/v1/resources/del-me", nil)
	resp := parseResponse(t, w)
	if resp.Code != 0 {
		t.Fatalf("code=%d, msg=%s", resp.Code, resp.Message)
	}

	// 验证已删除
	resources, _ := db.GetAllResources()
	vjk := models.ValJsonKey{Key: "del-me", Type: models.ORIGIN}
	if _, exists := resources[vjk]; exists {
		t.Error("资源未被删除")
	}
}

// ==================== POST /api/v1/resources/{key}/tags ====================

func TestPostTags_Success(t *testing.T) {
	cleanup := setupTempStorage(t)
	defer cleanup()

	_ = db.SaveResource(models.ValJsonKey{Key: "res", Type: models.ORIGIN}, models.ValJson{Val: "v", Tag: []string{"existing"}})

	w := doResource(t, http.MethodPost, "/api/v1/resources/res/tags", AddTagsRequest{Tags: []string{"new1", "existing"}})
	resp := parseResponse(t, w)
	if resp.Code != 0 {
		t.Fatalf("code=%d, msg=%s", resp.Code, resp.Message)
	}
	data, _ := json.Marshal(resp.Data)
	// existing 不应重复
	if bytes.Count(data, []byte("existing")) != 1 {
		t.Errorf("标签未去重: %s", data)
	}
}

// ==================== DELETE /api/v1/resources/{key}/tags/{tag} ====================

func TestDeleteTag_Success(t *testing.T) {
	cleanup := setupTempStorage(t)
	defer cleanup()

	_ = db.SaveResource(models.ValJsonKey{Key: "res", Type: models.ORIGIN}, models.ValJson{Val: "v", Tag: []string{"keep", "remove"}})

	w := doResource(t, http.MethodDelete, "/api/v1/resources/res/tags/remove", nil)
	resp := parseResponse(t, w)
	if resp.Code != 0 {
		t.Fatalf("code=%d, msg=%s", resp.Code, resp.Message)
	}
	data, _ := json.Marshal(resp.Data)
	if bytes.Contains(data, []byte("remove")) {
		t.Errorf("标签未移除: %s", data)
	}
}

// ==================== POST /api/v1/resources/{key}/rename ====================

func TestRenameResource_Success(t *testing.T) {
	cleanup := setupTempStorage(t)
	defer cleanup()

	_ = db.SaveResource(models.ValJsonKey{Key: "old", Type: models.ORIGIN}, models.ValJson{Val: "v", Tag: []string{"t1"}})

	w := doResource(t, http.MethodPost, "/api/v1/resources/old/rename", RenameRequest{NewKey: "new"})
	resp := parseResponse(t, w)
	if resp.Code != 0 {
		t.Fatalf("code=%d, msg=%s", resp.Code, resp.Message)
	}

	// 验证旧 key 消失
	resources, _ := db.GetAllResources()
	if _, exists := resources[models.ValJsonKey{Key: "old", Type: models.ORIGIN}]; exists {
		t.Error("旧 key 未删除")
	}
	if _, exists := resources[models.ValJsonKey{Key: "new", Type: models.ORIGIN}]; !exists {
		t.Error("新 key 不存在")
	}
}

// ==================== Auth Middleware ====================

func TestAuthMiddleware_NoKey(t *testing.T) {
	handler := AuthMiddleware("secret", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("期望 401, 实际: %d", w.Code)
	}
}

func TestAuthMiddleware_WrongKey(t *testing.T) {
	handler := AuthMiddleware("secret", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer wrong")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("期望 401, 实际: %d", w.Code)
	}
}

func TestAuthMiddleware_Correct(t *testing.T) {
	handler := AuthMiddleware("secret", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer secret")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200, 实际: %d", w.Code)
	}
}

func TestAuthMiddleware_Disabled(t *testing.T) {
	handler := AuthMiddleware("", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200 (无鉴权), 实际: %d", w.Code)
	}
}
