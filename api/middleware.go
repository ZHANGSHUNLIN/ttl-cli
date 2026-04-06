package api

import (
	"net/http"
	"strings"
	"ttl-cli/db"
)

func AuthMiddleware(apiKey string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if apiKey == "" {
			next.ServeHTTP(w, r)
			return
		}

		auth := r.Header.Get("Authorization")
		expected := "Bearer " + apiKey
		if auth != expected {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusUnauthorized)
			writeJSON(w, http.StatusUnauthorized, Response{Code: 1, Message: "Unauthorized: Invalid API Key"})
			return
		}

		next.ServeHTTP(w, r)
	})
}

func MultiTenantAuthMiddleware(userStore *db.UserStore, tenantMgr *db.TenantStorageManager, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			writeJSON(w, http.StatusUnauthorized, Response{Code: 1, Message: "Unauthorized: Missing API Key"})
			return
		}

		apiKey := strings.TrimPrefix(auth, "Bearer ")
		user := userStore.FindByAPIKey(apiKey)
		if user == nil {
			writeJSON(w, http.StatusUnauthorized, Response{Code: 1, Message: "Unauthorized: Invalid API Key"})
			return
		}

		if !user.Active {
			writeJSON(w, http.StatusUnauthorized, Response{Code: 1, Message: "Unauthorized: User disabled"})
			return
		}

		storage, err := tenantMgr.GetStorage(user.ID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, Response{Code: 2, Message: "failed to get user storage: " + err.Error()})
			return
		}

		ctx := db.WithUserStorage(r.Context(), user.ID, storage)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
