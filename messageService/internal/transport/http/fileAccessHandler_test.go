package httptransport

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

func TestCheckFileAccess(t *testing.T) {
	tests := []struct {
		name     string
		login    string
		fileID   string
		allowed  bool
		err      error
		expected int
	}{
		{
			name:     "allowed",
			login:    "alice",
			fileID:   "11111111-1111-1111-1111-111111111111",
			allowed:  true,
			expected: http.StatusNoContent,
		},
		{
			name:     "forbidden",
			login:    "alice",
			fileID:   "11111111-1111-1111-1111-111111111111",
			allowed:  false,
			expected: http.StatusForbidden,
		},
		{
			name:     "bad login",
			login:    "",
			fileID:   "11111111-1111-1111-1111-111111111111",
			allowed:  true,
			expected: http.StatusBadRequest,
		},
		{
			name:     "bad file id",
			login:    "alice",
			fileID:   "bad-id",
			allowed:  true,
			expected: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &fakeUseCase{err: tt.err}
			h := NewFileMessageHandler(uc)

			req := httptest.NewRequest(http.MethodGet, "/internal/files/"+tt.fileID+"/access?login="+tt.login, nil)
			req = mux.SetURLVars(req, map[string]string{"fileId": tt.fileID})
			rec := httptest.NewRecorder()

			if tt.fileID == "11111111-1111-1111-1111-111111111111" {
				uc.canAccessAllowed = tt.allowed
			}

			h.CheckFileAccess(rec, req)

			if rec.Code != tt.expected {
				t.Fatalf("unexpected status: got %d want %d", rec.Code, tt.expected)
			}
		})
	}
}

func TestCheckFileAccessInternalError(t *testing.T) {
	uc := &fakeUseCase{canAccessErr: context.DeadlineExceeded}
	h := NewFileMessageHandler(uc)

	req := httptest.NewRequest(http.MethodGet, "/internal/files/11111111-1111-1111-1111-111111111111/access?login=alice", nil)
	req = mux.SetURLVars(req, map[string]string{"fileId": "11111111-1111-1111-1111-111111111111"})
	rec := httptest.NewRecorder()

	h.CheckFileAccess(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("unexpected status: got %d want %d", rec.Code, http.StatusInternalServerError)
	}
}
