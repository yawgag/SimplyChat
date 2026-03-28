package messageStorage

import (
	"messageService/internal/models"
	"testing"
)

func TestAggregateHistoryRowsKeepsSingleMessageWithMultipleAttachments(t *testing.T) {
	fileID1 := "file-1"
	fileID2 := "file-2"
	filename1 := "photo.png"
	filename2 := "doc.pdf"
	mime1 := "image/png"
	mime2 := "application/pdf"
	size1 := int64(10)
	size2 := int64(20)
	kind1 := "image"
	kind2 := "file"

	rows := []historyRow{
		{
			messageID:        1,
			chatID:           42,
			senderLogin:      "alice",
			kind:             "file",
			content:          "",
			fileID:           &fileID1,
			originalFilename: &filename1,
			mimeType:         &mime1,
			size:             &size1,
			attachmentKind:   &kind1,
		},
		{
			messageID:        1,
			chatID:           42,
			senderLogin:      "alice",
			kind:             "file",
			content:          "",
			fileID:           &fileID2,
			originalFilename: &filename2,
			mimeType:         &mime2,
			size:             &size2,
			attachmentKind:   &kind2,
		},
	}

	history := aggregateHistoryRows(rows)
	if len(history) != 1 {
		t.Fatalf("expected 1 message, got %d", len(history))
	}
	if len(history[0].Attachments) != 2 {
		t.Fatalf("expected 2 attachments, got %d", len(history[0].Attachments))
	}
}

func TestReverseHistory(t *testing.T) {
	history := []models.EventSendMessagePayload{
		{Id: 3},
		{Id: 2},
		{Id: 1},
	}

	reverseHistory(history)

	if history[0].Id != 1 || history[1].Id != 2 || history[2].Id != 3 {
		t.Fatalf("unexpected order after reverse: %+v", history)
	}
}
