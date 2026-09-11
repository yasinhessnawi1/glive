package presenter

import (
	stderrors "errors"
	"fmt"
	"strings"

	domainerrors "github.com/glive/domain/errors"
)

// ErrorPresenter formats errors for user display
type ErrorPresenter struct {
	verbose  bool
	colorize bool
}

// NewErrorPresenter creates a new error presenter
func NewErrorPresenter(verbose, colorize bool) *ErrorPresenter {
	return &ErrorPresenter{
		verbose:  verbose,
		colorize: colorize,
	}
}

// Present formats an error for display to the user
func (p *ErrorPresenter) Present(err error) string {
	if err == nil {
		return ""
	}

	var sb strings.Builder

	// Get the GliveError if available
	var gliveErr *domainerrors.GliveError
	isGlive := stderrors.As(err, &gliveErr)

	// Error header
	if isGlive {
		sb.WriteString(p.formatHeader(gliveErr))
	} else {
		sb.WriteString(p.color("red", "Error: "))
		sb.WriteString(err.Error())
		sb.WriteString("\n")
	}

	// User action
	if action := domainerrors.GetUserAction(err); action != "" {
		sb.WriteString("\n")
		sb.WriteString(p.color("yellow", "What to do: "))
		sb.WriteString(action)
		sb.WriteString("\n")
	}

	// Verbose details
	if p.verbose && isGlive {
		sb.WriteString(p.formatDetails(gliveErr))
	}

	return sb.String()
}

func (p *ErrorPresenter) formatHeader(err *domainerrors.GliveError) string {
	var icon string
	var color string

	switch err.Severity {
	case domainerrors.SeverityWarning:
		icon = "⚠️ "
		color = "yellow"
	case domainerrors.SeverityError:
		icon = "❌ "
		color = "red"
	case domainerrors.SeverityCritical:
		icon = "🚨 "
		color = "red"
	case domainerrors.SeverityFatal:
		icon = "💀 "
		color = "red"
	default:
		icon = "❌ "
		color = "red"
	}

	return fmt.Sprintf("%s%s\n", icon, p.color(color, err.Message))
}

func (p *ErrorPresenter) formatDetails(err *domainerrors.GliveError) string {
	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString(p.color("dim", "─── Details ───\n"))
	fmt.Fprintf(&sb, "Code: %s\n", err.Code)
	fmt.Fprintf(&sb, "Category: %s\n", categoryName(err.Category))

	if len(err.Context) > 0 {
		sb.WriteString("Context:\n")
		for k, v := range err.Context {
			fmt.Fprintf(&sb, "  %s: %s\n", k, v)
		}
	}

	if err.Cause != nil {
		fmt.Fprintf(&sb, "Cause: %v\n", err.Cause)
	}

	if err.Retryable {
		sb.WriteString(p.color("green", "This error may be temporary. You can try again.\n"))
	}

	return sb.String()
}

// FormatForLog formats error for logging (no color, full details)
func FormatForLog(err error) map[string]interface{} {
	result := map[string]interface{}{
		"error": err.Error(),
	}

	var gliveErr *domainerrors.GliveError
	if stderrors.As(err, &gliveErr) {
		result["code"] = gliveErr.Code
		result["category"] = categoryName(gliveErr.Category)
		result["severity"] = severityName(gliveErr.Severity)
		result["retryable"] = gliveErr.Retryable
		if len(gliveErr.Context) > 0 {
			result["context"] = gliveErr.Context
		}
		if gliveErr.Cause != nil {
			result["cause"] = gliveErr.Cause.Error()
		}
	}

	return result
}

func (p *ErrorPresenter) color(colorName, text string) string {
	if !p.colorize {
		return text
	}

	// ANSI color codes
	codes := map[string]string{
		"red":    "\033[31m",
		"green":  "\033[32m",
		"yellow": "\033[33m",
		"blue":   "\033[34m",
		"dim":    "\033[2m",
		"reset":  "\033[0m",
	}

	code, ok := codes[colorName]
	if !ok {
		return text
	}

	return code + text + codes["reset"]
}

func categoryName(cat domainerrors.Category) string {
	names := map[domainerrors.Category]string{
		domainerrors.CategoryUser:      "User",
		domainerrors.CategoryNetwork:   "Network",
		domainerrors.CategorySystem:    "System",
		domainerrors.CategorySecurity:  "Security",
		domainerrors.CategoryAI:        "AI",
		domainerrors.CategoryGit:       "Git",
		domainerrors.CategoryExecution: "Execution",
		domainerrors.CategoryConfig:    "Config",
		domainerrors.CategoryInternal:  "Internal",
	}
	if name, ok := names[cat]; ok {
		return name
	}
	return "Unknown"
}

func severityName(sev domainerrors.Severity) string {
	names := map[domainerrors.Severity]string{
		domainerrors.SeverityWarning:  "Warning",
		domainerrors.SeverityError:    "Error",
		domainerrors.SeverityCritical: "Critical",
		domainerrors.SeverityFatal:    "Fatal",
	}
	if name, ok := names[sev]; ok {
		return name
	}
	return "Unknown"
}
