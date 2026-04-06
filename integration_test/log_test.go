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

func TestLogLifecycle(t *testing.T) {
	cleanup := setupTempStorage(t)
	defer cleanup()

	now := time.Now()

	record1 := models.LogRecord{
		ID:        now.UnixNano(),
		Content:   "完成用户模块重构",
		Tags:      []string{"项目A"},
		CreatedAt: now.Format("2006-01-02 15:04:05"),
		Date:      now.Format("2006-01-02"),
	}
	if err := db.SaveLogRecord(record1); err != nil {
		t.Fatalf("SaveLogRecord(1) 失败: %v", err)
	}

	time.Sleep(time.Millisecond)
	now2 := time.Now()
	record2 := models.LogRecord{
		ID:        now2.UnixNano(),
		Content:   "修复登录接口 bug",
		Tags:      []string{"项目B", "bugfix"},
		CreatedAt: now2.Format("2006-01-02 15:04:05"),
		Date:      now2.Format("2006-01-02"),
	}
	if err := db.SaveLogRecord(record2); err != nil {
		t.Fatalf("SaveLogRecord(2) 失败: %v", err)
	}

	today := now.Format("2006-01-02")
	records, err := db.GetLogRecords(today, today)
	if err != nil {
		t.Fatalf("GetLogRecords() 失败: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("日志数量 = %d, want 2", len(records))
	}
	if records[0].ID < records[1].ID {
		t.Error("日志未按 ID 倒序排列")
	}

	if err := db.DeleteLogRecord(record1.ID); err != nil {
		t.Fatalf("DeleteLogRecord() 失败: %v", err)
	}

	records, err = db.GetLogRecords(today, today)
	if err != nil {
		t.Fatalf("删除后 GetLogRecords() 失败: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("删除后日志数量 = %d, want 1", len(records))
	}
	if records[0].Content != "修复登录接口 bug" {
		t.Errorf("剩余日志内容 = %q, want %q", records[0].Content, "修复登录接口 bug")
	}
}

func TestLogWithTags(t *testing.T) {
	cleanup := setupTempStorage(t)
	defer cleanup()

	now := time.Now()
	record := models.LogRecord{
		ID:        now.UnixNano(),
		Content:   "需求评审",
		Tags:      []string{"项目A", "会议", "需求"},
		CreatedAt: now.Format("2006-01-02 15:04:05"),
		Date:      now.Format("2006-01-02"),
	}
	if err := db.SaveLogRecord(record); err != nil {
		t.Fatalf("SaveLogRecord() 失败: %v", err)
	}

	records, err := db.GetLogRecords(record.Date, record.Date)
	if err != nil {
		t.Fatalf("GetLogRecords() 失败: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("日志数量 = %d, want 1", len(records))
	}
	if len(records[0].Tags) != 3 {
		t.Errorf("标签数量 = %d, want 3", len(records[0].Tags))
	}
	tagSet := map[string]bool{}
	for _, tag := range records[0].Tags {
		tagSet[tag] = true
	}
	if !tagSet["项目A"] || !tagSet["会议"] || !tagSet["需求"] {
		t.Errorf("标签内容不符: got %v", records[0].Tags)
	}
}

func TestLogDateRangeFilter(t *testing.T) {
	cleanup := setupTempStorage(t)
	defer cleanup()

	dates := []string{"2026-04-01", "2026-04-02", "2026-04-03"}
	for i, date := range dates {
		record := models.LogRecord{
			ID:        time.Now().UnixNano() + int64(i),
			Content:   "工作内容 " + date,
			Tags:      nil,
			CreatedAt: date + " 10:00:00",
			Date:      date,
		}
		if err := db.SaveLogRecord(record); err != nil {
			t.Fatalf("SaveLogRecord(%s) 失败: %v", date, err)
		}
	}

	all, err := db.GetLogRecords("", "")
	if err != nil {
		t.Fatalf("GetLogRecords('','') 失败: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("全量日志数量 = %d, want 3", len(all))
	}

	day, err := db.GetLogRecords("2026-04-02", "2026-04-02")
	if err != nil {
		t.Fatalf("GetLogRecords(单日) 失败: %v", err)
	}
	if len(day) != 1 {
		t.Errorf("单日日志数量 = %d, want 1", len(day))
	}
	if day[0].Date != "2026-04-02" {
		t.Errorf("单日日志日期 = %q, want 2026-04-02", day[0].Date)
	}

	rangeRecords, err := db.GetLogRecords("2026-04-01", "2026-04-02")
	if err != nil {
		t.Fatalf("GetLogRecords(范围) 失败: %v", err)
	}
	if len(rangeRecords) != 2 {
		t.Errorf("范围日志数量 = %d, want 2", len(rangeRecords))
	}

	empty, err := db.GetLogRecords("2026-05-01", "2026-05-31")
	if err != nil {
		t.Fatalf("GetLogRecords(空范围) 失败: %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("空范围日志数量 = %d, want 0", len(empty))
	}
}

func TestLogDeleteNotFound(t *testing.T) {
	cleanup := setupTempStorage(t)
	defer cleanup()

	err := db.DeleteLogRecord(999999)
	if err == nil {
		t.Error("删除不存在的日志应返回错误")
	}
	if !strings.Contains(err.Error(), "未找到该日志记录") {
		t.Errorf("错误信息不符: got %q", err.Error())
	}
}

func TestLogExportCSV(t *testing.T) {
	cleanup := setupTempStorage(t)
	defer cleanup()

	record := models.LogRecord{
		ID:        1234567890,
		Content:   "完成接口联调",
		Tags:      []string{"项目A", "后端"},
		CreatedAt: "2026-04-04 14:30:00",
		Date:      "2026-04-04",
	}
	if err := db.SaveLogRecord(record); err != nil {
		t.Fatalf("SaveLogRecord() 失败: %v", err)
	}

	records, err := db.GetLogRecords("", "")
	if err != nil {
		t.Fatalf("GetLogRecords() 失败: %v", err)
	}

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	count, err := command.WriteLogCSV(w, records)
	if err != nil {
		t.Fatalf("WriteLogCSV() 失败: %v", err)
	}
	w.Flush()

	if count != 1 {
		t.Errorf("导出记录数 = %d, want 1", count)
	}

	r := csv.NewReader(&buf)
	rows, err := r.ReadAll()
	if err != nil {
		t.Fatalf("CSV 解析失败: %v", err)
	}

	if len(rows) != 2 {
		t.Fatalf("CSV 行数 = %d, want 2", len(rows))
	}

	expectedHeader := []string{"id", "content", "tags", "created_at", "date"}
	for i, h := range expectedHeader {
		if rows[0][i] != h {
			t.Errorf("header[%d] = %q, want %q", i, rows[0][i], h)
		}
	}

	row := rows[1]
	if row[0] != "1234567890" {
		t.Errorf("id = %q, want 1234567890", row[0])
	}
	if row[1] != "完成接口联调" {
		t.Errorf("content = %q, want 完成接口联调", row[1])
	}
	if row[2] != "项目A|后端" {
		t.Errorf("tags = %q, want 项目A|后端", row[2])
	}
	if row[3] != "2026-04-04 14:30:00" {
		t.Errorf("created_at = %q, want 2026-04-04 14:30:00", row[3])
	}
	if row[4] != "2026-04-04" {
		t.Errorf("date = %q, want 2026-04-04", row[4])
	}
}
