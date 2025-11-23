package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// Agent represents an autonomous AI agent
type Agent struct {
	client       *Client
	toolExecutor *ToolExecutor
	maxSteps     int
	history      []Message
}

// NewAgent creates a new agent
func NewAgent(client *Client, workingDir string) *Agent {
	return &Agent{
		client:       client,
		toolExecutor: NewToolExecutor(workingDir),
		maxSteps:     10, // Limit steps to prevent infinite loops
		history:      make([]Message, 0),
	}
}

// AgentResult represents the final outcome of the agent's work
type AgentResult struct {
	Success            bool     `json:"success"`
	Message            string   `json:"message"`
	ManualInstructions []string `json:"manual_instructions,omitempty"`
}

// Run executes the agent loop
func (a *Agent) Run(ctx context.Context, goal string) (*AgentResult, error) {
	// Initialize system prompt
	a.history = []Message{
		{
			Role:    "system",
			Content: a.getSystemPrompt(),
		},
		{
			Role:    "user",
			Content: fmt.Sprintf("Goal: %s", goal),
		},
	}

	for i := 0; i < a.maxSteps; i++ {
		// 1. Get AI response
		response, err := a.client.ChatWithContext(ctx, a.history)
		if err != nil {
			return nil, fmt.Errorf("AI chat failed: %w", err)
		}

		// Add assistant response to history
		a.history = append(a.history, Message{
			Role:    "assistant",
			Content: response,
		})

		// 2. Parse response for tool calls or final answer
		toolCall, finalResult := a.parseResponse(response)

		// 3. If final answer, return it
		if finalResult != nil {
			return finalResult, nil
		}

		// 4. If tool call, execute it
		if toolCall != nil {
			result := a.toolExecutor.Execute(ctx, *toolCall)

			// Add tool result to history
			content := fmt.Sprintf("Tool Output:\n%s", result.Output)
			if result.Error != "" {
				content = fmt.Sprintf("Tool Error:\n%s", result.Error)
			}

			a.history = append(a.history, Message{
				Role:    "user",
				Content: content,
			})
		} else {
			// No tool call and no final result? Ask AI to continue
			a.history = append(a.history, Message{
				Role:    "user",
				Content: "Please continue. Use a tool or provide a final answer.",
			})
		}
	}

	return &AgentResult{
		Success: false,
		Message: "Agent reached maximum steps without resolution",
	}, nil
}

func (a *Agent) getSystemPrompt() string {
	return `You are an autonomous AI agent capable of fixing software issues.
You have access to the following tools:

1. run_command(command): Execute a shell command.
2. read_file(path): Read the content of a file.
3. list_dir(path): List files in a directory.
4. read_docs(path): Find and read all documentation (.md files) in a path.
5. edit_file(path, content): Overwrite a file with new content.

RESPONSE FORMAT:
You must respond with a JSON object. Do not include any other text.

Option 1: Execute a Tool
{
  "action": "tool",
  "tool": "tool_name",
  "arguments": {
    "arg_name": "arg_value"
  },
  "reasoning": "Why I am doing this"
}

Option 2: Final Answer (Success)
{
  "action": "final",
  "success": true,
  "message": "I have fixed the issue by...",
  "reasoning": "Explanation of the fix"
}

Option 3: Final Answer (User Action Required)
{
  "action": "final",
  "success": false,
  "message": "I cannot fix this automatically because...",
  "manual_instructions": [
    "Step 1: Go to...",
    "Step 2: Click..."
  ],
  "reasoning": "Why user intervention is needed"
}

STRATEGY:
1. EXPLORE: Use list_dir and read_file to understand the context.
2. DIAGNOSE: Use run_command to reproduce errors or check versions.
3. FIX: Use edit_file or run_command to apply fixes.
4. VERIFY: Verify the fix worked.
5. GUIDE: If you hit a hard constraint (e.g., missing API key, 2FA), stop and provide manual_instructions.

CRITICAL:
- Always check if a file exists before reading it.
- If an error says "command not found", check what IS installed.
- Do not guess file paths.
`
}

type agentResponse struct {
	Action             string            `json:"action"`
	Tool               string            `json:"tool"`
	Arguments          map[string]string `json:"arguments"`
	Success            bool              `json:"success"`
	Message            string            `json:"message"`
	ManualInstructions []string          `json:"manual_instructions"`
	Reasoning          string            `json:"reasoning"`
}

func (a *Agent) parseResponse(response string) (*ToolCall, *AgentResult) {
	// Clean up response (remove markdown code blocks if present)
	cleanResp := strings.TrimSpace(response)
	if strings.HasPrefix(cleanResp, "```json") {
		cleanResp = strings.TrimPrefix(cleanResp, "```json")
		cleanResp = strings.TrimSuffix(cleanResp, "```")
	} else if strings.HasPrefix(cleanResp, "```") {
		cleanResp = strings.TrimPrefix(cleanResp, "```")
		cleanResp = strings.TrimSuffix(cleanResp, "```")
	}
	cleanResp = strings.TrimSpace(cleanResp)

	var resp agentResponse
	if err := json.Unmarshal([]byte(cleanResp), &resp); err != nil {
		// If parsing fails, return a "fake" tool result to prompt retry
		// In a real system, we might want to handle this more gracefully
		return nil, nil
	}

	if resp.Action == "tool" {
		return &ToolCall{
			Type:      ToolType(resp.Tool),
			Arguments: resp.Arguments,
		}, nil
	}

	if resp.Action == "final" {
		return nil, &AgentResult{
			Success:            resp.Success,
			Message:            resp.Message,
			ManualInstructions: resp.ManualInstructions,
		}
	}

	return nil, nil
}
