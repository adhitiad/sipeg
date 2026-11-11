// Package apierror menyediakan bentuk respons error yang seragam untuk seluruh
// endpoint API. Paket ini dipakai bersama oleh auth dan httpserver supaya
// kode, bentuk JSON, dan semantik status tidak berbeda antar paket.
package apierror

import (
	"encoding/json"
	"net/http"
)

// Kode error kanonik yang dikembalikan API.
const (
	CodeUnauthorized       = "unauthorized"
	CodeForbidden          = "forbidden"
	CodeNotFound           = "not_found"
	CodeMethodNotAllowed   = "method_not_allowed"
	CodeBadRequest         = "bad_request"
	CodeInvalidQuery       = "invalid_query"
	CodeInvalidJSON        = "invalid_json"
	CodeInvalidID          = "invalid_id"
	CodeValidation         = "validation_error"
	CodeConflict           = "conflict"
	CodeNotReady           = "not_ready"
	CodeUnversionedPath    = "unversioned_path"
	CodeServiceUnavailable = "service_unavailable"
	CodeInternal           = "internal_error"
)

// Payload adalah isi objek "error" pada respons.
type Payload struct {
	Code      string            `json:"code"`
	Message   string            `json:"message,omitempty"`
	RequestID string            `json:"requestId,omitempty"`
	Fields    map[string]string `json:"fields,omitempty"`
	Rows      any               `json:"rows,omitempty"`
}

type envelope struct {
	Error Payload `json:"error"`
}

// Write menulis respons error dengan bentuk seragam.
func Write(w http.ResponseWriter, status int, code string) {
	WriteWith(w, status, Payload{Code: code})
}

// WriteWith menulis respons error dengan payload lengkap.
func WriteWith(w http.ResponseWriter, status int, payload Payload) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(envelope{Error: payload})
}
