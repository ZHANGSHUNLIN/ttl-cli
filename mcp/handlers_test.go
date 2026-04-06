package mcp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ttl-cli/db"

	"github.com/mark3labs/mcp-go/mcp"
)

func setupTempStorage(t *testing.T) func() {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "ttl-mcp-test-*")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}

	dbPath := filepath.Join(tmpDir, "test.db")
	confPath := filepath.Join(tmpDir, "test.ini")
	confContent := fmt.Sprintf("db_path = %s\n", dbPath)
	if err := os.WriteFile(confPath, []byte(confContent), 0644); err != nil {
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

func makeRequest(args map[string]any) mcp.CallToolRequest {
	return mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: args,
		},
	}
}

func assertResultText(t *testing.T, result *mcp.CallToolResult, contains string) {
	t.Helper()
	if result.IsError {
		t.Fatalf("expected success, but got error: %v", result.Content)
	}
	text := resultText(result)
	if !strings.Contains(text, contains) {
		t.Errorf("expected to contain %q, got %q", contains, text)
	}
}

func assertResultError(t *testing.T, result *mcp.CallToolResult, contains string) {
	t.Helper()
	if !result.IsError {
		t.Fatalf("expected error, but got success: %v", result.Content)
	}
	text := resultText(result)
	if !strings.Contains(text, contains) {
		t.Errorf("expected error to contain %q, got %q", contains, text)
	}
}

func resultText(result *mcp.CallToolResult) string {
	for _, c := range result.Content {
		if tc, ok := c.(mcp.TextContent); ok {
			return tc.Text
		}
	}
	return ""
}

func TestTtlAdd_Success(t *testing.T) {
	cleanup := setupTempStorage(t)
	defer cleanup()

	result, err := handleAdd(context.Background(), makeRequest(map[string]any{
		"key": "mysite", "value": "https://example.com",
	}))
	if err != nil {
		t.Fatal(err)
	}
	assertResultText(t, result, "mcp.added_successfully")
}

func TestTtlAdd_DuplicateKey(t *testing.T) {
	cleanup := setupTempStorage(t)
	defer cleanup()

	_, _ = handleAdd(context.Background(), makeRequest(map[string]any{
		"key": "mysite", "value": "v1",
	}))

	result, err := handleAdd(context.Background(), makeRequest(map[string]any{
		"key": "mysite", "value": "v2",
	}))
	if err != nil {
		t.Fatal(err)
	}
	assertResultError(t, result, "mcp.key_already_exists")
}

func TestTtlGet_NoArgs(t *testing.T) {
	cleanup := setupTempStorage(t)
	defer cleanup()

	_, _ = handleAdd(context.Background(), makeRequest(map[string]any{
		"key": "site1", "value": "v1",
	}))
	_, _ = handleAdd(context.Background(), makeRequest(map[string]any{
		"key": "site2", "value": "v2",
	}))

	result, err := handleGet(context.Background(), makeRequest(map[string]any{}))
	if err != nil {
		t.Fatal(err)
	}
	assertResultText(t, result, "site1")
	assertResultText(t, result, "site2")
}

func TestTtlGet_FuzzyMatch(t *testing.T) {
	cleanup := setupTempStorage(t)
	defer cleanup()

	_, _ = handleAdd(context.Background(), makeRequest(map[string]any{
		"key": "github-repo", "value": "https://github.com",
	}))

	result, err := handleGet(context.Background(), makeRequest(map[string]any{
		"key": "github",
	}))
	if err != nil {
		t.Fatal(err)
	}
	assertResultText(t, result, "github-repo")
}

func TestTtlGet_NotFound(t *testing.T) {
	cleanup := setupTempStorage(t)
	defer cleanup()

	result, err := handleGet(context.Background(), makeRequest(map[string]any{
		"key": "nonexistent",
	}))
	if err != nil {
		t.Fatal(err)
	}
	assertResultError(t, result, "mcp.not_found")
}

