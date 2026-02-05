// Package response provides HTTP response helpers for JSON APIs.
package response

import (
	"encoding/json"
	"net/http"
)

// JSON writes a JSON response with the given status code and data.
func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

// OK writes a 200 OK JSON response with the given data.
func OK(w http.ResponseWriter, data any) {
	JSON(w, http.StatusOK, data)
}

// Created writes a 201 Created JSON response with the given data.
func Created(w http.ResponseWriter, data any) {
	JSON(w, http.StatusCreated, data)
}

// NoContent writes a 204 No Content response.
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// MultiStatus writes a 207 Multi-Status JSON response with the given data.
func MultiStatus(w http.ResponseWriter, data any) {
	JSON(w, http.StatusMultiStatus, data)
}
