package ai

import (
	"fmt"
	"regexp"
	"strings"
)

// RiskAssessor evaluates risk of recovery operations
type RiskAssessor struct {
	dangerousPatterns []*regexp.Regexp
}

// NewRiskAssessor creates a new risk assessor
func NewRiskAssessor() *RiskAssessor {
	return &RiskAssessor{
		dangerousPatterns: buildDangerousPatterns(),
	}
}

// buildDangerousPatterns creates patterns for dangerous commands
func buildDangerousPatterns() []*regexp.Regexp {
	patterns := []string{
		`(?i)rm\s+-rf\s+/`,                    // Dangerous rm commands
		`(?i)rm\s+-rf\s+[^/]*(?:etc|usr|Windows|System32)`, // System directories
		`(?i)chmod\s+777`,                     // Insecure permissions
		`(?i)curl.*\|\s*bash`,                 // Pipe to bash
		`(?i)wget.*\|\s*(?:sh|bash)`,          // Pipe to shell
		`(?i)sudo\s+`,                          // Sudo usage
		`(?i)--force`,                         // Force flags
		`(?i)npm.*-g`,                         // Global installs
		`(?i)pip.*--system`,                   // System-wide pip
		`(?i)pip.*--user`,                     // User-wide pip (less dangerous but still risky)
		`(?i)registry\s+add`,                  // Windows registry
		`(?i)systemctl`,                       // System service control
		`(?i)service\s+\w+\s+(?:start|stop|restart)`, // Service control
		`(?i)format\s+[a-zA-Z]:`,              // Disk format
		`(?i)del\s+/[fs]`,                     // Windows dangerous delete
		`(?i)rd\s+/[sq]`,                      // Windows dangerous remove directory
	}

	compiled := make([]*regexp.Regexp, len(patterns))
	for i, p := range patterns {
		compiled[i] = regexp.MustCompile(p)
	}

	return compiled
}

// RiskAssessment contains risk evaluation results
type RiskAssessment struct {
	Level   RiskLevel
	Factors []string
}

// AssessRisk evaluates the risk of a recovery step
func (ra *RiskAssessor) AssessRisk(step *RecoveryStep) RiskAssessment {
	assessment := RiskAssessment{
		Level:   RiskLow,
		Factors: []string{},
	}

	// Check for dangerous patterns
	for _, pattern := range ra.dangerousPatterns {
		if pattern.MatchString(step.Command) {
			assessment.Level = RiskHigh
			assessment.Factors = append(assessment.Factors,
				fmt.Sprintf("Matches dangerous pattern: %s", pattern.String()))
		}
	}

	// Check AI-provided risk level
	if step.RiskLevel == RiskHigh || step.RiskLevel == RiskCritical {
		if assessment.Level == "" || ra.compareRiskLevel(assessment.Level, step.RiskLevel) < 0 {
			assessment.Level = step.RiskLevel
		}
		assessment.Factors = append(assessment.Factors, "AI flagged as high risk")
	} else if step.RiskLevel != "" && assessment.Level == "" {
		assessment.Level = step.RiskLevel
	}

	// Check if command modifies system
	if ra.modifiesSystem(step.Command) {
		if assessment.Level == "" || ra.compareRiskLevel(assessment.Level, RiskMedium) < 0 {
			assessment.Level = RiskMedium
		}
		assessment.Factors = append(assessment.Factors, "Modifies system configuration")
	}

	// Check if reversible
	if !step.CanRollback {
		if assessment.Level == "" || ra.compareRiskLevel(assessment.Level, RiskMedium) < 0 {
			assessment.Level = RiskMedium
		}
		assessment.Factors = append(assessment.Factors, "Cannot be rolled back")
	}

	// Default to low if no factors found
	if assessment.Level == "" {
		assessment.Level = RiskLow
	}

	return assessment
}

// modifiesSystem checks if a command modifies system configuration
func (ra *RiskAssessor) modifiesSystem(command string) bool {
	systemModifiers := []string{
		"systemctl",
		"service",
		"update-alternatives",
		"apt-get install",
		"yum install",
		"brew install",
		"choco install",
		"winget install",
		"registry",
		"reg add",
		"reg delete",
		"setx",
		"export",
		"PATH=",
	}

	lowerCmd := strings.ToLower(command)
	for _, modifier := range systemModifiers {
		if strings.Contains(lowerCmd, modifier) {
			return true
		}
	}

	return false
}

// compareRiskLevel compares two risk levels
// Returns: -1 if a < b, 0 if a == b, 1 if a > b
func (ra *RiskAssessor) compareRiskLevel(a, b RiskLevel) int {
	levels := map[RiskLevel]int{
		RiskLow:      1,
		RiskMedium:   2,
		RiskHigh:     3,
		RiskCritical: 4,
	}

	levelA := levels[a]
	levelB := levels[b]

	if levelA < levelB {
		return -1
	} else if levelA > levelB {
		return 1
	}
	return 0
}

