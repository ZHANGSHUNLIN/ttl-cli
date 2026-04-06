package ai

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClient(t *testing.T) {
	c := NewClient("https://api.example.com", "sk-test", "gpt-4o-mini", 30)
	if c.BaseURL != "https://api.example.com" {
		t.Errorf("BaseURL = %s, want https://api.example.com", c.BaseURL)
	}
	if c.APIKey != "sk-test" {
		t.Errorf("APIKey = %s, want sk-test", c.APIKey)
	}
	if c.Model != "gpt-4o-mini" {
		t.Errorf("Model = %s, want gpt-4o-mini", c.Model)
	}
}

func TestNewClient_TrailingSlash(t *testing.T) {
	c := NewClient("https://api.example.com/", "sk-test", "gpt-4o-mini", 30)
	if c.BaseURL != "https://api.example.com" {
		t.Errorf("BaseURL should strip trailing slash, got %s", c.BaseURL)
	}
}

func TestChat_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证请求头
		if r.Header.Get("Authorization") != "Bearer sk-test" {
			t.Errorf("Authorization = %s, want Bearer sk-test", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type = %s, want application/json", r.Header.Get("Content-Type"))
		}
		if r.Method != "POST" {
			t.Errorf("Method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("Path = %s, want /v1/chat/completions", r.URL.Path)
		}

		// 验证请求体
		var req chatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Model != "test-model" {
			t.Errorf("Model = %s, want test-model", req.Model)
		}
		if len(req.Messages) != 1 {
			t.Fatalf("Messages count = %d, want 1", len(req.Messages))
		}

		// 返回响应
		resp := chatResponse{
			Choices: []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			}{
				{Message: struct {
					Content string `json:"content"`
				}{Content: "Hello!"}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := NewClient(server.URL, "sk-test", "test-model", 10)
	result, err := c.Chat([]Message{{Role: "user", Content: "Hi"}})
	if err != nil {
		t.Fatalf("Chat error: %v", err)
	}
	if result != "Hello!" {
		t.Errorf("Chat result = %q, want %q", result, "Hello!")
	}
}

func TestChat_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, "invalid api key")
	}))
	defer server.Close()

	c := NewClient(server.URL, "bad-key", "test-model", 10)
	_, err := c.Chat([]Message{{Role: "user", Content: "Hi"}})
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
	if got := err.Error(); got == "" {
		t.Error("error message should not be empty")
	}
}

func TestChat_EmptyChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := chatResponse{Choices: nil}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := NewClient(server.URL, "sk-test", "test-model", 10)
	_, err := c.Chat([]Message{{Role: "user", Content: "Hi"}})
	if err == nil {
		t.Fatal("expected error for empty choices")
	}
}

func TestChat_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"error": map[string]string{
				"message": "rate limit exceeded",
			},
			"choices": []interface{}{},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := NewClient(server.URL, "sk-test", "test-model", 10)
	_, err := c.Chat([]Message{{Role: "user", Content: "Hi"}})
	if err == nil {
		t.Fatal("expected error for API error response")
	}
}
