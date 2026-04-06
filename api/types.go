package api

import (
	"encoding/json"
	"net/http"
)

type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

type ResourceDTO struct {
	Key   string   `json:"key"`
	Value string   `json:"value"`
	Tags  []string `json:"tags"`
}

type CreateResourceRequest struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type UpdateResourceRequest struct {
	Value string `json:"value"`
}

type AddTagsRequest struct {
	Tags []string `json:"tags"`
}

type RenameRequest struct {
	NewKey string `json:"new_key"`
}

func writeJSON(w http.ResponseWriter, statusCode int, resp Response) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(resp)
}

func writeSuccess(w http.ResponseWriter, data any) {
	writeJSON(w, http.StatusOK, Response{Code: 0, Message: "success", Data: data})
}

func writeCreated(w http.ResponseWriter, data any) {
	writeJSON(w, http.StatusCreated, Response{Code: 0, Message: "success", Data: data})
}

func writeBizError(w http.ResponseWriter, msg string) {
	writeJSON(w, http.StatusOK, Response{Code: 1, Message: msg})
}

func writeSysError(w http.ResponseWriter, msg string) {
	writeJSON(w, http.StatusInternalServerError, Response{Code: 2, Message: msg})
}

func writeNotFound(w http.ResponseWriter, msg string) {
	writeJSON(w, http.StatusNotFound, Response{Code: 1, Message: msg})
}
