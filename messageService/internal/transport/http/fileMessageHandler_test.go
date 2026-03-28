package httptransport

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"messageService/internal/models"
	messageService "messageService/internal/service/messageService"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

type fakeUseCase struct {
	input            messageService.SendFileMessageInput
	result           *models.EventSendMessagePayload
	err              error
	called           bool
	canAccessAllowed bool
	canAccessErr     error
}

func (f *fakeUseCase) SendFileMessage(ctx context.Context, input messageService.SendFileMessageInput) (*models.EventSendMessagePayload, error) {
	f.called = true
	f.input = input
	return f.result, f.err
}

func (f *fakeUseCase) CanAccessFile(ctx context.Context, login string, fileID string) (bool, error) {
	return f.canAccessAllowed, f.canAccessErr
}

func newMultipartRequest(t *testing.T, files ...string) *http.Request {
	t.Helper()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	for idx, content := range files {
		part, err := writer.CreateFormFile("files", "file-"+string(rune('a'+idx))+".txt")
		if err != nil {
			t.Fatalf("create form file: %v", err)
		}
		_, _ = io.Copy(part, strings.NewReader(content))
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/chats/42/messages/files?login=alice", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

func newEmptyMultipartRequest(t *testing.T) *http.Request {
	t.Helper()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/chats/42/messages/files?login=alice", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

func TestUploadFilesSuccess(t *testing.T) {
	uc := &fakeUseCase{
		result: &models.EventSendMessagePayload{
			Id:          1,
			ChatId:      42,
			SenderLogin: "alice",
			Kind:        "file",
			Content:     "",
			Attachments: []models.AttachmentPayload{
				{
					FileID:           "file-1",
					OriginalFilename: "file-a.txt",
					MimeType:         "text/plain",
					Size:             3,
					Kind:             "file",
					DownloadURL:      "/files/file-1/download",
				},
			},
			CreatedAt: time.Unix(1, 0).UTC(),
		},
	}
	h := NewFileMessageHandler(uc)

	req := newMultipartRequest(t, "abc", "def")
	req = mux.SetURLVars(req, map[string]string{"chatId": "42"})

	rec := httptest.NewRecorder()
	h.UploadFileMessage(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
	if !uc.called {
		t.Fatal("expected use case to be called")
	}
	if uc.input.ChatID != 42 || uc.input.SenderLogin != "alice" || len(uc.input.Files) != 2 {
		t.Fatalf("unexpected input: %+v", uc.input)
	}

	var payload models.EventSendMessagePayload
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload.Kind != "file" || payload.Content != "" || len(payload.Attachments) != 1 {
		t.Fatalf("unexpected payload: %+v", payload)
	}
}

func TestUploadFilesErrors(t *testing.T) {
	tests := []struct {
		name   string
		req    *http.Request
		status int
		err    error
	}{
		{
			name:   "no files",
			req:    mux.SetURLVars(newEmptyMultipartRequest(t), map[string]string{"chatId": "42"}),
			status: http.StatusBadRequest,
			err:    nil,
		},
		{
			name:   "bad chat id",
			req:    mux.SetURLVars(newEmptyMultipartRequest(t), map[string]string{"chatId": "abc"}),
			status: http.StatusBadRequest,
			err:    nil,
		},
		{
			name: "sender not member",
			req: func() *http.Request {
				req := newMultipartRequest(t, "abc")
				req = mux.SetURLVars(req, map[string]string{"chatId": "42"})
				return req
			}(),
			status: http.StatusForbidden,
			err:    messageService.ErrSenderNotMember,
		},
		{
			name: "validation",
			req: func() *http.Request {
				req := newMultipartRequest(t, "abc")
				req = mux.SetURLVars(req, map[string]string{"chatId": "42"})
				return req
			}(),
			status: http.StatusUnprocessableEntity,
			err:    messageService.ErrFileValidation,
		},
		{
			name: "too many files",
			req: func() *http.Request {
				files := make([]string, 11)
				for i := range files {
					files[i] = "abc"
				}
				req := newMultipartRequest(t, files...)
				req = mux.SetURLVars(req, map[string]string{"chatId": "42"})
				return req
			}(),
			status: http.StatusBadRequest,
			err:    nil,
		},
		{
			name: "file service unavailable",
			req: func() *http.Request {
				req := newMultipartRequest(t, "abc")
				req = mux.SetURLVars(req, map[string]string{"chatId": "42"})
				return req
			}(),
			status: http.StatusServiceUnavailable,
			err:    messageService.ErrFileServiceUnavailable,
		},
		{
			name: "chat not found",
			req: func() *http.Request {
				req := newMultipartRequest(t, "abc")
				req = mux.SetURLVars(req, map[string]string{"chatId": "42"})
				return req
			}(),
			status: http.StatusNotFound,
			err:    messageService.ErrChatNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &fakeUseCase{err: tt.err}
			h := NewFileMessageHandler(uc)
			rec := httptest.NewRecorder()
			h.UploadFileMessage(rec, tt.req)
			if rec.Code != tt.status {
				t.Fatalf("unexpected status: %d", rec.Code)
			}
		})
	}
}
