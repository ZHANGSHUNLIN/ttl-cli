package integration_test

import (
	"os"
	"testing"

	"ttl-cli/db"
	"ttl-cli/models"
)

var (
	testKey1 = models.ValJsonKey{Key: "test-resource-1", Type: models.ORIGIN}
	testVal1 = models.ValJson{Val: "value1", Tag: []string{"work", "dev"}}
	testKey2 = models.ValJsonKey{Key: "test-resource-2", Type: models.ORIGIN}
	testVal2 = models.ValJson{Val: "value2", Tag: []string{"work", "ci"}}
	testKey3 = models.ValJsonKey{Key: "test-resource-3", Type: models.ORIGIN}
	testVal3 = models.ValJson{Val: "value3", Tag: []string{"deploy"}}
)

func TestTagsList(t *testing.T) {
	cleanup := setupTempStorage(t)
	defer cleanup()

	_ = db.SaveResource(testKey1, testVal1)
	_ = db.SaveResource(testKey2, testVal2)
	_ = db.SaveResource(testKey3, testVal3)

	stats, err := db.GetTagStats()
	if err != nil {
		t.Fatalf("GetTagStats failed: %v", err)
	}

	if len(stats) != 4 {
		t.Errorf("Expected 4 tags (work, dev, ci, deploy), got %d", len(stats))
	}

	tagMap := make(map[string]int)
	for _, stat := range stats {
		tagMap[stat.Tag] = stat.Count
	}

	if tagMap["work"] != 2 {
		t.Errorf("Expected work tag count 2, got %d", tagMap["work"])
	}
	if tagMap["dev"] != 1 {
		t.Errorf("Expected dev tag count 1, got %d", tagMap["dev"])
	}
	if tagMap["ci"] != 1 {
		t.Errorf("Expected ci tag count 1, got %d", tagMap["ci"])
	}
	if tagMap["deploy"] != 1 {
		t.Errorf("Expected deploy tag count 1, got %d", tagMap["deploy"])
	}
}

func TestTagsEmpty(t *testing.T) {
	cleanup := setupTempStorage(t)
	defer cleanup()

	stats, err := db.GetTagStats()
	if err != nil {
		t.Fatalf("GetTagStats failed: %v", err)
	}

	if len(stats) != 0 {
		t.Errorf("Expected 0 tags, got %d", len(stats))
	}
}

func TestTagsSortOrder(t *testing.T) {
	cleanup := setupTempStorage(t)
	defer cleanup()

	_ = db.SaveResource(models.ValJsonKey{Key: "z-test", Type: models.ORIGIN}, models.ValJson{Val: "value", Tag: []string{"zebra"}})
	_ = db.SaveResource(models.ValJsonKey{Key: "a-test", Type: models.ORIGIN}, models.ValJson{Val: "value", Tag: []string{"alpha"}})

	stats, err := db.GetTagStats()
	if err != nil {
		t.Fatalf("GetTagStats failed: %v", err)
	}

	if len(stats) != 2 {
		t.Fatalf("Expected 2 tags, got %d", len(stats))
	}

	if stats[0].Tag != "alpha" {
		t.Errorf("First tag should be 'alpha', got '%s'", stats[0].Tag)
	}
	if stats[1].Tag != "zebra" {
		t.Errorf("Second tag should be 'zebra', got '%s'", stats[1].Tag)
	}
}

func TestTagsWithMultipleResources(t *testing.T) {
	cleanup := setupTempStorage(t)
	defer cleanup()

	for i := 0; i < 5; i++ {
		key := models.ValJsonKey{Key: "res-" + string(rune('a'+i)), Type: models.ORIGIN}
		val := models.ValJson{Val: "value", Tag: []string{"common"}}
		_ = db.SaveResource(key, val)
	}

	stats, err := db.GetTagStats()
	if err != nil {
		t.Fatalf("GetTagStats failed: %v", err)
	}

	if len(stats) != 1 {
		t.Errorf("Expected 1 tag, got %d", len(stats))
	}

	if stats[0].Tag != "common" {
		t.Errorf("Expected tag 'common', got '%s'", stats[0].Tag)
	}

	if stats[0].Count != 5 {
		t.Errorf("Expected 5 resources, got %d", stats[0].Count)
	}
}

func TestTagsWithNoTagResources(t *testing.T) {
	cleanup := setupTempStorage(t)
	defer cleanup()

	_ = db.SaveResource(models.ValJsonKey{Key: "no-tag-1", Type: models.ORIGIN}, models.ValJson{Val: "value", Tag: []string{}})
	_ = db.SaveResource(models.ValJsonKey{Key: "no-tag-2", Type: models.ORIGIN}, models.ValJson{Val: "value", Tag: []string{}})

	stats, err := db.GetTagStats()
	if err != nil {
		t.Fatalf("GetTagStats failed: %v", err)
	}

	if len(stats) != 0 {
		t.Errorf("Expected 0 tags, got %d", len(stats))
	}
}

func TestTagTypesExcluded(t *testing.T) {
	cleanup := setupTempStorage(t)
	defer cleanup()

	_ = db.SaveResource(testKey1, testVal1)
	_ = db.SaveResource(models.ValJsonKey{Key: "tag-type", Type: models.TAG, OriginKey: testKey1.Key}, models.ValJson{Val: "value"})

	stats, err := db.GetTagStats()
	if err != nil {
		t.Fatalf("GetTagStats failed: %v", err)
	}

	if len(stats) != 2 {
		t.Errorf("Expected 2 tags (TAG type resources should be excluded, testVal1 has 'work' and 'dev'), got %d", len(stats))
	}
}

func testKey(prefix string, i int) models.ValJsonKey {
	return models.ValJsonKey{
		Key:  prefix + "-" + string(rune('a'+i)),
		Type: models.ORIGIN,
	}
}

func TestGetTagStats_FileStorageOnly(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ttl-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	testConf := tmpDir + "/test.conf"
	testDB := tmpDir + "/test.bbolt"

	confContent := "db_path = " + testDB + "\nstorage_type = bbolt\n"
	if err := os.WriteFile(testConf, []byte(confContent), 0644); err != nil {
		t.Fatal(err)
	}

	ls := db.NewLocalStorage()
	ls.SetDBPath(testDB)
	if err := ls.Init(); err != nil {
		t.Fatal(err)
	}
	defer ls.Close()

	key1 := models.ValJsonKey{Key: "file-test-1", Type: models.ORIGIN}
	val1 := models.ValJson{Val: "value1", Tag: []string{"file-tag"}}
	if err := ls.SaveResource(key1, val1); err != nil {
		t.Fatal(err)
	}

	stats, err := ls.GetTagStats()
	if err != nil {
		t.Fatalf("GetTagStats failed: %v", err)
	}

	if len(stats) != 1 {
		t.Errorf("Expected 1 tag, got %d", len(stats))
	}

	if stats[0].Tag != "file-tag" {
		t.Errorf("Expected tag 'file-tag', got '%s'", stats[0].Tag)
	}

	if stats[0].Count != 1 {
		t.Errorf("Expected 1 resource, got %d", stats[0].Count)
	}
}
