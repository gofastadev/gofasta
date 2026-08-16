package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContentType_SetsJSONHeader(t *testing.T) {
	handler := ContentType()(noopHandler)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	assert.Equal(t, http.StatusOK, rec.Code)
}
