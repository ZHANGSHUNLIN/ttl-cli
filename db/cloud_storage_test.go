package db

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"ttl-cli/models"
)

// mockAPIServer 创建模拟的 ttl server REST API
func mockAPIServer(t *testing.T) *httptest.Server {
	t.Helper()

	// 内存数据存储
	resources := make(map[string]struct {
		Value string   `json:"value"`
		Tags  []string `json:"tags"`
	})

	mux := http.NewServeMux()

	mux.HandleFunc("/api/v1/resources", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			var list []map[string]any
			for k, v := range resources {
				list = append(list, map[string]any{"key": k, "value": v.Value, "tags": v.Tags})
			}
			if list == nil {
				list = []map[string]any{}
			}
			resp, _ := json.Marshal(map[string]any{"code": 0, "message": "success", "data": list})
			w.Write(resp)
		case http.MethodPost:
			var req map[string]string
			json.NewDecoder(r.Body).Decode(&req)
			key := req["key"]
			if _, exists := resources[key]; exists {
				resp, _ := json.Marshal(map[string]any{"code": 1, "message": "当前key已经存在数据: " + key})
				w.Write(resp)
				return
			}
			resources[key] = struct {
				Value string   `json:"value"`
				Tags  []string `json:"tags"`
			}{Value: req["value"], Tags: []string{}}
			w.WriteHeader(http.StatusCreated)
			resp, _ := json.Marshal(map[string]any{"code": 0, "message": "success", "data": map[string]any{"key": key, "value": req["value"], "tags": []string{}}})
			w.Write(resp)
		}
	})

	mux.HandleFunc("/api/v1/resources/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		path := strings.TrimPrefix(r.URL.Path, "/api/v1/resources/")
		key := strings.SplitN(path, "/", 2)[0]

		switch r.Method {
		case http.MethodPut:
			if _, exists := resources[key]; !exists {
				w.WriteHeader(http.StatusNotFound)
				resp, _ := json.Marshal(map[string]any{"code": 1, "message": "未找到该资源"})
				w.Write(resp)
				return
			}
			var req map[string]string
			json.NewDecoder(r.Body).Decode(&req)
			res := resources[key]
			res.Value = req["value"]
			resources[key] = res
			resp, _ := json.Marshal(map[string]any{"code": 0, "message": "success"})
			w.Write(resp)
		case http.MethodDelete:
			if _, exists := resources[key]; !exists {
				w.WriteHeader(http.StatusNotFound)
				resp, _ := json.Marshal(map[string]any{"code": 1, "message": "未找到该资源"})
				w.Write(resp)
				return
			}
			delete(resources, key)
			resp, _ := json.Marshal(map[string]any{"code": 0, "message": "success"})
			w.Write(resp)
		}
	})

	mux.HandleFunc("/api/v1/audit/stats", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp, _ := json.Marshal(map[string]any{"code": 0, "message": "success", "data": models.AuditStats{
			TotalOperations: 5,
			ByOperation:     map[string]int{"get": 3, "add": 2},
			ByResource:      map[string]int{"mykey": 5},
		}})
		w.Write(resp)
	})

	mux.HandleFunc("/api/v1/history", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp, _ := json.Marshal(map[string]any{"code": 0, "message": "success", "data": []models.HistoryRecord{
			{ID: 1, ResourceKey: "mykey", Operation: "add", Timestamp: 1000, TimeStr: "2025-01-01 00:00:00"},
		}})
		w.Write(resp)
	})

	return httptest.NewServer(mux)
}

func TestCloudStorage_GetAllResources(t *testing.T) {
	srv := mockAPIServer(t)
	defer srv.Close()

	cs := NewCloudStorage(srv.URL, "", 30)
	_ = cs.Init()
	defer cs.Close()

	// 先保存一个资源
	key := models.ValJsonKey{Key: "test", Type: models.ORIGIN}
	_ = cs.SaveResource(key, models.ValJson{Val: "hello", Tag: []string{}})

	resources, err := cs.GetAllResources()
	if err != nil {
		t.Fatalf("GetAllResources 失败: %v", err)
	}
	if len(resources) != 1 {
		t.Fatalf("期望 1 个资源，实际: %d", len(resources))
	}
	if resources[key].Val != "hello" {
		t.Errorf("期望 value=hello，实际: %s", resources[key].Val)
	}
}

