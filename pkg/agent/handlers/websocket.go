package handlers

import (
	"encoding/json"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/glive/core/types"
	"github.com/glive/interface/api"
	"github.com/gofiber/websocket/v2"
)

// eventToJSON converts an event to JSON bytes
func (h *Handler) eventToJSON(event interface{}) ([]byte, error) {
	// Check if it's already a WSMessage
	if wsMsg, ok := event.(*api.WSMessage); ok {
		return wsMsg.ToJSON()
	}

	if update, ok := event.(types.ProgressUpdate); ok {
		// Convert types.ProgressUpdate to WSMessage with correct payload structure
		switch update.Stage {
		case "command_started":
			// Command started event
			wsMsg := api.NewCommandStartedMessage(
				update.ProjectID,
				update.CommandID,
				update.Command,
				update.Timestamp.UnixMilli(),
			)
			return wsMsg.ToJSON()

		case "command_complete":
			// Command completed event
			wsMsg := api.NewCommandCompleteMessage(
				update.ProjectID,
				update.CommandID,
				update.ExitCode,
				update.Success,
				update.Message,
				update.Duration,
			)
			return wsMsg.ToJSON()

		case "execution_completed":
			// Execution completed event
			status := "success"
			if !update.Success {
				status = "failed"
			}
			wsMsg := api.NewExecutionCompletedMessage(
				update.ProjectID,
				status,
				update.Message,
			)
			return wsMsg.ToJSON()

		case "running", "log":
			// This is a log line - send as command.output
			stream := "stdout"
			message := update.Message

			// Check if it's an error message
			if strings.Contains(message, "⚠") || strings.Contains(message, "❌") || strings.Contains(message, "[ERROR]") {
				stream = "stderr"
			}

			wsMsg := api.NewCommandOutputMessage(
				update.ProjectID,
				update.CommandID,
				stream,
				message,
			)
			return wsMsg.ToJSON()

		default:
			// This is a status update - send as project.status
			wsMsg := api.NewProjectStatusMessage(
				update.ProjectID,
				update.Stage,
				update.Message,
			)
			return wsMsg.ToJSON()
		}
	}

	return nil, nil
}

// HandleWebSocket handles WebSocket connections for real-time updates
func (h *Handler) HandleWebSocket(c *websocket.Conn) {
	projectID := c.Params("project_id")

	log.Printf("WebSocket connected for project: %s", projectID)

	// Set read/write deadlines and enable ping/pong
	c.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.SetPongHandler(func(string) error {
		c.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// Mutex to prevent concurrent writes to WebSocket
	var writeMutex sync.Mutex

	defer func() {
		c.Close()
		log.Printf("WebSocket closed for project: %s", projectID)
	}()

	// Helper function for safe writes
	safeWrite := func(messageType int, data []byte) error {
		writeMutex.Lock()
		defer writeMutex.Unlock()
		c.SetWriteDeadline(time.Now().Add(10 * time.Second))
		return c.WriteMessage(messageType, data)
	}

	// Send welcome message with current project status
	welcomeMsg := api.NewProjectStatusMessage(projectID, "connected", "WebSocket connection established")
	if msgBytes, err := welcomeMsg.ToJSON(); err == nil {
		if err := safeWrite(websocket.TextMessage, msgBytes); err != nil {
			log.Printf("Failed to send welcome message: %v", err)
			return
		}
	}

	// Replay buffered events (for reconnection scenarios)
	bufferedEvents := h.EventBus.GetBufferedEvents(projectID)
	if len(bufferedEvents) > 0 {
		for _, event := range bufferedEvents {
			msgBytes, err := h.eventToJSON(event)
			if err != nil || msgBytes == nil {
				continue
			}
			if err := safeWrite(websocket.TextMessage, msgBytes); err != nil {
				log.Printf("Failed to send buffered event: %v", err)
				return
			}
		}
	}

	// Subscribe to project events
	eventChan := make(chan interface{}, 100)
	h.EventBus.Subscribe(projectID, eventChan)
	defer h.EventBus.Unsubscribe(projectID, eventChan)

	// Channel to signal reader goroutine to stop
	done := make(chan struct{})

	// Start ping ticker for keepalive
	pingTicker := time.NewTicker(30 * time.Second)
	defer pingTicker.Stop()

	// Handle incoming messages (keep alive / commands)
	go func() {
		defer close(done)
		for {
			messageType, msg, err := c.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("WebSocket error: %v", err)
				}
				return
			}

			// Reset read deadline on any message
			c.SetReadDeadline(time.Now().Add(60 * time.Second))

			// Parse message to check type
			var msgData map[string]interface{}
			if err := json.Unmarshal(msg, &msgData); err == nil {
				msgType, ok := msgData["type"].(string)
				if ok && msgType == "ping" {
					// Respond to ping with pong
					pongMsg := map[string]string{"type": "pong"}
					if pongBytes, err := json.Marshal(pongMsg); err == nil {
						if err := safeWrite(messageType, pongBytes); err != nil {
							log.Printf("Failed to send pong: %v", err)
							return
						}
					}
					continue
				}
			}

			// Don't log every message to reduce noise
		}
	}()

	// Forward events to WebSocket
	for {
		select {
		case <-pingTicker.C:
			// Send WebSocket ping to keep connection alive
			writeMutex.Lock()
			c.SetWriteDeadline(time.Now().Add(10 * time.Second))
			err := c.WriteMessage(websocket.PingMessage, nil)
			writeMutex.Unlock()
			if err != nil {
				log.Printf("Failed to send ping: %v", err)
				return
			}

		case event, ok := <-eventChan:
			if !ok {
				// Channel closed, exit
				return
			}

			msgBytes, err := h.eventToJSON(event)
			if err != nil || msgBytes == nil {
				continue
			}

			if err := safeWrite(websocket.TextMessage, msgBytes); err != nil {
				log.Printf("Failed to send event: %v", err)
				return
			}

		case <-done:
			// Reader goroutine exited, close connection
			return
		}
	}
}