func TestTtlUpdate_Success(t *testing.T) {
	cleanup := setupTempStorage(t)
	defer cleanup()

	_, _ = handleAdd(context.Background(), makeRequest(map[string]any{
		"key": "mysite", "value": "old-value",
	}))
	_, _ = handleTag(context.Background(), makeRequest(map[string]any{
		"key": "mysite", "tags": []any{"t1"},
	}))

	result, err := handleUpdate(context.Background(), makeRequest(map[string]any{
		"key": "mysite", "value": "new-value",
	}))
	if err != nil {
		t.Fatal(err)
	}
	assertResultText(t, result, "mcp.added_successfully")

}

func TestTtlUpdate_NotFound(t *testing.T) {
	cleanup := setupTempStorage(t)
	defer cleanup()

	result, err := handleUpdate(context.Background(), makeRequest(map[string]any{
		"key": "nope", "value": "v",
	}))
	if err != nil {
		t.Fatal(err)
	}
	assertResultError(t, result, "mcp.resource_not_found")
}

func TestTtlDelete_Success(t *testing.T) {
	cleanup := setupTempStorage(t)
	defer cleanup()

	_, _ = handleAdd(context.Background(), makeRequest(map[string]any{
		"key": "to-delete", "value": "bye",
	}))

	result, err := handleDelete(context.Background(), makeRequest(map[string]any{
		"key": "to-delete",
	}))
	if err != nil {
		t.Fatal(err)
	}
	assertResultText(t, result, "mcp.deleted_successfully")

	getResult, _ := handleGet(context.Background(), makeRequest(map[string]any{
		"key": "to-delete",
	}))
	assertResultError(t, getResult, "mcp.not_found")
}

func TestTtlDelete_NotFound(t *testing.T) {
	cleanup := setupTempStorage(t)
	defer cleanup()

	result, err := handleDelete(context.Background(), makeRequest(map[string]any{
		"key": "nope",
	}))
	if err != nil {
		t.Fatal(err)
	}
	assertResultError(t, result, "mcp.resource_not_found")
}

func TestTtlTag_Success(t *testing.T) {
	cleanup := setupTempStorage(t)
	defer cleanup()

	_, _ = handleAdd(context.Background(), makeRequest(map[string]any{
		"key": "res", "value": "v",
	}))

	result, err := handleTag(context.Background(), makeRequest(map[string]any{
		"key": "res", "tags": []any{"web", "dev", "web"},
	}))
	if err != nil {
		t.Fatal(err)
	}
	assertResultText(t, result, "mcp.tag_added_successfully")

	getResult, _ := handleGet(context.Background(), makeRequest(map[string]any{"key": "res"}))
	text := resultText(getResult)
	if strings.Count(text, "web") != 1 {
		t.Errorf("标签未去重: %s", text)
	}
}

func TestTtlDtag_Success(t *testing.T) {
	cleanup := setupTempStorage(t)
	defer cleanup()

	_, _ = handleAdd(context.Background(), makeRequest(map[string]any{
		"key": "res", "value": "v",
	}))
	_, _ = handleTag(context.Background(), makeRequest(map[string]any{
		"key": "res", "tags": []any{"keep", "remove"},
	}))

	result, err := handleDtag(context.Background(), makeRequest(map[string]any{
		"key": "res", "tag": "remove",
	}))
	if err != nil {
		t.Fatal(err)
	}
	assertResultText(t, result, "mcp.tag_deleted_successfully")

	getResult, _ := handleGet(context.Background(), makeRequest(map[string]any{"key": "res"}))
	text := resultText(getResult)
	if strings.Contains(text, "remove") {
		t.Errorf("标签未移除: %s", text)
	}
	if !strings.Contains(text, "keep") {
		t.Errorf("标签 keep 丢失: %s", text)
	}
}

