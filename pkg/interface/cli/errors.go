package cli

import (
	stderrors "errors"
	"fmt"
	"strings"

	domainerrors "github.com/glive/domain/errors"
)

// FormatError formats an error for display with context and suggestions
func FormatError(err error) string {
	if err == nil {
		return ""
	}

	var sb strings.Builder

	// Get the GliveError if available
	var gliveErr *domainerrors.GliveError
	isGlive := stderrors.As(err, &gliveErr)

	// Error header with icon and color
	if isGlive {
		sb.WriteString(formatErrorHeader(gliveErr))
	} else {
		sb.WriteString(Error("Error: " + err.Error()))
		sb.WriteString("\n")
	}

	// User action
	if action := domainerrors.GetUserAction(err); action != "" {
		sb.WriteString("\n")
		sb.WriteString(Info("What to do:"))
		sb.WriteString("\n  " + action + "\n")
	}

	// Suggestion based on error category
	if suggestion := getSuggestion(err); suggestion != "" {
		sb.WriteString("\n")
		sb.WriteString(Colorize("Suggestion: "+suggestion, Gray))
		sb.WriteString("\n")
	}

	return sb.String()
}

// formatErrorHeader formats the error header with appropriate icon and color
func formatErrorHeader(err *domainerrors.GliveError) string {
	var icon string
	var color Color

	switch err.Severity {
	case domainerrors.SeverityWarning:
		icon = "⚠"
		color = Yellow
	case domainerrors.SeverityError:
		icon = "✗"
		color = Red
	case domainerrors.SeverityCritical:
		icon = "🚨"
		color = Red
	case domainerrors.SeverityFatal:
		icon = "💀"
		color = Red
	default:
		icon = "✗"
		color = Red
	}

	return fmt.Sprintf("%s %s\n", icon, Colorize(err.Message, color))
}

// getSuggestion provides contextual suggestions based on error type
func getSuggestion(err error) string {
	var gliveErr *domainerrors.GliveError
	if !stderrors.As(err, &gliveErr) {
		return ""
	}

	switch gliveErr.Category {
	case domainerrors.CategoryNetwork:
		if gliveErr.Retryable {
			return "This appears to be a network issue. Check your internet connection and try again."
		}
		return "Network connectivity issue detected. Verify your connection and firewall settings."
	case domainerrors.CategoryConfig:
		return "Check your configuration file (~/.glive.json) or use 'glive config' commands."
	case domainerrors.CategoryGit:
		return "Verify the repository URL is correct and accessible. Check your Git credentials if needed."
	case domainerrors.CategoryExecution:
		return "The command may have failed due to missing dependencies or incorrect environment setup."
	case domainerrors.CategorySecurity:
		return "Review the security findings before proceeding. Use --force flag only if you trust the repository."
	case domainerrors.CategoryAI:
		if gliveErr.Retryable {
			return "The AI service may be temporarily unavailable. Try again in a few moments."
		}
		return "AI service error. Check your API key configuration."
	case domainerrors.CategorySystem:
		return "System resource issue detected. Check disk space, permissions, and system requirements."
	default:
		return ""
	}
}

// getMainMessage extracts the main error message
func getMainMessage(err error) string {
	var gliveErr *domainerrors.GliveError
	if stderrors.As(err, &gliveErr) {
		return gliveErr.Message
	}
	return err.Error()
}


