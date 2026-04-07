package conf

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateWorkspaceName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"Valid simple", "work", true},
		{"Valid with numbers", "work1", true},
		{"Valid with underscore", "work_space", true},
		{"Valid with hyphen", "work-space", true},
		{"Valid complex", "work-space_123", true},
		{"Empty string", "", false},
		{"Invalid with space", "work space", false},
		{"Invalid with dot", "work.space", false},
		{"Invalid with special", "work@space", false},
		{"Invalid with Chinese", "工作空间", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateWorkspaceName(tt.input)
			if result != tt.expected {
				t.Errorf("ValidateWorkspaceName(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestWorkspaceCRUD(t *testing.T) {
	tmpDir := t.TempDir()
	confFile := filepath.Join(tmpDir, "test.ini")

	t.Run("Create workspace", func(t *testing.T) {
		dbPath, err := CreateWorkspace(confFile, "test-workspace")
		if err != nil {
			t.Fatalf("CreateWorkspace failed: %v", err)
		}
		if dbPath == "" {
			t.Error("Expected non-empty dbPath")
		}

		_, err = os.Stat(dbPath)
		if err == nil {
			t.Error("Expected database file to not exist yet")
		}

		content, err := os.ReadFile(confFile)
		if err != nil {
			t.Fatalf("Failed to read config file: %v", err)
		}
		contentStr := string(content)
		if !contains(contentStr, "[workspaces.test-workspace]") {
			t.Error("Expected workspace section in config file")
		}
		if !contains(contentStr, "db_path") {
			t.Error("Expected db_path in config file")
		}
	})

	t.Run("Create duplicate workspace", func(t *testing.T) {
		_, err := CreateWorkspace(confFile, "test-workspace")
		if err == nil {
			t.Error("Expected error when creating duplicate workspace")
		}
	})

	t.Run("Create invalid workspace name", func(t *testing.T) {
		_, err := CreateWorkspace(confFile, "invalid name")
		if err == nil {
			t.Error("Expected error for invalid workspace name")
		}
	})

	t.Run("List workspaces", func(t *testing.T) {
		names, current, err := ListWorkspaces(confFile)
		if err != nil {
			t.Fatalf("ListWorkspaces failed: %v", err)
		}
		if len(names) != 1 {
			t.Errorf("Expected 1 workspace, got %d", len(names))
		}
		if names[0] != "test-workspace" {
			t.Errorf("Expected workspace name 'test-workspace', got '%s'", names[0])
		}
		if current != "" && current != "default" {
			t.Errorf("Expected current workspace to be empty or 'default', got '%s'", current)
		}
	})

	t.Run("Create second workspace", func(t *testing.T) {
		_, err := CreateWorkspace(confFile, "test-workspace-2")
		if err != nil {
			t.Fatalf("CreateWorkspace failed: %v", err)
		}

		names, _, err := ListWorkspaces(confFile)
		if err != nil {
			t.Fatalf("ListWorkspaces failed: %v", err)
		}
		if len(names) != 2 {
			t.Errorf("Expected 2 workspaces, got %d", len(names))
		}
	})

	t.Run("Switch workspace", func(t *testing.T) {
		err := SwitchWorkspace(confFile, "test-workspace")
		if err != nil {
			t.Fatalf("SwitchWorkspace failed: %v", err)
		}

		current, err := GetCurrentWorkspace(confFile)
		if err != nil {
			t.Fatalf("GetCurrentWorkspace failed: %v", err)
		}
		if current != "test-workspace" {
			t.Errorf("Expected current workspace 'test-workspace', got '%s'", current)
		}
	})

	t.Run("Switch to non-existent workspace", func(t *testing.T) {
		err := SwitchWorkspace(confFile, "non-existent")
		if err == nil {
			t.Error("Expected error when switching to non-existent workspace")
		}
	})

	t.Run("Get workspace info", func(t *testing.T) {
		dbPath, storageType, count, err := GetWorkspaceInfo(confFile, "test-workspace")
		if err != nil {
			t.Fatalf("GetWorkspaceInfo failed: %v", err)
		}
		if dbPath == "" {
			t.Error("Expected non-empty dbPath")
		}
		if storageType != "sqlite" {
			t.Errorf("Expected storage type 'sqlite', got '%s'", storageType)
		}
		if count != 0 {
			t.Errorf("Expected count 0 (db doesn't exist), got %d", count)
		}
	})

	t.Run("Get info for non-existent workspace", func(t *testing.T) {
		_, _, _, err := GetWorkspaceInfo(confFile, "non-existent")
		if err == nil {
			t.Error("Expected error when getting info for non-existent workspace")
		}
	})

	t.Run("Delete workspace", func(t *testing.T) {
		_, err := CreateWorkspace(confFile, "to-delete")
		if err != nil {
			t.Fatalf("CreateWorkspace failed: %v", err)
		}

		err = DeleteWorkspace(confFile, "to-delete")
		if err != nil {
			t.Fatalf("DeleteWorkspace failed: %v", err)
		}

		names, _, err := ListWorkspaces(confFile)
		if err != nil {
			t.Fatalf("ListWorkspaces failed: %v", err)
		}
		if len(names) != 2 {
			t.Errorf("Expected 2 workspaces after deletion, got %d", len(names))
		}
	})

	t.Run("Delete non-existent workspace", func(t *testing.T) {
		err := DeleteWorkspace(confFile, "non-existent")
		if err == nil {
			t.Error("Expected error when deleting non-existent workspace")
		}
	})
}

func TestMigrateToWorkspaces(t *testing.T) {
	tmpDir := t.TempDir()
	confFile := filepath.Join(tmpDir, "test.ini")

	t.Run("Migrate fresh config", func(t *testing.T) {
		err := MigrateToWorkspaces(confFile)
		if err != nil {
			t.Fatalf("MigrateToWorkspaces failed: %v", err)
		}

		content, err := os.ReadFile(confFile)
		if err != nil {
			t.Fatalf("Failed to read config file: %v", err)
		}
		contentStr := string(content)
		if !contains(contentStr, "[workspaces.default]") {
			t.Error("Expected default workspace section after migration")
		}
	})

	t.Run("Migrate existing config", func(t *testing.T) {
		homeDir := t.TempDir()
		oldDbPath := filepath.Join(homeDir, "mydata.db")
		configContent := "[storage]\ntype = sqlite\npath = " + oldDbPath + "\n"
		if err := os.WriteFile(confFile, []byte(configContent), 0644); err != nil {
			t.Fatalf("Failed to write config: %v", err)
		}

		err := MigrateToWorkspaces(confFile)
		if err != nil {
			t.Fatalf("MigrateToWorkspaces failed: %v", err)
		}

		content, err := os.ReadFile(confFile)
		if err != nil {
			t.Fatalf("Failed to read config file: %v", err)
		}
		contentStr := string(content)
		if !contains(contentStr, "[workspaces.default]") {
			t.Error("Expected default workspace section")
		}
		if !contains(contentStr, oldDbPath) {
			t.Error("Expected original db path to be preserved")
		}
	})
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
