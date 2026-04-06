package api

import (
	"net/http"
	"ttl-cli/db"
)

func GetUserID(r *http.Request) string {
	return db.GetUserIDFromCtx(r.Context())
}

func GetStorage(r *http.Request) db.Storage {
	return db.GetStorageFromCtx(r.Context())
}
