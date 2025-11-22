package monitoring

// ErrorCategory classifies types of errors
type ErrorCategory string

const (
	CategoryDependency      ErrorCategory = "missing_dependency"
	CategoryConfiguration   ErrorCategory = "configuration_error"
	CategoryBuildFailure    ErrorCategory = "build_failure"
	CategoryPermission      ErrorCategory = "permission_error"
	CategoryNetwork         ErrorCategory = "network_error"
	CategoryEnvironment     ErrorCategory = "environment_error"
	CategoryVersionConflict ErrorCategory = "version_conflict"
	CategorySyntax          ErrorCategory = "syntax_error"
	CategoryRuntime         ErrorCategory = "runtime_error"
	CategoryUnknown         ErrorCategory = "unknown_error"
)

// Severity levels
type Severity int

const (
	SeverityDebug Severity = iota
	SeverityInfo
	SeverityWarning
	SeverityError
	SeverityCritical
)

// String returns the string representation of severity
func (s Severity) String() string {
	switch s {
	case SeverityDebug:
		return "debug"
	case SeverityInfo:
		return "info"
	case SeverityWarning:
		return "warning"
	case SeverityError:
		return "error"
	case SeverityCritical:
		return "critical"
	default:
		return "unknown"
	}
}


