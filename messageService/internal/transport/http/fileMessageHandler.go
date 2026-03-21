package httptransport

import (
	"context"
	"encoding/json"
	"errors"
	"messageService/internal/models"
	messageService "messageService/internal/service/messageService"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

var (
	ErrInvalidChatID = errors.New("invalid chat id")
	ErrNoFiles       = errors.New("no files")
	ErrTooManyFiles  = errors.New("too many files")
	ErrInvalidLogin  = errors.New("invalid login")
	ErrBadMultipart  = errors.New("bad multipart")
	ErrInvalidFileID = errors.New("invalid file id")
	ErrFileForbidden = errors.New("file access forbidden")
)

var uuidPattern = regexp.MustCompile(`^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12}$`)

type SendFileMessageUseCase interface {
	SendFileMessage(ctx context.Context, input messageService.SendFileMessageInput) (*models.EventSendMessagePayload, error)
	CanAccessFile(ctx context.Context, login string, fileID string) (bool, error)
}

type FileMessageHandler struct {
	useCase SendFileMessageUseCase
}

func NewFileMessageHandler(useCase SendFileMessageUseCase) *FileMessageHandler {
	return &FileMessageHandler{useCase: useCase}
}

func (h *FileMessageHandler) UploadFileMessage(w http.ResponseWriter, r *http.Request) {
	login := strings.TrimSpace(r.URL.Query().Get("login"))
	if login == "" {
		writeHTTPError(w, ErrInvalidLogin)
		return
	}

	chatID, err := parseChatID(r)
	if err != nil {
		writeHTTPError(w, err)
		return
	}

	uploads, cleanup, err := parseUploadedFiles(r)
	if err != nil {
		writeHTTPError(w, err)
		return
	}
	defer cleanup()

	if len(uploads) == 0 {
		writeHTTPError(w, ErrNoFiles)
		return
	}
	if len(uploads) > 10 {
		writeHTTPError(w, ErrTooManyFiles)
		return
	}

	result, err := h.useCase.SendFileMessage(r.Context(), messageService.SendFileMessageInput{
		ChatID:      chatID,
		SenderLogin: login,
		Files:       uploads,
	})
	if err != nil {
		writeHTTPError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, result)
}

func (h *FileMessageHandler) CheckFileAccess(w http.ResponseWriter, r *http.Request) {
	login := strings.TrimSpace(r.URL.Query().Get("login"))
	if login == "" {
		writeHTTPError(w, ErrInvalidLogin)
		return
	}

	fileID, err := parseFileID(r)
	if err != nil {
		writeHTTPError(w, err)
		return
	}

	allowed, err := h.useCase.CanAccessFile(r.Context(), login, fileID)
	if err != nil {
		writeHTTPError(w, err)
		return
	}
	if !allowed {
		writeHTTPError(w, ErrFileForbidden)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func parseChatID(r *http.Request) (int, error) {
	rawID := strings.TrimSpace(mux.Vars(r)["chatId"])
	if rawID == "" {
		return 0, ErrInvalidChatID
	}

	chatID, err := strconv.Atoi(rawID)
	if err != nil || chatID <= 0 {
		return 0, ErrInvalidChatID
	}

	return chatID, nil
}

func parseFileID(r *http.Request) (string, error) {
	rawID := strings.TrimSpace(mux.Vars(r)["fileId"])
	if rawID == "" {
		return "", ErrInvalidFileID
	}
	if !uuidPattern.MatchString(rawID) {
		return "", ErrInvalidFileID
	}
	return rawID, nil
}

func parseUploadedFiles(r *http.Request) ([]messageService.FileUpload, func(), error) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		return nil, func() {}, ErrBadMultipart
	}

	var uploads []messageService.FileUpload
	var closers []messageService.FileUpload

	fileHeaders := r.MultipartForm.File["files"]
	for _, header := range fileHeaders {
		file, err := header.Open()
		if err != nil {
			closeAll(closers)
			return nil, func() {}, ErrBadMultipart
		}

		upload := messageService.FileUpload{
			Filename:    header.Filename,
			ContentType: header.Header.Get("Content-Type"),
			Size:        header.Size,
			Reader:      file,
		}
		closers = append(closers, upload)
		uploads = append(uploads, upload)
	}

	cleanup := func() {
		closeAll(closers)
		if r.MultipartForm != nil {
			_ = r.MultipartForm.RemoveAll()
		}
	}

	return uploads, cleanup, nil
}

func closeAll(closers []messageService.FileUpload) {
	for _, closer := range closers {
		if closer.Reader != nil {
			_ = closer.Reader.Close()
		}
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeHTTPError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError

	switch {
	case errors.Is(err, ErrInvalidChatID), errors.Is(err, ErrInvalidLogin), errors.Is(err, ErrNoFiles), errors.Is(err, ErrTooManyFiles), errors.Is(err, ErrBadMultipart), errors.Is(err, ErrInvalidFileID):
		status = http.StatusBadRequest
	case errors.Is(err, messageService.ErrChatNotFound):
		status = http.StatusNotFound
	case errors.Is(err, messageService.ErrSenderNotMember):
		status = http.StatusForbidden
	case errors.Is(err, ErrFileForbidden):
		status = http.StatusForbidden
	case errors.Is(err, messageService.ErrFileValidation):
		status = http.StatusUnprocessableEntity
	case errors.Is(err, messageService.ErrFileServiceUnavailable):
		status = http.StatusServiceUnavailable
	}

	http.Error(w, http.StatusText(status), status)
}
