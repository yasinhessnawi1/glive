package metrics

// Predefined metric names following Prometheus naming conventions
const (
	// Project metrics
	MetricProjectsTotal = "glive_projects_total"

	// Command execution metrics
	MetricCommandsExecutedTotal = "glive_commands_executed_total"

	// Duration metrics (histograms)
	MetricCloneDurationSeconds        = "glive_clone_duration_seconds"
	MetricAnalysisDurationSeconds     = "glive_analysis_duration_seconds"
	MetricAIRequestDurationSeconds    = "glive_ai_request_duration_seconds"

	// Error metrics
	MetricErrorsTotal = "glive_errors_total"

	// AI metrics
	MetricAIRequestsTotal = "glive_ai_requests_total"
)

// Instrumentation provides helper functions for common metrics
type Instrumentation struct {
	metrics Metrics
}

// NewInstrumentation creates a new instrumentation helper
func NewInstrumentation(metrics Metrics) *Instrumentation {
	return &Instrumentation{metrics: metrics}
}

// RecordProjectCreated records a project creation event
func (i *Instrumentation) RecordProjectCreated() {
	i.metrics.Counter(MetricProjectsTotal).Inc()
}

// RecordCommandExecuted records a command execution event
func (i *Instrumentation) RecordCommandExecuted(command string, success bool) {
	labels := map[string]string{
		"command": command,
		"status":  "success",
	}
	if !success {
		labels["status"] = "failure"
	}
	i.metrics.Counter(MetricCommandsExecutedTotal).WithLabels(labels).Inc()
}

// RecordCloneDuration records the duration of a clone operation
func (i *Instrumentation) RecordCloneDuration(durationSeconds float64) {
	i.metrics.Histogram(MetricCloneDurationSeconds).Observe(durationSeconds)
}

// RecordAnalysisDuration records the duration of an analysis operation
func (i *Instrumentation) RecordAnalysisDuration(durationSeconds float64) {
	i.metrics.Histogram(MetricAnalysisDurationSeconds).Observe(durationSeconds)
}

// RecordAIRequest records an AI request event
func (i *Instrumentation) RecordAIRequest(provider string, durationSeconds float64) {
	labels := map[string]string{
		"provider": provider,
	}
	i.metrics.Counter(MetricAIRequestsTotal).WithLabels(labels).Inc()
	i.metrics.Histogram(MetricAIRequestDurationSeconds).WithLabels(labels).Observe(durationSeconds)
}

// RecordError records an error event
func (i *Instrumentation) RecordError(errorType string) {
	labels := map[string]string{
		"type": errorType,
	}
	i.metrics.Counter(MetricErrorsTotal).WithLabels(labels).Inc()
}


