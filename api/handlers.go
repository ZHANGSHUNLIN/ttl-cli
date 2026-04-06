package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
	"ttl-cli/db"
	"ttl-cli/models"
	"ttl-cli/util"
)

func storageFrom(r *http.Request) db.Storage {
	if s := GetStorage(r); s != nil {
		return s
	}
	return db.Stor
}

func ResourcesHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handleGetResources(w, r)
	case http.MethodPost:
		handleCreateResource(w, r)
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

func ResourceHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/resources/")
	if path == "" {
		writeNotFound(w, "missing resource key")
		return
	}

	parts := strings.SplitN(path, "/", 3)
	key := parts[0]

	if len(parts) == 1 {
		switch r.Method {
		case http.MethodPut:
			handleUpdateResource(w, r, key)
		case http.MethodDelete:
			handleDeleteResource(w, r, key)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	subPath := parts[1]

	if subPath == "tags" {
		if len(parts) == 2 {
			if r.Method == http.MethodPost {
				handleAddTags(w, r, key)
			} else {
				http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			}
		} else {
			tag := parts[2]
			if r.Method == http.MethodDelete {
				handleDeleteTag(w, r, key, tag)
			} else {
				http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			}
		}
		return
	}

	if subPath == "rename" && r.Method == http.MethodPost {
		handleRenameResource(w, r, key)
		return
	}

	writeNotFound(w, "unknown path")
}

func AuditStatsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	stor := storageFrom(r)
	stats, err := stor.GetAuditStats()
	if err != nil {
		writeSysError(w, "failed to get audit stats: "+err.Error())
		return
	}
	writeSuccess(w, stats)
}

func HistoryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	stor := storageFrom(r)
	records, err := stor.GetAllHistoryRecords()
	if err != nil {
		writeSysError(w, "failed to get history records: "+err.Error())
		return
	}

	limitStr := r.URL.Query().Get("limit")
	if limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 && len(records) > limit {
			records = records[:limit]
		}
	}

	writeSuccess(w, records)
}

func handleGetResources(w http.ResponseWriter, r *http.Request) {
	stor := storageFrom(r)
	resources, err := stor.GetAllResources()
	if err != nil {
		writeSysError(w, "failed to get resources: "+err.Error())
		return
	}

	keyword := r.URL.Query().Get("q")

	var result []ResourceDTO
	for k, v := range resources {
		if k.Type != models.ORIGIN {
			continue
		}
		if keyword != "" {
			matched := util.ContainsIgnoreCase(k.Key, keyword)
			if !matched {
				for _, tag := range v.Tag {
					if util.ContainsIgnoreCase(tag, keyword) {
						matched = true
						break
					}
				}
			}
			if !matched {
				continue
			}
		}
		result = append(result, ResourceDTO{Key: k.Key, Value: v.Val, Tags: v.Tag})
	}

	if result == nil {
		result = []ResourceDTO{}
	}
	writeSuccess(w, result)
}

func handleCreateResource(w http.ResponseWriter, r *http.Request) {
	var req CreateResourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBizError(w, "request body parse failed: "+err.Error())
		return
	}
	if req.Key == "" {
		writeBizError(w, "key cannot be empty")
		return
	}

	stor := storageFrom(r)
	vjk := models.ValJsonKey{Key: req.Key, Type: models.ORIGIN}
	resources, err := stor.GetAllResources()
	if err != nil {
		writeSysError(w, "failed to get resources: "+err.Error())
		return
	}
	if _, exists := resources[vjk]; exists {
		writeBizError(w, "key already exists: "+req.Key)
		return
	}

	value := util.UnescapeString(req.Value)
	newResource := models.ValJson{Val: value, Tag: []string{}}
	if err := stor.SaveResource(vjk, newResource); err != nil {
		writeSysError(w, "failed to save resource: "+err.Error())
		return
	}
	_ = stor.SaveAuditRecord(models.AuditRecord{
		ResourceKey: req.Key,
		Operation:   "add",
		Timestamp:   time.Now().Unix(),
		Count:       1,
	})

	writeCreated(w, ResourceDTO{Key: req.Key, Value: value, Tags: []string{}})
}

