package messageService

import (
	"context"
	"encoding/json"
	"errors"
	fileServiceClient "messageService/internal/client/fileService"
	"messageService/internal/models"
	"messageService/internal/storage/postgres/messageStorage"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

type sendFileMessageStorageStub struct {
	chatExists        bool
	isMember          bool
	createResult      *models.EventSendMessagePayload
	createErr         error
	getAllChatMembers []string
}

func (s *sendFileMessageStorageStub) SaveMessage(ctx context.Context, msg *models.EventSendMessagePayload) (*models.EventSendMessagePayload, error) {
	return nil, errors.New("unexpected SaveMessage call")
}

func (s *sendFileMessageStorageStub) GetMessageHistory(ctx context.Context, startNum int, chatId int) ([]models.EventSendMessagePayload, error) {
	return nil, nil
}

func (s *sendFileMessageStorageStub) GetAllUserChats(ctx context.Context, login string) ([]models.EventChatPayload, error) {
	return nil, nil
}

func (s *sendFileMessageStorageStub) CreateChat(ctx context.Context, maxMembers int) (int, error) {
	return 0, nil
}

func (s *sendFileMessageStorageStub) AddUserToChat(ctx context.Context, chatId int, userLogin string) error {
	return nil
}

func (s *sendFileMessageStorageStub) GetAllChatMembers(ctx context.Context, chatId int) ([]string, error) {
	return s.getAllChatMembers, nil
}

func (s *sendFileMessageStorageStub) ChatExists(ctx context.Context, chatId int) (bool, error) {
	return s.chatExists, nil
}

func (s *sendFileMessageStorageStub) IsChatMember(ctx context.Context, chatId int, login string) (bool, error) {
	return s.isMember, nil
}

func (s *sendFileMessageStorageStub) UserHasAccessToFile(ctx context.Context, login string, fileID string) (bool, error) {
	return false, nil
}

func (s *sendFileMessageStorageStub) CreateFileMessage(ctx context.Context, chatId int, senderLogin string, attachments []messageStorage.FileAttachmentRecord) (*models.EventSendMessagePayload, error) {
	if s.createErr != nil {
		return nil, s.createErr
	}
	if s.createResult != nil {
		return s.createResult, nil
	}
	return &models.EventSendMessagePayload{
		Id:          1,
		ChatId:      chatId,
		SenderLogin: senderLogin,
		Kind:        "file",
		Content:     "",
		Attachments: []models.AttachmentPayload{},
		CreatedAt:   time.Unix(1, 0).UTC(),
	}, nil
}

type sendFileMessageFileClientStub struct {
	uploadedFiles []*fileServiceClient.UploadedFile
	uploadErrAt   int
	uploadErr     error
	deleteCalls   []string
	uploadCalls   int
}

func (s *sendFileMessageFileClientStub) Upload(ctx context.Context, request fileServiceClient.UploadRequest) (*fileServiceClient.UploadedFile, error) {
	s.uploadCalls++
	if s.uploadErr != nil && s.uploadCalls == s.uploadErrAt {
		return nil, s.uploadErr
	}
	return s.uploadedFiles[s.uploadCalls-1], nil
}

func (s *sendFileMessageFileClientStub) Delete(ctx context.Context, fileID string) error {
	s.deleteCalls = append(s.deleteCalls, fileID)
	return nil
}

type sendFileChatMembersStorageStub struct {
	members []string
}

func (s *sendFileChatMembersStorageStub) CreateMembersList(ctx context.Context, chatId int, usersLogin ...string) error {
	return nil
}

func (s *sendFileChatMembersStorageStub) GetMembersList(ctx context.Context, chatId int) ([]string, error) {
	return s.members, nil
}

func (s *sendFileChatMembersStorageStub) AddMemberToList(ctx context.Context, chatId int, userLogin string) error {
	return nil
}

func (s *sendFileChatMembersStorageStub) DeleteMemberFromList(ctx context.Context, chatId int, userLogin string) error {
	return nil
}

func (s *sendFileChatMembersStorageStub) UpdateListTTL(ctx context.Context, chatId int) error {
	return nil
}

type sendFileClientStorageStub struct{}

func (s *sendFileClientStorageStub) GetClient(login string) (*models.Client, error) {
	return &models.Client{Login: login, Conn: &websocket.Conn{}, Send: make(chan json.RawMessage)}, errors.New("offline")
}

func (s *sendFileClientStorageStub) SaveClient(login string, conn *websocket.Conn, cancel context.CancelFunc) {
}

func (s *sendFileClientStorageStub) DeleteClient(login string) {}

type nopReadSeekCloser struct {
	*strings.Reader
}

func (n nopReadSeekCloser) Close() error { return nil }

func TestSendFileMessageSuccess(t *testing.T) {
	fileClient := &sendFileMessageFileClientStub{
		uploadedFiles: []*fileServiceClient.UploadedFile{
			{ID: "file-1", OriginalFilename: "photo.png", MimeType: "image/png", Size: 10},
			{ID: "file-2", OriginalFilename: "doc.pdf", MimeType: "application/pdf", Size: 20},
		},
	}
	storage := &sendFileMessageStorageStub{
		chatExists: true,
		isMember:   true,
		createResult: &models.EventSendMessagePayload{
			Id:          7,
			ChatId:      42,
			SenderLogin: "alice",
			Kind:        "file",
			Content:     "",
			Attachments: []models.AttachmentPayload{
				{FileID: "file-1", OriginalFilename: "photo.png", MimeType: "image/png", Size: 10, Kind: "image"},
				{FileID: "file-2", OriginalFilename: "doc.pdf", MimeType: "application/pdf", Size: 20, Kind: "file"},
			},
			CreatedAt: time.Unix(1, 0).UTC(),
		},
	}
	handler := &messageHandler{
		ClientStorage:      &sendFileClientStorageStub{},
		ChatMembersStorage: &sendFileChatMembersStorageStub{members: []string{"alice"}},
		MessageStorage:     storage,
		FileService:        fileClient,
	}

	payload, err := handler.SendFileMessage(context.Background(), SendFileMessageInput{
		ChatID:      42,
		SenderLogin: "alice",
		Files: []FileUpload{
			{Filename: "photo.png", Reader: nopReadSeekCloser{strings.NewReader("a")}},
			{Filename: "doc.pdf", Reader: nopReadSeekCloser{strings.NewReader("b")}},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if payload.Kind != "file" || payload.Content != "" || len(payload.Attachments) != 2 {
		t.Fatalf("unexpected payload: %+v", payload)
	}
	if payload.Attachments[0].DownloadURL != "/files/file-1/download" {
		t.Fatalf("unexpected download url: %+v", payload.Attachments[0])
	}
	if len(fileClient.deleteCalls) != 0 {
		t.Fatalf("did not expect compensation delete, got %v", fileClient.deleteCalls)
	}
}

func TestSendFileMessageCompensatesOnPartialUploadFailure(t *testing.T) {
	fileClient := &sendFileMessageFileClientStub{
		uploadedFiles: []*fileServiceClient.UploadedFile{
			{ID: "file-1", OriginalFilename: "ok.txt", MimeType: "text/plain", Size: 10},
		},
		uploadErrAt: 2,
		uploadErr:   fileServiceClient.ErrValidation,
	}
	handler := &messageHandler{
		ClientStorage:      &sendFileClientStorageStub{},
		ChatMembersStorage: &sendFileChatMembersStorageStub{members: []string{"alice"}},
		MessageStorage:     &sendFileMessageStorageStub{chatExists: true, isMember: true},
		FileService:        fileClient,
	}

	_, err := handler.SendFileMessage(context.Background(), SendFileMessageInput{
		ChatID:      42,
		SenderLogin: "alice",
		Files: []FileUpload{
			{Filename: "ok.txt", Reader: nopReadSeekCloser{strings.NewReader("a")}},
			{Filename: "bad.txt", Reader: nopReadSeekCloser{strings.NewReader("b")}},
		},
	})
	if !errors.Is(err, ErrFileValidation) {
		t.Fatalf("expected validation error, got %v", err)
	}
	if len(fileClient.deleteCalls) != 1 || fileClient.deleteCalls[0] != "file-1" {
		t.Fatalf("expected compensation delete for uploaded file, got %v", fileClient.deleteCalls)
	}
}

func TestSendFileMessageCompensatesOnDBFailure(t *testing.T) {
	fileClient := &sendFileMessageFileClientStub{
		uploadedFiles: []*fileServiceClient.UploadedFile{
			{ID: "file-1", OriginalFilename: "doc.pdf", MimeType: "application/pdf", Size: 10},
		},
	}
	handler := &messageHandler{
		ClientStorage:      &sendFileClientStorageStub{},
		ChatMembersStorage: &sendFileChatMembersStorageStub{members: []string{"alice"}},
		MessageStorage:     &sendFileMessageStorageStub{chatExists: true, isMember: true, createErr: errors.New("db down")},
		FileService:        fileClient,
	}

	_, err := handler.SendFileMessage(context.Background(), SendFileMessageInput{
		ChatID:      42,
		SenderLogin: "alice",
		Files: []FileUpload{
			{Filename: "doc.pdf", Reader: nopReadSeekCloser{strings.NewReader("a")}},
		},
	})
	if !errors.Is(err, ErrSaveFileMessage) {
		t.Fatalf("expected save error, got %v", err)
	}
	if len(fileClient.deleteCalls) != 1 || fileClient.deleteCalls[0] != "file-1" {
		t.Fatalf("expected compensation delete, got %v", fileClient.deleteCalls)
	}
}
