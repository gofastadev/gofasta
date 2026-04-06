package httputil

import (
	"encoding/json"
	"net/http"
)

// JSON writes a JSON response with the given status code and data.
func JSON(w http.ResponseWriter, status int, data interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(data)
}

// Created writes a 201 Created JSON response.
func Created(w http.ResponseWriter, data interface{}) error {
	return JSON(w, http.StatusCreated, data)
}

// OK writes a 200 OK JSON response.
func OK(w http.ResponseWriter, data interface{}) error {
	return JSON(w, http.StatusOK, data)
}

// NoContent writes a 204 No Content response.
func NoContent(w http.ResponseWriter) error {
	w.WriteHeader(http.StatusNoContent)
	return nil
}
