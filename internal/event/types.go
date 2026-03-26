package event

import (
	"encoding/json"
	"time"
)

// SigningEvent is the event envelope published by the signing service
type SigningEvent struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	Source    string          `json:"source"`
	Timestamp time.Time       `json:"timestamp"`
	TenantID  uint32          `json:"tenant_id"`
	Data      json.RawMessage `json:"data"`
}

// SubmissionCompletedData is the data payload for submission.completed events
type SubmissionCompletedData struct {
	SubmissionID     string `json:"submission_id"`
	TemplateID       string `json:"template_id"`
	SignedDocumentKey string `json:"signed_document_key"`
	TenantID         uint32 `json:"tenant_id"`
}
