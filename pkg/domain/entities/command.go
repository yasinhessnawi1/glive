package entities

import (
	"fmt"
	"time"

	"github.com/glive/domain/errors"
	"github.com/glive/domain/values"
)

// Command represents a command to be executed
type Command struct {
	id          string
	description string
	command     string
	workingDir  *values.Path
	stage       CommandStage
	required    bool
	status      CommandStatus
	output      string
	errorMsg    string
	exitCode    int
	executedAt  *time.Time
}

// CommandStage represents the stage of command execution
type CommandStage string

const (
	StageSetup CommandStage = "setup"
	StageBuild CommandStage = "build"
	StageRun   CommandStage = "run"
)

// CommandStatus represents the execution status of a command
type CommandStatus string

const (
	CommandPending   CommandStatus = "pending"
	CommandRunning   CommandStatus = "running"
	CommandCompleted CommandStatus = "completed"
	CommandFailed    CommandStatus = "failed"
	CommandSkipped   CommandStatus = "skipped"
)

// NewCommand creates a new Command entity
func NewCommand(id, description, command string, workingDir *values.Path, stage CommandStage, required bool) (*Command, error) {
	if id == "" {
		return nil, errors.NewDomainError(errors.ErrCodeInvalidInput, "command ID cannot be empty")
	}
	if description == "" {
		return nil, errors.NewDomainError(errors.ErrCodeInvalidInput, "command description cannot be empty")
	}
	if command == "" {
		return nil, errors.NewDomainError(errors.ErrCodeInvalidInput, "command cannot be empty")
	}
	if workingDir == nil {
		return nil, errors.NewDomainError(errors.ErrCodeInvalidInput, "working directory cannot be nil")
	}

	return &Command{
		id:          id,
		description: description,
		command:     command,
		workingDir:  workingDir,
		stage:       stage,
		required:    required,
		status:      CommandPending,
		exitCode:    -1,
	}, nil
}

// ID returns the command ID
func (c *Command) ID() string {
	return c.id
}

// Description returns the command description
func (c *Command) Description() string {
	return c.description
}

// Command returns the command string
func (c *Command) Command() string {
	return c.command
}

// WorkingDir returns the working directory
func (c *Command) WorkingDir() *values.Path {
	return c.workingDir
}

// Stage returns the command stage
func (c *Command) Stage() CommandStage {
	return c.stage
}

// Required returns whether the command is required
func (c *Command) Required() bool {
	return c.required
}

// Status returns the command status
func (c *Command) Status() CommandStatus {
	return c.status
}

// Output returns the command output
func (c *Command) Output() string {
	return c.output
}

// Error returns the command error message
func (c *Command) Error() string {
	return c.errorMsg
}

// ExitCode returns the command exit code
func (c *Command) ExitCode() int {
	return c.exitCode
}

// ExecutedAt returns when the command was executed
func (c *Command) ExecutedAt() *time.Time {
	return c.executedAt
}

// MarkRunning marks the command as running
func (c *Command) MarkRunning() {
	c.status = CommandRunning
}

// MarkCompleted marks the command as completed
func (c *Command) MarkCompleted(output string) {
	c.status = CommandCompleted
	c.output = output
	c.exitCode = 0
	now := time.Now()
	c.executedAt = &now
}

// MarkFailed marks the command as failed
func (c *Command) MarkFailed(output, errorMsg string, exitCode int) {
	c.status = CommandFailed
	c.output = output
	c.errorMsg = errorMsg
	c.exitCode = exitCode
	now := time.Now()
	c.executedAt = &now
}

// MarkSkipped marks the command as skipped
func (c *Command) MarkSkipped() {
	c.status = CommandSkipped
}

// IsCompleted checks if the command completed successfully
func (c *Command) IsCompleted() bool {
	return c.status == CommandCompleted
}

// IsFailed checks if the command failed
func (c *Command) IsFailed() bool {
	return c.status == CommandFailed
}

// IsRunning checks if the command is currently running
func (c *Command) IsRunning() bool {
	return c.status == CommandRunning
}

// CanRetry checks if the command can be retried
func (c *Command) CanRetry() bool {
	return c.status == CommandFailed && c.required
}

// UpdateOutput appends output to the command
func (c *Command) UpdateOutput(line string) {
	if c.output == "" {
		c.output = line
	} else {
		c.output = fmt.Sprintf("%s\n%s", c.output, line)
	}
}
