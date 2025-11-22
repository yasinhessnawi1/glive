package audit

// Audit event type constants
const (
	EventTypeProjectClone      = "project.clone"
	EventTypeProjectAnalyze    = "project.analyze"
	EventTypeCommandExecute    = "command.execute"
	EventTypeConfigChange      = "config.change"
	EventTypeSecurityViolation = "security.violation"
)

// Result constants
const (
	ResultSuccess = "success"
	ResultFailure = "failure"
	ResultWarning = "warning"
)

// Helper functions for common audit events

// LogProjectClone logs a project clone event
func LogProjectClone(auditor Auditor, projectID, githubURL string, success bool) error {
	result := ResultSuccess
	if !success {
		result = ResultFailure
	}
	return auditor.LogEvent(
		EventTypeProjectClone,
		projectID,
		"clone",
		result,
		map[string]interface{}{
			"github_url": githubURL,
		},
	)
}

// LogProjectAnalyze logs a project analysis event
func LogProjectAnalyze(auditor Auditor, projectID string, success bool) error {
	result := ResultSuccess
	if !success {
		result = ResultFailure
	}
	return auditor.LogEvent(
		EventTypeProjectAnalyze,
		projectID,
		"analyze",
		result,
		nil,
	)
}

// LogCommandExecute logs a command execution event
func LogCommandExecute(auditor Auditor, projectID, command string, success bool) error {
	result := ResultSuccess
	if !success {
		result = ResultFailure
	}
	return auditor.LogEvent(
		EventTypeCommandExecute,
		projectID,
		"execute",
		result,
		map[string]interface{}{
			"command": command,
		},
	)
}

// LogConfigChange logs a configuration change event
func LogConfigChange(auditor Auditor, configKey string, success bool) error {
	result := ResultSuccess
	if !success {
		result = ResultFailure
	}
	return auditor.LogEvent(
		EventTypeConfigChange,
		"config",
		"change",
		result,
		map[string]interface{}{
			"key": configKey,
		},
	)
}

// LogSecurityViolation logs a security violation event
func LogSecurityViolation(auditor Auditor, projectID, violationType, reason string) error {
	return auditor.LogEvent(
		EventTypeSecurityViolation,
		projectID,
		"violation",
		ResultWarning,
		map[string]interface{}{
			"type":   violationType,
			"reason": reason,
		},
	)
}


