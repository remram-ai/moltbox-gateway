package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleMCPRejectsMissingTokenBeforeMethodCheck(t *testing.T) {
	server := NewServer(Config{})

	request := httptest.NewRequest(http.MethodGet, "/mcp", nil)
	recorder := httptest.NewRecorder()

	server.handleMCP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected %d, got %d", http.StatusUnauthorized, recorder.Code)
	}
}
