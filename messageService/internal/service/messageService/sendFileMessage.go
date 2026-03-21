package messageService

import (
	"context"
	"errors"
	"fmt"
	"log"
	fileServiceClient "messageService/internal/client/fileService"
	"messageService/internal/models"
	"messageService/internal/storage/postgres/messageStorage"
	"strings"
)

const (
	maxFilesPerMessage  = 10
	messageKindText     = "text"
	messageKindFile     = "file"
	attachmentKindFile  = "file"
	attachmentKindImage = "image"
)

var (
	ErrNoFiles                = errors.New("no files provided")
	ErrTooManyFiles           = errors.New("too many files")
	ErrChatNotFound           = errors.New("chat not found")
	ErrSenderNotMember        = errors.New("sender not member")
	ErrFileValidation         = errors.New("file validation failed")
	ErrFileServiceUnavailable = errors.New("file service unavailable")
	ErrSaveFileMessage        = errors.New("save file message failed")
)

type FileServiceClient interface {
	Upload(ctx context.Context, request fileServiceClient.UploadRequest) (*fileServiceClient.UploadedFile, error)
	Delete(ctx context.Context, fileID string) error
}

type SendFileMessageInput struct {
	ChatID      int
	SenderLogin string
	Files       []FileUpload
}

type FileUpload struct {
	Filename    string
	ContentType string
	Size        int64
	Reader      fileServiceClient.ReadSeekCloser
}

