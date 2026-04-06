package sync

import (
	"fmt"
	"ttl-cli/db"
	"ttl-cli/models"
)

type DiffItem struct {
	Key       string          `json:"key"`
	LocalVal  *models.ValJson `json:"local_val,omitempty"`
	RemoteVal *models.ValJson `json:"remote_val,omitempty"`
	DiffType  DiffType        `json:"diff_type"`
}

type DiffType string

const (
	LocalOnly  DiffType = "local_only"
	RemoteOnly DiffType = "remote_only"
	Conflict   DiffType = "conflict"
)

type DiffResult struct {
	LocalOnly  []DiffItem `json:"local_only"`
	RemoteOnly []DiffItem `json:"remote_only"`
	Conflicts  []DiffItem `json:"conflicts"`
	InSync     bool       `json:"in_sync"`
}

func ComputeDiff(local, remote map[models.ValJsonKey]models.ValJson) DiffResult {
	result := DiffResult{
		LocalOnly:  []DiffItem{},
		RemoteOnly: []DiffItem{},
		Conflicts:  []DiffItem{},
	}

	remoteByKey := make(map[string]models.ValJson)
	for k, v := range remote {
		if k.Type == models.ORIGIN {
			remoteByKey[k.Key] = v
		}
	}

	for k, localVal := range local {
		if k.Type != models.ORIGIN {
			continue
		}
		remoteVal, existsRemote := remoteByKey[k.Key]
		if !existsRemote {
			lv := localVal
			result.LocalOnly = append(result.LocalOnly, DiffItem{
				Key: k.Key, LocalVal: &lv, DiffType: LocalOnly,
			})
		} else {
			if !valJsonEqual(localVal, remoteVal) {
				lv := localVal
				rv := remoteVal
				result.Conflicts = append(result.Conflicts, DiffItem{
					Key: k.Key, LocalVal: &lv, RemoteVal: &rv, DiffType: Conflict,
				})
			}
			delete(remoteByKey, k.Key)
		}
	}

	for key, remoteVal := range remoteByKey {
		rv := remoteVal
		result.RemoteOnly = append(result.RemoteOnly, DiffItem{
			Key: key, RemoteVal: &rv, DiffType: RemoteOnly,
		})
	}

	result.InSync = len(result.LocalOnly) == 0 && len(result.RemoteOnly) == 0 && len(result.Conflicts) == 0
	return result
}

func valJsonEqual(a, b models.ValJson) bool {
	if a.Val != b.Val {
		return false
	}
	if len(a.Tag) != len(b.Tag) {
		return false
	}
	tagSet := make(map[string]bool, len(a.Tag))
	for _, t := range a.Tag {
		tagSet[t] = true
	}
	for _, t := range b.Tag {
		if !tagSet[t] {
			return false
		}
	}
	return true
}

func ExecutePull(diff DiffResult, localStorage, remoteStorage db.Storage, dryRun bool) error {
	if diff.InSync {
		fmt.Println("Data is already in sync, no action needed")
		return nil
	}

	for _, item := range diff.LocalOnly {
		if dryRun {
			fmt.Printf("[dry-run] Delete local resource: %s\n", item.Key)
			continue
		}
		vjk := models.ValJsonKey{Key: item.Key, Type: models.ORIGIN}
		if err := localStorage.DeleteResource(vjk); err != nil {
			return fmt.Errorf("failed to delete local resource %s: %w", item.Key, err)
		}
		fmt.Printf("Deleted local resource: %s\n", item.Key)
	}

	for _, item := range diff.RemoteOnly {
		if dryRun {
			fmt.Printf("[dry-run] Add local resource: %s\n", item.Key)
			continue
		}
		vjk := models.ValJsonKey{Key: item.Key, Type: models.ORIGIN}
		if err := localStorage.SaveResource(vjk, *item.RemoteVal); err != nil {
			return fmt.Errorf("failed to add local resource %s: %w", item.Key, err)
		}
		fmt.Printf("Added local resource: %s\n", item.Key)
	}

	for _, item := range diff.Conflicts {
		if dryRun {
			fmt.Printf("[dry-run] Overwrite local resource: %s\n", item.Key)
			continue
		}
		vjk := models.ValJsonKey{Key: item.Key, Type: models.ORIGIN}
		if err := localStorage.UpdateResource(vjk, *item.RemoteVal); err != nil {
			return fmt.Errorf("failed to overwrite local resource %s: %w", item.Key, err)
		}
		fmt.Printf("Overwrote local resource: %s\n", item.Key)
	}

	return nil
}

func ExecutePush(diff DiffResult, localStorage, remoteStorage db.Storage, dryRun bool) error {
	if diff.InSync {
		fmt.Println("Data is already in sync, no action needed")
		return nil
	}

	for _, item := range diff.LocalOnly {
		if dryRun {
			fmt.Printf("[dry-run] Push to remote: %s\n", item.Key)
			continue
		}
		vjk := models.ValJsonKey{Key: item.Key, Type: models.ORIGIN}
		if err := remoteStorage.SaveResource(vjk, *item.LocalVal); err != nil {
			return fmt.Errorf("failed to push resource %s: %w", item.Key, err)
		}
		fmt.Printf("Pushed to remote: %s\n", item.Key)
	}

	for _, item := range diff.RemoteOnly {
		if dryRun {
			fmt.Printf("[dry-run] Delete remote resource: %s\n", item.Key)
			continue
		}
		vjk := models.ValJsonKey{Key: item.Key, Type: models.ORIGIN}
		if err := remoteStorage.DeleteResource(vjk); err != nil {
			return fmt.Errorf("failed to delete remote resource %s: %w", item.Key, err)
		}
		fmt.Printf("Deleted remote resource: %s\n", item.Key)
	}

	for _, item := range diff.Conflicts {
		if dryRun {
			fmt.Printf("[dry-run] Overwrite remote resource: %s\n", item.Key)
			continue
		}
		vjk := models.ValJsonKey{Key: item.Key, Type: models.ORIGIN}
		if err := remoteStorage.UpdateResource(vjk, *item.LocalVal); err != nil {
			return fmt.Errorf("failed to overwrite remote resource %s: %w", item.Key, err)
		}
		fmt.Printf("Overwrote remote resource: %s\n", item.Key)
	}

	return nil
}

func PrintDiff(diff DiffResult, remoteURL string) {
	fmt.Println("=== Data Sync Comparison ===")
	fmt.Printf("Remote: %s\n\n", remoteURL)

	if diff.InSync {
		fmt.Println("Data is already in sync, no action needed")
		return
	}

	if len(diff.LocalOnly) > 0 {
		fmt.Printf("Local only (%d):\n", len(diff.LocalOnly))
		for i, item := range diff.LocalOnly {
			fmt.Printf("  %d. %s\n", i+1, item.Key)
		}
		fmt.Println()
	}

	if len(diff.RemoteOnly) > 0 {
		fmt.Printf("Remote only (%d):\n", len(diff.RemoteOnly))
		for i, item := range diff.RemoteOnly {
			fmt.Printf("  %d. %s\n", i+1, item.Key)
		}
		fmt.Println()
	}

	if len(diff.Conflicts) > 0 {
		fmt.Printf("Content differs (%d):\n", len(diff.Conflicts))
		for i, item := range diff.Conflicts {
			localPreview := truncate(item.LocalVal.Val, 30)
			remotePreview := truncate(item.RemoteVal.Val, 30)
			fmt.Printf("  %d. %s  (local: %q / remote: %q)\n", i+1, item.Key, localPreview, remotePreview)
		}
		fmt.Println()
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
