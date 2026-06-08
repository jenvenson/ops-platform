package security

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestWriteFIMExecutionError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		errMessage string
		wantStatus int
	}{
		{name: "conflict", errMessage: "fim execution already running for policy 2 and server 1", wantStatus: http.StatusConflict},
		{name: "not_found", errMessage: "fim target not found for policy 2 and server 1", wantStatus: http.StatusNotFound},
		{name: "bad_request", errMessage: "fim policy 2 has no watch paths configured", wantStatus: http.StatusBadRequest},
		{name: "internal", errMessage: "ssh dial failed", wantStatus: http.StatusInternalServerError},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			writeFIMExecutionError(c, assertError(tc.errMessage), "failed to run fim scan")
			if w.Code != tc.wantStatus {
				t.Fatalf("expected status %d, got %d", tc.wantStatus, w.Code)
			}
		})
	}
}

type simpleError string

func (e simpleError) Error() string { return string(e) }

func assertError(message string) error {
	return simpleError(message)
}