func (m *messageHandler) SendFileMessage(ctx context.Context, input SendFileMessageInput) (*models.EventSendMessagePayload, error) {
	log.Printf("[messageService][SendFileMessage] start chat_id=%d sender=%s files=%d", input.ChatID, input.SenderLogin, len(input.Files))
	if len(input.Files) == 0 {
		log.Printf("[messageService][SendFileMessage] rejected_no_files chat_id=%d sender=%s", input.ChatID, input.SenderLogin)
		return nil, ErrNoFiles
	}
	if len(input.Files) > maxFilesPerMessage {
		log.Printf("[messageService][SendFileMessage] rejected_too_many_files chat_id=%d sender=%s files=%d", input.ChatID, input.SenderLogin, len(input.Files))
		return nil, ErrTooManyFiles
	}

	chatExists, err := m.MessageStorage.ChatExists(ctx, input.ChatID)
	if err != nil {
		log.Printf("[messageService][SendFileMessage] chat_exists_error chat_id=%d sender=%s err=%v", input.ChatID, input.SenderLogin, err)
		return nil, fmt.Errorf("[SendFileMessage] error: %w", err)
	}
	if !chatExists {
		log.Printf("[messageService][SendFileMessage] chat_not_found chat_id=%d sender=%s", input.ChatID, input.SenderLogin)
		return nil, ErrChatNotFound
	}

	isMember, err := m.MessageStorage.IsChatMember(ctx, input.ChatID, input.SenderLogin)
	if err != nil {
		log.Printf("[messageService][SendFileMessage] membership_check_error chat_id=%d sender=%s err=%v", input.ChatID, input.SenderLogin, err)
		return nil, fmt.Errorf("[SendFileMessage] error: %w", err)
	}
	if !isMember {
		log.Printf("[messageService][SendFileMessage] sender_not_member chat_id=%d sender=%s", input.ChatID, input.SenderLogin)
		return nil, ErrSenderNotMember
	}
	log.Printf("[messageService][SendFileMessage] membership_confirmed chat_id=%d sender=%s", input.ChatID, input.SenderLogin)

	uploadedFiles := make([]*fileServiceClient.UploadedFile, 0, len(input.Files))
	defer closeUploads(input.Files)

	for index, file := range input.Files {
		log.Printf("[messageService][SendFileMessage] upload_start chat_id=%d sender=%s index=%d filename=%q size=%d content_type=%q", input.ChatID, input.SenderLogin, index, file.Filename, file.Size, file.ContentType)
		uploadedFile, uploadErr := m.FileService.Upload(ctx, fileServiceClient.UploadRequest{
			Filename:     file.Filename,
			Content:      file.Reader,
			Uploader:     input.SenderLogin,
			OwnerService: "messageService",
		})
		if uploadErr != nil {
			log.Printf("[messageService][SendFileMessage] upload_error chat_id=%d sender=%s index=%d filename=%q err=%v uploaded_before_error=%d", input.ChatID, input.SenderLogin, index, file.Filename, uploadErr, len(uploadedFiles))
			m.compensateUploadedFiles(ctx, uploadedFiles)
			if errors.Is(uploadErr, fileServiceClient.ErrValidation) {
				return nil, ErrFileValidation
			}
			if errors.Is(uploadErr, fileServiceClient.ErrUnavailable) {
				return nil, ErrFileServiceUnavailable
			}
			return nil, fmt.Errorf("[SendFileMessage] error: %w", uploadErr)
		}
		log.Printf("[messageService][SendFileMessage] upload_success chat_id=%d sender=%s index=%d filename=%q file_id=%s mime_type=%s size=%d", input.ChatID, input.SenderLogin, index, file.Filename, uploadedFile.ID, uploadedFile.MimeType, uploadedFile.Size)
		uploadedFiles = append(uploadedFiles, uploadedFile)
	}

	attachments := make([]messageStorage.FileAttachmentRecord, 0, len(uploadedFiles))
	for _, uploadedFile := range uploadedFiles {
		attachments = append(attachments, messageStorage.FileAttachmentRecord{
			FileID:           uploadedFile.ID,
			OriginalFilename: uploadedFile.OriginalFilename,
			MimeType:         uploadedFile.MimeType,
			Size:             uploadedFile.Size,
			Kind:             detectAttachmentKind(uploadedFile.MimeType),
		})
	}
	log.Printf("[messageService][SendFileMessage] db_create_start chat_id=%d sender=%s attachments=%d", input.ChatID, input.SenderLogin, len(attachments))

	payload, err := m.MessageStorage.CreateFileMessage(ctx, input.ChatID, input.SenderLogin, attachments)
	if err != nil {
		log.Printf("[messageService][SendFileMessage] db_create_error chat_id=%d sender=%s err=%v uploaded_files=%d", input.ChatID, input.SenderLogin, err, len(uploadedFiles))
		m.compensateUploadedFiles(ctx, uploadedFiles)
		return nil, ErrSaveFileMessage
	}
	log.Printf("[messageService][SendFileMessage] db_create_success chat_id=%d sender=%s message_id=%d", input.ChatID, input.SenderLogin, payload.Id)

	enrichMessagePayload(payload)
	log.Printf("[messageService][SendFileMessage] payload_enriched chat_id=%d sender=%s message_id=%d attachments=%d", input.ChatID, input.SenderLogin, payload.Id, len(payload.Attachments))

	if err := m.fanOutMessage(ctx, payload); err != nil {
		log.Printf("[SendFileMessage] fanout error: %v\n", err)
	} else {
		log.Printf("[messageService][SendFileMessage] fanout_success chat_id=%d sender=%s message_id=%d", input.ChatID, input.SenderLogin, payload.Id)
	}

	log.Printf("[messageService][SendFileMessage] done chat_id=%d sender=%s message_id=%d", input.ChatID, input.SenderLogin, payload.Id)
	return payload, nil
}

func (m *messageHandler) compensateUploadedFiles(ctx context.Context, uploadedFiles []*fileServiceClient.UploadedFile) {
	for _, uploadedFile := range uploadedFiles {
		if uploadedFile == nil {
			continue
		}
		log.Printf("[messageService][SendFileMessage] compensation_delete_start file_id=%s", uploadedFile.ID)
		if err := m.FileService.Delete(ctx, uploadedFile.ID); err != nil {
			log.Printf("[messageService][SendFileMessage] compensation_delete_error file_id=%s err=%v", uploadedFile.ID, err)
			continue
		}
		log.Printf("[messageService][SendFileMessage] compensation_delete_success file_id=%s", uploadedFile.ID)
	}
}

func detectAttachmentKind(mimeType string) string {
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(mimeType)), "image/") {
		return attachmentKindImage
	}
	return attachmentKindFile
}

func closeUploads(files []FileUpload) {
	for _, file := range files {
		if file.Reader != nil {
			_ = file.Reader.Close()
		}
	}
}
