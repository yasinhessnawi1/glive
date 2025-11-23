package handlers

import (
	"encoding/json"
	"log"
	"strings"
	"sync"

	"github.com/glive/core/types"
	"github.com/glive/interface/api"
	"github.com/gofiber/websocket/v2"
)

// HandleWebSocket handles WebSocket connections for real-time updates
func (h *Handler) HandleWebSocket(c *websocket.Conn) {
	projectID := c.Params("project_id")

	log.Printf("WebSocket connected for project: %s", projectID)

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

	// Subscribe to project events
	eventChan := make(chan interface{}, 100)
	h.EventBus.Subscribe(projectID, eventChan)
	defer h.EventBus.Unsubscribe(projectID, eventChan)

	// Channel to signal reader goroutine to stop
	done := make(chan struct{})

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

			log.Printf("Received message: %s", msg)
			// Note: Removed automatic acknowledgment to prevent unnecessary traffic
		}
	}()

	// Forward events to WebSocket
	for {
		select {
		case event, ok := <-eventChan:
			if !ok {
				// Channel closed, exit
				return
			}

			var msgBytes []byte
			var err error

			// Check if it's already a WSMessage
			if wsMsg, ok := event.(*api.WSMessage); ok {
				msgBytes, err = wsMsg.ToJSON()
			} else if update, ok := event.(types.ProgressUpdate); ok {
				// Convert types.ProgressUpdate to WSMessage with correct payload structure
				if update.Stage == "running" || update.Stage == "log" {
					// This is a log line - send as command.output
					// Determine stream type based on message content
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
					msgBytes, err = wsMsg.ToJSON()
				} else {
					// This is a status update - send as project.status
					wsMsg := api.NewProjectStatusMessage(
						update.ProjectID,
						update.Stage,
						update.Message,
					)
					msgBytes, err = wsMsg.ToJSON()
				}
			} else {
				// Unknown type, skip
				continue
			}

			if err != nil {
				log.Printf("Failed to marshal event: %v", err)
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
