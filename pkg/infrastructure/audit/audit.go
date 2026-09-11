package audit

import (
	"encoding/json"
	"io"
	"os"
	"sync"
	"time"
)

// AuditEvent represents a security audit event
type AuditEvent struct {
	Timestamp time.Time              `json:"timestamp"`
	EventType string                 `json:"event_type"`
	UserID    string                 `json:"user_id,omitempty"`
	Resource  string                 `json:"resource"`
	Action    string                 `json:"action"`
	Result    string                 `json:"result"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// Auditor is the interface for audit logging
type Auditor interface {
	Log(event AuditEvent) error
	LogEvent(eventType, resource, action, result string, metadata map[string]interface{}) error
}

// FileAuditor implements audit logging to a file
type FileAuditor struct {
	writer io.Writer
	mu     sync.Mutex
}

// NewFileAuditor creates a new file-based auditor
func NewFileAuditor(writer io.Writer) *FileAuditor {
	if writer == nil {
		writer = os.Stderr // Default to stderr for audit logs
	}
	return &FileAuditor{writer: writer}
}

// Log writes an audit event to the log
func (a *FileAuditor) Log(event AuditEvent) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Ensure timestamp is set
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	// Serialize to JSON
	jsonData, err := json.Marshal(event)
	if err != nil {
		return err
	}

	jsonData = append(jsonData, '\n')
	_, err = a.writer.Write(jsonData)
	return err
}

// LogEvent is a convenience method for logging audit events
func (a *FileAuditor) LogEvent(eventType, resource, action, result string, metadata map[string]interface{}) error {
	event := AuditEvent{
		Timestamp: time.Now().UTC(),
		EventType: eventType,
		Resource:  resource,
		Action:    action,
		Result:    result,
		Metadata:  metadata,
	}
	return a.Log(event)
}

// NoOpAuditor is a no-op implementation for when audit logging is disabled
type NoOpAuditor struct{}

// Log does nothing
func (n *NoOpAuditor) Log(event AuditEvent) error {
	return nil
}

// LogEvent does nothing
func (n *NoOpAuditor) LogEvent(eventType, resource, action, result string, metadata map[string]interface{}) error {
	return nil
}