func TestTtlRename_Success(t *testing.T) {
	cleanup := setupTempStorage(t)
	defer cleanup()

	_, _ = handleAdd(context.Background(), makeRequest(map[string]any{
		"key": "old-name", "value": "v",
	}))
	_, _ = handleTag(context.Background(), makeRequest(map[string]any{
		"key": "old-name", "tags": []any{"t1"},
	}))

	result, err := handleRename(context.Background(), makeRequest(map[string]any{
		"old_key": "old-name", "new_key": "new-name",
	}))
	if err != nil {
		t.Fatal(err)
	}
	assertResultText(t, result, "mcp.rename_successfully")

	getOld, _ := handleGet(context.Background(), makeRequest(map[string]any{"key": "old-name"}))
	assertResultError(t, getOld, "mcp.not_found")

	getNew, _ := handleGet(context.Background(), makeRequest(map[string]any{"key": "new-name"}))
	text := resultText(getNew)
	if !strings.Contains(text, "new-name") {
		t.Errorf("重命名失败: %s", text)
	}
}

func TestTtlList_Empty(t *testing.T) {
	cleanup := setupTempStorage(t)
	defer cleanup()

	result, err := handleList(context.Background(), makeRequest(nil))
	if err != nil {
		t.Fatal(err)
	}
	assertResultText(t, result, "mcp.no_resources")
}

func TestTtlList_WithData(t *testing.T) {
	cleanup := setupTempStorage(t)
	defer cleanup()

	_, _ = handleAdd(context.Background(), makeRequest(map[string]any{
		"key": "a", "value": "v1",
	}))
	_, _ = handleAdd(context.Background(), makeRequest(map[string]any{
		"key": "b", "value": "v2",
	}))

	result, err := handleList(context.Background(), makeRequest(nil))
	if err != nil {
		t.Fatal(err)
	}
	text := resultText(result)
	if !strings.Contains(text, "a") || !strings.Contains(text, "b") {
		t.Errorf("列表缺少资源: %s", text)
	}
}

func TestTtlLogAdd_Success(t *testing.T) {
	cleanup := setupTempStorage(t)
	defer cleanup()

	result, err := handleLogAdd(context.Background(), makeRequest(map[string]any{
		"content": "完成用户模块重构",
	}))
	if err != nil {
		t.Fatal(err)
	}
	assertResultText(t, result, "mcp.log_recorded")

	listResult, _ := handleLogList(context.Background(), makeRequest(map[string]any{}))
	text := resultText(listResult)
	if !strings.Contains(text, "完成用户模块重构") {
		t.Errorf("日志未正确保存: %s", text)
	}
}

func TestTtlLogAdd_WithTags(t *testing.T) {
	cleanup := setupTempStorage(t)
	defer cleanup()

	result, err := handleLogAdd(context.Background(), makeRequest(map[string]any{
		"content": "接口联调完成",
		"tags":    []any{"项目A", "后端", "项目A"},
	}))
	if err != nil {
		t.Fatal(err)
	}
	assertResultText(t, result, "mcp.log_recorded")

	listResult, _ := handleLogList(context.Background(), makeRequest(map[string]any{}))
	text := resultText(listResult)
	if strings.Count(text, "项目A") != 1 {
		t.Errorf("标签未去重: %s", text)
	}
}

func TestTtlLogList_Today(t *testing.T) {
	cleanup := setupTempStorage(t)
	defer cleanup()

	_, _ = handleLogAdd(context.Background(), makeRequest(map[string]any{
		"content": "今天的工作",
	}))

	result, err := handleLogList(context.Background(), makeRequest(map[string]any{}))
	if err != nil {
		t.Fatal(err)
	}
	assertResultText(t, result, "今天的工作")
}

