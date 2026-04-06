package integration_test

import (
	"bytes"
	"encoding/csv"
	"strings"
	"testing"
	"time"

	"ttl-cli/command"
	"ttl-cli/db"
	"ttl-cli/models"
)

func parseCSV(t *testing.T, data []byte) [][]string {
	t.Helper()
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})
	r := csv.NewReader(strings.NewReader(string(data)))
	rows, err := r.ReadAll()
	if err != nil {
		t.Fatalf("解析 CSV 失败: %v", err)
	}
	return rows
}

func TestExportResourcesCSV(t *testing.T) {
	cleanup := setupTempStorage(t)
	defer cleanup()

	_ = db.SaveResource(
		models.ValJsonKey{Key: "github", Type: models.ORIGIN},
		models.ValJson{Val: "https://github.com", Tag: []string{"dev", "work"}},
	)
	_ = db.SaveResource(
		models.ValJsonKey{Key: "note", Type: models.ORIGIN},
		models.ValJson{Val: "hello world", Tag: []string{}},
	)

	resources, err := db.GetAllResources()
	if err != nil {
		t.Fatalf("GetAllResources() 失败: %v", err)
	}

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	count, err := command.WriteResourcesCSV(w, resources)
	w.Flush()
	if err != nil {
		t.Fatalf("WriteResourcesCSV() 失败: %v", err)
	}
	if count != 2 {
		t.Errorf("记录数 = %d, want 2", count)
	}

	rows := parseCSV(t, buf.Bytes())
	if len(rows) != 3 {
		t.Fatalf("行数 = %d, want 3", len(rows))
	}

	if rows[0][0] != "key" || rows[0][1] != "value" || rows[0][2] != "tags" {
		t.Errorf("header 不符: %v", rows[0])
	}

	dataByKey := map[string][]string{}
	for _, row := range rows[1:] {
		dataByKey[row[0]] = row
	}

	githubRow, ok := dataByKey["github"]
	if !ok {
		t.Fatal("未找到 github 行")
	}
	if githubRow[1] != "https://github.com" {
		t.Errorf("github value = %q, want %q", githubRow[1], "https://github.com")
	}
	for _, tag := range []string{"dev", "work"} {
		if !strings.Contains(githubRow[2], tag) {
			t.Errorf("github tags 未包含 %q: got %q", tag, githubRow[2])
		}
	}

	noteRow, ok := dataByKey["note"]
	if !ok {
		t.Fatal("未找到 note 行")
	}
	if noteRow[2] != "" {
		t.Errorf("note tags = %q, want empty", noteRow[2])
	}
}

func TestExportEmptyResources(t *testing.T) {
	cleanup := setupTempStorage(t)
	defer cleanup()

	resources, _ := db.GetAllResources()
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	count, err := command.WriteResourcesCSV(w, resources)
	w.Flush()
	if err != nil {
		t.Fatalf("WriteResourcesCSV() 失败: %v", err)
	}
	if count != 0 {
		t.Errorf("空库记录数 = %d, want 0", count)
	}

	rows := parseCSV(t, buf.Bytes())
	if len(rows) != 1 {
		t.Fatalf("行数 = %d, want 1（只有 header）", len(rows))
	}
	if rows[0][0] != "key" {
		t.Errorf("header[0] = %q, want key", rows[0][0])
	}
}

func TestExportAuditCSV(t *testing.T) {
	cleanup := setupTempStorage(t)
	defer cleanup()

	_ = db.RecordAudit("mykey", "add")
	_ = db.RecordAudit("mykey", "get")

	records, err := db.GetAllAuditRecords()
	if err != nil {
		t.Fatalf("GetAllAuditRecords() 失败: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("审计记录数 = %d, want 2", len(records))
	}

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	count, err := command.WriteAuditCSV(w, records)
	w.Flush()
	if err != nil {
		t.Fatalf("WriteAuditCSV() 失败: %v", err)
	}
	if count != 2 {
		t.Errorf("导出记录数 = %d, want 2", count)
	}

	rows := parseCSV(t, buf.Bytes())
	if len(rows) != 3 {
		t.Fatalf("行数 = %d, want 3", len(rows))
	}
	if rows[0][0] != "resource_key" || rows[0][1] != "operation" {
		t.Errorf("audit header 不符: %v", rows[0])
	}
	for _, row := range rows[1:] {
		if row[0] != "mykey" {
			t.Errorf("resource_key = %q, want mykey", row[0])
		}
	}
}

func TestExportHistoryCSV(t *testing.T) {
	cleanup := setupTempStorage(t)
	defer cleanup()

	_ = db.RecordCommandHistory("add", "keyA", false)
	time.Sleep(time.Millisecond)
	_ = db.RecordCommandHistory("get", "keyB", false)

	histRecords, err := db.GetAllHistoryRecords()
	if err != nil {
		t.Fatalf("GetAllHistoryRecords() 失败: %v", err)
	}

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	count, err := command.WriteHistoryCSV(w, histRecords)
	w.Flush()
	if err != nil {
		t.Fatalf("WriteHistoryCSV() 失败: %v", err)
	}
	if count != 2 {
		t.Errorf("导出记录数 = %d, want 2", count)
	}

	rows := parseCSV(t, buf.Bytes())
	if len(rows) != 3 {
		t.Fatalf("行数 = %d, want 3", len(rows))
	}
	if rows[0][0] != "id" || rows[0][2] != "operation" {
		t.Errorf("history header 不符: %v", rows[0])
	}
}

func TestExportBOM(t *testing.T) {
	cleanup := setupTempStorage(t)
	defer cleanup()

	resources, _ := db.GetAllResources()
	var buf bytes.Buffer

	buf.Write([]byte{0xEF, 0xBB, 0xBF})
	w := csv.NewWriter(&buf)
	_, err := command.WriteResourcesCSV(w, resources)
	w.Flush()
	if err != nil {
		t.Fatalf("WriteResourcesCSV() 失败: %v", err)
	}

	b := buf.Bytes()
	if len(b) < 3 || b[0] != 0xEF || b[1] != 0xBB || b[2] != 0xBF {
		t.Error("BOM 前缀不正确")
	}
	rows := parseCSV(t, b)
	if len(rows) < 1 || rows[0][0] != "key" {
		t.Errorf("BOM 后 CSV 格式不正确: %v", rows)
	}
}
