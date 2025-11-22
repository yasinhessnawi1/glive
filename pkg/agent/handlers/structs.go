package handlers

import (
	"github.com/glive/agent/events"
	"github.com/glive/core"
	"github.com/glive/core/state"
)

// Handler holds dependencies for API handlers
type Handler struct {
	StateManager *state.Manager
	Orchestrator *core.Orchestrator
	EventBus     *events.Bus
}

// New creates a new Handler instance
func New(stateMgr *state.Manager, orchestrator *core.Orchestrator, eventBus *events.Bus) *Handler {
	return &Handler{
		StateManager: stateMgr,
		Orchestrator: orchestrator,
		EventBus:     eventBus,
	}
}