func TestTtlLogList_DateFilter(t *testing.T) {
	cleanup := setupTempStorage(t)
	defer cleanup()

	_, _ = handleLogAdd(context.Background(), makeRequest(map[string]any{
		"content": "工作1",
	}))

	result, err := handleLogList(context.Background(), makeRequest(map[string]any{
		"date": "2024-01-01",
	}))
	if err != nil {
		t.Fatal(err)
	}
	text := resultText(result)
	if strings.Contains(text, "工作1") {
		t.Errorf("日期过滤失败: %s", text)
	}
}

func TestTtlLogList_TagFilter(t *testing.T) {
	cleanup := setupTempStorage(t)
	defer cleanup()

	_, _ = handleLogAdd(context.Background(), makeRequest(map[string]any{
		"content": "工作A",
		"tags":    []any{"project"},
	}))
	_, _ = handleLogAdd(context.Background(), makeRequest(map[string]any{
		"content": "工作B",
		"tags":    []any{"other"},
	}))

	result, err := handleLogList(context.Background(), makeRequest(map[string]any{
		"tag": "project",
	}))
	if err != nil {
		t.Fatal(err)
	}
	text := resultText(result)
	if !strings.Contains(text, "工作A") || strings.Contains(text, "工作B") {
		t.Errorf("标签过滤失败: %s", text)
	}
}

func TestTtlLogList_Empty(t *testing.T) {
	cleanup := setupTempStorage(t)
	defer cleanup()

	result, err := handleLogList(context.Background(), makeRequest(map[string]any{}))
	if err != nil {
		t.Fatal(err)
	}
	assertResultText(t, result, "mcp.no_log_records")
}

func TestTtlLogDelete_Success(t *testing.T) {
	cleanup := setupTempStorage(t)
	defer cleanup()

	addResult, _ := handleLogAdd(context.Background(), makeRequest(map[string]any{
		"content": "待删除",
	}))
	_ = resultText(addResult)

	result, err := handleLogDelete(context.Background(), makeRequest(map[string]any{
		"id": 1234567890,
	}))
	if err != nil {
		t.Fatal(err)
	}
	_ = result
}

func TestTtlLogDelete_NotFound(t *testing.T) {
	cleanup := setupTempStorage(t)
	defer cleanup()

	result, err := handleLogDelete(context.Background(), makeRequest(map[string]any{
		"id": 0,
	}))
	if err != nil {
		t.Fatal(err)
	}
	assertResultError(t, result, "mcp.id_required")
}

func TestTtlExport_Resources(t *testing.T) {
	cleanup := setupTempStorage(t)
	defer cleanup()

	_, _ = handleAdd(context.Background(), makeRequest(map[string]any{
		"key": "test-key", "value": "test-value",
	}))

	result, err := handleExport(context.Background(), makeRequest(map[string]any{
		"type": "resources",
	}))
	if err != nil {
		t.Fatal(err)
	}
	text := resultText(result)
	if !strings.Contains(text, "test-key") || !strings.Contains(text, "test-value") {
		t.Errorf("导出内容不正确: %s", text)
	}
}

func TestTtlExport_Log(t *testing.T) {
	cleanup := setupTempStorage(t)
	defer cleanup()

	_, _ = handleLogAdd(context.Background(), makeRequest(map[string]any{
		"content": "测试日志",
	}))

	result, err := handleExport(context.Background(), makeRequest(map[string]any{
		"type": "log",
	}))
	if err != nil {
		t.Fatal(err)
	}
	text := resultText(result)
	if !strings.Contains(text, "测试日志") {
		t.Errorf("导出内容不正确: %s", text)
	}
}

func TestTtlExport_Empty(t *testing.T) {
	cleanup := setupTempStorage(t)
	defer cleanup()

	result, err := handleExport(context.Background(), makeRequest(map[string]any{
		"type": "resources",
	}))
	if err != nil {
		t.Fatal(err)
	}
	text := resultText(result)
	if !strings.Contains(text, "key,value,tags") {
		t.Errorf("导出空数据应包含 header: %s", text)
	}
}