func handleUpdateResource(w http.ResponseWriter, r *http.Request, key string) {
	var req UpdateResourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBizError(w, "request body parse failed: "+err.Error())
		return
	}

	stor := storageFrom(r)
	vjk := models.ValJsonKey{Key: key, Type: models.ORIGIN}
	resources, err := stor.GetAllResources()
	if err != nil {
		writeSysError(w, "failed to get resources: "+err.Error())
		return
	}
	existing, exists := resources[vjk]
	if !exists {
		writeNotFound(w, "resource not found: "+key)
		return
	}

	value := util.UnescapeString(req.Value)
	updated := models.ValJson{Val: value, Tag: existing.Tag}
	if err := stor.UpdateResource(vjk, updated); err != nil {
		writeSysError(w, "failed to update resource: "+err.Error())
		return
	}
	_ = stor.SaveAuditRecord(models.AuditRecord{
		ResourceKey: key,
		Operation:   "update",
		Timestamp:   time.Now().Unix(),
		Count:       1,
	})

	writeSuccess(w, ResourceDTO{Key: key, Value: value, Tags: existing.Tag})
}

func handleDeleteResource(w http.ResponseWriter, r *http.Request, key string) {
	stor := storageFrom(r)
	vjk := models.ValJsonKey{Key: key, Type: models.ORIGIN}
	resources, err := stor.GetAllResources()
	if err != nil {
		writeSysError(w, "failed to get resources: "+err.Error())
		return
	}
	if _, exists := resources[vjk]; !exists {
		writeNotFound(w, "resource not found: "+key)
		return
	}

	_ = stor.DeleteHistoryRecords(key)
	_ = stor.DeleteAuditRecords(key)

	if err := stor.DeleteResource(vjk); err != nil {
		writeSysError(w, "failed to delete resource: "+err.Error())
		return
	}

	writeSuccess(w, nil)
}

func handleAddTags(w http.ResponseWriter, r *http.Request, key string) {
	var req AddTagsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBizError(w, "request body parse failed: "+err.Error())
		return
	}
	if len(req.Tags) == 0 {
		writeBizError(w, "at least one tag is required")
		return
	}

	stor := storageFrom(r)
	vjk := models.ValJsonKey{Key: key, Type: models.ORIGIN}
	resources, err := stor.GetAllResources()
	if err != nil {
		writeSysError(w, "failed to get resources: "+err.Error())
		return
	}
	resource, exists := resources[vjk]
	if !exists {
		writeNotFound(w, "resource not found: "+key)
		return
	}

	newTags := append(resource.Tag, req.Tags...)
	resource.Tag = util.RemoveDuplicates(newTags)
	if err := stor.SaveResource(vjk, resource); err != nil {
		writeSysError(w, "failed to save resource: "+err.Error())
		return
	}

	writeSuccess(w, ResourceDTO{Key: key, Value: resource.Val, Tags: resource.Tag})
}

func handleDeleteTag(w http.ResponseWriter, r *http.Request, key, tag string) {
	stor := storageFrom(r)
	vjk := models.ValJsonKey{Key: key, Type: models.ORIGIN}
	resources, err := stor.GetAllResources()
	if err != nil {
		writeSysError(w, "failed to get resources: "+err.Error())
		return
	}
	resource, exists := resources[vjk]
	if !exists {
		writeNotFound(w, "resource not found: "+key)
		return
	}

	newTags := make([]string, 0, len(resource.Tag))
	for _, t := range resource.Tag {
		if t != tag {
			newTags = append(newTags, t)
		}
	}
	resource.Tag = newTags
	if err := stor.SaveResource(vjk, resource); err != nil {
		writeSysError(w, "failed to save resource: "+err.Error())
		return
	}

	writeSuccess(w, ResourceDTO{Key: key, Value: resource.Val, Tags: resource.Tag})
}

func handleRenameResource(w http.ResponseWriter, r *http.Request, key string) {
	var req RenameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBizError(w, "request body parse failed: "+err.Error())
		return
	}
	if req.NewKey == "" {
		writeBizError(w, "new_key cannot be empty")
		return
	}

	stor := storageFrom(r)
	oldVjk := models.ValJsonKey{Key: key, Type: models.ORIGIN}
	resources, err := stor.GetAllResources()
	if err != nil {
		writeSysError(w, "failed to get resources: "+err.Error())
		return
	}
	resource, exists := resources[oldVjk]
	if !exists {
		writeNotFound(w, "resource not found: "+key)
		return
	}

	if err := stor.DeleteResource(oldVjk); err != nil {
		writeSysError(w, "failed to delete old resource: "+err.Error())
		return
	}

	newVjk := models.ValJsonKey{Key: req.NewKey, Type: models.ORIGIN}
	if err := stor.SaveResource(newVjk, resource); err != nil {
		writeSysError(w, "failed to save new resource: "+err.Error())
		return
	}

	writeSuccess(w, ResourceDTO{Key: req.NewKey, Value: resource.Val, Tags: resource.Tag})
}
