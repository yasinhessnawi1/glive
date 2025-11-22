package api

import (
	"encoding/json"
)

// WebSocket message type constants
const (
	WSMessageTypeProjectStatus = "project.status"
	WSMessageTypeCommandOutput = "command.output"
	WSMessageTypeCommandComplete = "command.complete"
	WSMessageTypeError = "error"
)

// WSMessage represents a standardized WebSocket message
type WSMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// NewWSMessage creates a new WebSocket message
func NewWSMessage(msgType string, payload interface{}) *WSMessage {
	return &WSMessage{
		Type:    msgType,
		Payload: payload,
	}
}

// ToJSON converts the message to JSON bytes
func (m *WSMessage) ToJSON() ([]byte, error) {
	return json.Marshal(m)
}

// ProjectStatusPayload represents payload for project.status messages
type ProjectStatusPayload struct {
	ProjectID string `json:"project_id"`
	Status    string `json:"status"`
	Message   string `json:"message,omitempty"`
}

// CommandOutputPayload represents payload for command.output messages
type CommandOutputPayload struct {
	ProjectID string `json:"project_id"`
	CommandID string `json:"command_id,omitempty"`
	Stream    string `json:"stream"` // "stdout" or "stderr"
	Output    string `json:"output"`
}

// CommandCompletePayload represents payload for command.complete messages
type CommandCompletePayload struct {
	ProjectID string `json:"project_id"`
	CommandID string `json:"command_id,omitempty"`
	ExitCode  int    `json:"exit_code"`
	Success   bool   `json:"success"`
	Message   string `json:"message,omitempty"`
}

// ErrorPayload represents payload for error messages
type ErrorPayload struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// NewProjectStatusMessage creates a project.status message
func NewProjectStatusMessage(projectID, status, message string) *WSMessage {
	return NewWSMessage(WSMessageTypeProjectStatus, ProjectStatusPayload{
		ProjectID: projectID,
		Status:    status,
		Message:   message,
	})
}

// NewCommandOutputMessage creates a command.output message
func NewCommandOutputMessage(projectID, commandID, stream, output string) *WSMessage {
	return NewWSMessage(WSMessageTypeCommandOutput, CommandOutputPayload{
		ProjectID: projectID,
		CommandID: commandID,
		Stream:    stream,
		Output:    output,
	})
}

// NewCommandCompleteMessage creates a command.complete message
func NewCommandCompleteMessage(projectID, commandID string, exitCode int, success bool, message string) *WSMessage {
	return NewWSMessage(WSMessageTypeCommandComplete, CommandCompletePayload{
		ProjectID: projectID,
		CommandID: commandID,
		ExitCode:  exitCode,
		Success:   success,
		Message:   message,
	})
}

// NewErrorMessage creates an error message
func NewErrorMessage(code, message string, details map[string]interface{}) *WSMessage {
	return NewWSMessage(WSMessageTypeError, ErrorPayload{
		Code:    code,
		Message: message,
		Details: details,
	})
}

