package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetRoot(t *testing.T) {
	req := httptest.NewRequest("GET", "http://test/", nil)
	rw := httptest.NewRecorder()

	getRoot(rw, req)

	resp := rw.Result()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status %v, got %v", http.StatusOK, resp.StatusCode)
	}
}
