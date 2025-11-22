package handlers

import (
	"log"

	"github.com/glive/core/types"
	"github.com/glive/interface/api"
	"github.com/gofiber/websocket/v2"
)

// HandleWebSocket handles WebSocket connections for real-time updates
func (h *Handler) HandleWebSocket(c *websocket.Conn) {
	projectID := c.Params("project_id")

	log.Printf("WebSocket connected for project: %s", projectID)

	defer func() {
		c.Close()
		log.Printf("WebSocket closed for project: %s", projectID)
	}()

	// Send welcome message with current project status
	welcomeMsg := api.NewProjectStatusMessage(projectID, "connected", "WebSocket connection established")
	if msgBytes, err := welcomeMsg.ToJSON(); err == nil {
		if err := c.WriteMessage(websocket.TextMessage, msgBytes); err != nil {
			log.Printf("Failed to send welcome message: %v", err)
			return
		}
	}

	// Subscribe to project events
	eventChan := make(chan interface{}, 100)
	h.EventBus.Subscribe(projectID, eventChan)
	defer h.EventBus.Unsubscribe(projectID, eventChan)

	// Handle incoming messages (keep alive / commands)
	go func() {
		for {
			messageType, msg, err := c.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("WebSocket error: %v", err)
				}
				// Close channel to stop the writer loop
				close(eventChan)
				break
			}

			log.Printf("Received message: %s", msg)

			// For now, just acknowledge receipt
			ackMsg := api.NewProjectStatusMessage(projectID, "message_received", "Message acknowledged")
			h.EventBus.Publish(projectID, ackMsg)

			// If we need to write back to this specific connection immediately, we can do it here
			// but better to go through the event bus if it's a broadcast
			if msgBytes, err := ackMsg.ToJSON(); err == nil {
				if err := c.WriteMessage(messageType, msgBytes); err != nil {
					log.Printf("Write error: %v", err)
					break
				}
			}
		}
	}()

	// Forward events to WebSocket
	for event := range eventChan {
		var msgBytes []byte
		var err error

		// Check if it's already a WSMessage
		if wsMsg, ok := event.(*api.WSMessage); ok {
			msgBytes, err = wsMsg.ToJSON()
		} else if update, ok := event.(types.ProgressUpdate); ok {
			// Convert types.ProgressUpdate to WSMessage with correct payload structure
			if update.Stage == "running" {
				// This is a log line - send as command.output
				wsMsg := api.NewCommandOutputMessage(
					update.ProjectID,
					update.CommandID,
					"stdout", // Assume stdout for now
					update.Message,
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

		if err := c.WriteMessage(websocket.TextMessage, msgBytes); err != nil {
			log.Printf("Failed to send event: %v", err)
			break
		}
	}
}
