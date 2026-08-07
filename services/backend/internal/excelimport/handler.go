package excelimport

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/al-hudhud/backend/internal/auth"
	"github.com/al-hudhud/backend/internal/models"
	"github.com/al-hudhud/backend/internal/queue"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// BulkIngester is the slice of receiver.Service used by this feature. Keeping
// it as an interface isolates this package and makes it easily testable.
type BulkIngester interface {
	IngestGlobalEvent(ctx context.Context, tenantID uuid.UUID, senderIP string, rawJSON string) (*models.GlobalIncomingEvent, error)
}

type Handler struct {
	ingester BulkIngester
	store    *Store
}

func NewHandler(ingester BulkIngester, store *Store) *Handler {
	return &Handler{ingester: ingester, store: store}
}

// Upload receives a multipart .xlsx/.csv file, parses and normalizes it on the
// server, and returns a preview to the app for confirmation. Nothing is
// ingested here — the actual ingestion only happens on Confirm.
func (h *Handler) Upload(c *gin.Context) {
	// Middleware already authenticated the tenant; abort if it did not.
	if _, ok := auth.GetTenantIDOrAbort(c); !ok {
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file field required (multipart field: file)"})
		return
	}
	defer file.Close()

	if header.Size > MaxFileBytes {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file exceeds the 5MB limit"})
		return
	}

	data, err := io.ReadAll(io.LimitReader(file, MaxFileBytes+1))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read file"})
		return
	}
	if len(data) > MaxFileBytes {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file exceeds the 5MB limit"})
		return
	}

	ext := strings.ToLower(filepath.Ext(header.Filename))
	rows, err := Parse(bytes.NewReader(data), ext)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(rows) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrNoDataRows.Error()})
		return
	}
	if len(rows) > MaxRows {
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrTooManyRows.Error()})
		return
	}

	seen := make(map[string]struct{}, len(rows))
	previewRows := make([]PreviewRow, 0, len(rows))
	valid, invalid, duplicates := 0, 0, 0
	for _, r := range rows {
		phone, ok := Normalize(r.Phone)
		if !ok || strings.TrimSpace(r.Content) == "" {
			invalid++
			reason := "رقم غير صالح"
			if ok {
				reason = "لا يوجد نص رسالة"
			}
			previewRows = append(previewRows, PreviewRow{Phone: r.Phone, Content: r.Content, Valid: false, Reason: reason})
			continue
		}
		if _, dup := seen[phone]; dup {
			duplicates++
			continue
		}
		seen[phone] = struct{}{}
		valid++

		content := r.Content
		if runes := []rune(content); len(runes) > MaxContentLen {
			content = string(runes[:MaxContentLen])
		}
		previewRows = append(previewRows, PreviewRow{Phone: phone, Content: content, Valid: true})
	}

	if valid == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no valid rows found in file"})
		return
	}

	preview := h.store.Set(Preview{
		FileName:   header.Filename,
		Rows:       previewRows,
		TotalRows:  len(rows),
		Valid:      valid,
		Invalid:    invalid,
		Duplicates: duplicates,
		CreatedAt:  time.Now(),
	})

	sample := previewRows
	if len(sample) > PreviewSample {
		sample = sample[:PreviewSample]
	}

	c.JSON(http.StatusOK, gin.H{
		"token":      preview.Token,
		"file_name":  header.Filename,
		"total_rows": preview.TotalRows,
		"valid":      preview.Valid,
		"invalid":    preview.Invalid,
		"duplicates": preview.Duplicates,
		"sample":     sample,
	})
}

// Confirm validates the preview token and pushes the accepted rows through the
// existing global ingestion pipeline (global_incoming_events -> messages -> phone).
func (h *Handler) Confirm(c *gin.Context) {
	tenantID, ok := auth.GetTenantIDOrAbort(c)
	if !ok {
		return
	}

	var req struct {
		Token string `json:"token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Token) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token required"})
		return
	}

	preview, ok := h.store.Get(strings.TrimSpace(req.Token))
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "preview expired or not found"})
		return
	}

	payload := queue.BulkMessagePayload{
		MessageType: "EXCEL_IMPORT",
		SenderName:  "Mobile App",
	}
	for _, r := range preview.Rows {
		if !r.Valid {
			continue
		}
		payload.Messages = append(payload.Messages, queue.SingleMessageItem{
			PhoneNumber: r.Phone,
			Content:     r.Content,
		})
	}
	if len(payload.Messages) == 0 {
		h.store.Delete(req.Token)
		c.JSON(http.StatusBadRequest, gin.H{"error": "no valid messages to send"})
		return
	}

	rawJSON, err := json.Marshal(payload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to prepare payload"})
		return
	}

	event, err := h.ingester.IngestGlobalEvent(c, tenantID, c.ClientIP(), string(rawJSON))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to ingest Excel data"})
		return
	}
	h.store.Delete(req.Token)

	c.JSON(http.StatusAccepted, gin.H{
		"message":  "Excel data received successfully",
		"event_id": event.CustomID,
		"status":   event.Status,
		"count":    len(payload.Messages),
	})
}
