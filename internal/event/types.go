package event

import (
	"encoding/json"
	"time"
)

// SigningEvent is the event envelope published by the paperless service
type SigningEvent struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	Source    string          `json:"source"`
	Timestamp time.Time       `json:"timestamp"`
	TenantID  uint32          `json:"tenant_id"`
	Data      json.RawMessage `json:"data"`
}

// SigningRequestCompletedData is the data payload for signing.request.completed events
type SigningRequestCompletedData struct {
	RequestID    string `json:"request_id"`
	TemplateID   string `json:"template_id"`
	TemplateName string `json:"template_name"`
	SignedFileKey string `json:"signed_file_key"`
	TenantID     uint32 `json:"tenant_id"`
}
