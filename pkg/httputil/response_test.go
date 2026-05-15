package httputil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSON_WritesStatusAndBody(t *testing.T) {
	tests := []struct {
		name   string
		status int
		data   interface{}
	}{
		{"ok with map", http.StatusOK, map[string]string{"msg": "hello"}},
		{"created with struct", http.StatusCreated, struct {
			ID int `json:"id"`
		}{ID: 42}},
		{"bad request", http.StatusBadRequest, map[string]string{"error": "bad"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			err := JSON(rec, tt.status, tt.data)
			require.NoError(t, err)

			assert.Equal(t, tt.status, rec.Code)
			assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

			var got map[string]interface{}
			err = json.Unmarshal(rec.Body.Bytes(), &got)
			require.NoError(t, err)
		})
	}
}

func TestOK_Returns200(t *testing.T) {
	rec := httptest.NewRecorder()
	err := OK(rec, map[string]string{"status": "ok"})
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var body map[string]string
	json.Unmarshal(rec.Body.Bytes(), &body)
	assert.Equal(t, "ok", body["status"])
}

func TestCreated_Returns201(t *testing.T) {
	rec := httptest.NewRecorder()
	err := Created(rec, map[string]int{"id": 1})
	require.NoError(t, err)

	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var body map[string]int
	json.Unmarshal(rec.Body.Bytes(), &body)
	assert.Equal(t, 1, body["id"])
}

func TestNoContent_Returns204(t *testing.T) {
	rec := httptest.NewRecorder()
	err := NoContent(rec)
	require.NoError(t, err)

	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Empty(t, rec.Body.String())
}

func TestJSON_NilData(t *testing.T) {
	rec := httptest.NewRecorder()
	err := JSON(rec, http.StatusOK, nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "null")
}

func TestJSON_SliceData(t *testing.T) {
	rec := httptest.NewRecorder()
	data := []string{"a", "b", "c"}
	err := JSON(rec, http.StatusOK, data)
	require.NoError(t, err)

	var result []string
	json.Unmarshal(rec.Body.Bytes(), &result)
	assert.Equal(t, data, result)
}