func TestCloudStorage_SaveResource(t *testing.T) {
	srv := mockAPIServer(t)
	defer srv.Close()

	cs := NewCloudStorage(srv.URL, "", 30)
	_ = cs.Init()
	defer cs.Close()

	key := models.ValJsonKey{Key: "new-res", Type: models.ORIGIN}
	err := cs.SaveResource(key, models.ValJson{Val: "value1", Tag: []string{}})
	if err != nil {
		t.Fatalf("SaveResource 失败: %v", err)
	}

	// 重复保存应该失败
	err = cs.SaveResource(key, models.ValJson{Val: "value2", Tag: []string{}})
	if err == nil {
		t.Fatal("重复保存应该返回错误")
	}
}

func TestCloudStorage_DeleteResource(t *testing.T) {
	srv := mockAPIServer(t)
	defer srv.Close()

	cs := NewCloudStorage(srv.URL, "", 30)
	_ = cs.Init()
	defer cs.Close()

	key := models.ValJsonKey{Key: "del-me", Type: models.ORIGIN}
	_ = cs.SaveResource(key, models.ValJson{Val: "bye", Tag: []string{}})

	err := cs.DeleteResource(key)
	if err != nil {
		t.Fatalf("DeleteResource 失败: %v", err)
	}

	// 验证已删除
	resources, _ := cs.GetAllResources()
	if len(resources) != 0 {
		t.Errorf("删除后仍有 %d 个资源", len(resources))
	}
}

func TestCloudStorage_UpdateResource(t *testing.T) {
	srv := mockAPIServer(t)
	defer srv.Close()

	cs := NewCloudStorage(srv.URL, "", 30)
	_ = cs.Init()
	defer cs.Close()

	key := models.ValJsonKey{Key: "upd", Type: models.ORIGIN}
	_ = cs.SaveResource(key, models.ValJson{Val: "old", Tag: []string{}})

	err := cs.UpdateResource(key, models.ValJson{Val: "new", Tag: []string{}})
	if err != nil {
		t.Fatalf("UpdateResource 失败: %v", err)
	}
}

func TestCloudStorage_AuthHeader(t *testing.T) {
	var receivedAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		resp, _ := json.Marshal(map[string]any{"code": 0, "message": "success", "data": []any{}})
		w.Write(resp)
	}))
	defer srv.Close()

	cs := NewCloudStorage(srv.URL, "my-secret-key", 30)
	_ = cs.Init()
	defer cs.Close()

	_, _ = cs.GetAllResources()

	if receivedAuth != "Bearer my-secret-key" {
		t.Errorf("期望 Authorization: Bearer my-secret-key，实际: %s", receivedAuth)
	}
}

func TestCloudStorage_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.Write([]byte(`{"code":0}`))
	}))
	defer srv.Close()

	cs := NewCloudStorage(srv.URL, "", 1) // 1 秒超时
	_ = cs.Init()
	defer cs.Close()

	_, err := cs.GetAllResources()
	if err == nil {
		t.Fatal("期望超时错误")
	}
}

func TestCloudStorage_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"code":2,"message":"内部错误"}`))
	}))
	defer srv.Close()

	cs := NewCloudStorage(srv.URL, "", 30)
	_ = cs.Init()
	defer cs.Close()

	_, err := cs.GetAllResources()
	if err == nil {
		t.Fatal("期望错误")
	}
	if !strings.Contains(err.Error(), "内部错误") {
		t.Errorf("错误信息应包含 '内部错误'，实际: %s", err.Error())
	}
}
