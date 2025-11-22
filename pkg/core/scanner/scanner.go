package scanner

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Scanner scans code for suspicious patterns
type Scanner struct {
	projectPath string
	patterns    []SuspiciousPattern
}

// SuspiciousPattern represents a pattern to detect
type SuspiciousPattern struct {
	Pattern     *regexp.Regexp
	Description string
	Severity    string // high, medium, low
}

// ScanResult contains scan results
type ScanResult struct {
	IsSuspicious      bool
	SuspiciousReasons []string
	Findings          []Finding
}

// Finding represents a suspicious finding
type Finding struct {
	File        string
	Line        int
	Pattern     string
	Description string
	Severity    string
	Code        string
}

// New creates a new Scanner
func New(projectPath string) *Scanner {
	return &Scanner{
		projectPath: projectPath,
		patterns:    getSuspiciousPatterns(),
	}
}

// getSuspiciousPatterns returns a list of patterns to check
func getSuspiciousPatterns() []SuspiciousPattern {
	return []SuspiciousPattern{
		{
			Pattern:     regexp.MustCompile(`eval\s*\(`),
			Description: "Use of eval() - potential code injection",
			Severity:    "high",
		},
		{
			Pattern:     regexp.MustCompile(`exec\s*\(`),
			Description: "Use of exec() - potential command injection",
			Severity:    "high",
		},
		{
			Pattern:     regexp.MustCompile(`rm\s+-rf\s+/`),
			Description: "Dangerous file deletion command",
			Severity:    "high",
		},
		{
			Pattern:     regexp.MustCompile(`sudo\s+`),
			Description: "Requires elevated privileges",
			Severity:    "medium",
		},
		{
			Pattern:     regexp.MustCompile(`chmod\s+777`),
			Description: "Overly permissive file permissions",
			Severity:    "medium",
		},
		{
			Pattern:     regexp.MustCompile(`curl.*\|\s*bash`),
			Description: "Piping remote content to bash",
			Severity:    "high",
		},
		{
			Pattern:     regexp.MustCompile(`wget.*\|\s*sh`),
			Description: "Piping remote content to shell",
			Severity:    "high",
		},
		{
			Pattern:     regexp.MustCompile(`__import__\s*\(\s*['"]os['"]\s*\)`),
			Description: "Dynamic import of os module",
			Severity:    "medium",
		},
		{
			Pattern:     regexp.MustCompile(`subprocess\.call\s*\(`),
			Description: "Subprocess execution",
			Severity:    "low",
		},
		{
			Pattern:     regexp.MustCompile(`child_process`),
			Description: "Child process execution in Node.js",
			Severity:    "low",
		},
		{
			Pattern:     regexp.MustCompile(`crypto\.createCipher`),
			Description: "Deprecated crypto method",
			Severity:    "low",
		},
		{
			Pattern:     regexp.MustCompile(`\.env\s*=`),
			Description: "Environment variable manipulation",
			Severity:    "low",
		},
	}
}

// Scan performs a security scan of the project
func (s *Scanner) Scan() (*ScanResult, error) {
	result := &ScanResult{
		IsSuspicious:      false,
		SuspiciousReasons: []string{},
		Findings:          []Finding{},
	}

	// Walk through all files
	err := filepath.Walk(s.projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-code files
		if info.IsDir() || s.shouldSkipFile(path) {
			return nil
		}

		// Scan file
		findings, err := s.scanFile(path)
		if err != nil {
			// Log error but continue
			return nil
		}

		result.Findings = append(result.Findings, findings...)

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Analyze findings
	s.analyzeFindings(result)

	return result, nil
}

// scanFile scans a single file for suspicious patterns
func (s *Scanner) scanFile(filePath string) ([]Finding, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	findings := []Finding{}
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Check each pattern
		for _, pattern := range s.patterns {
			if pattern.Pattern.MatchString(line) {
				findings = append(findings, Finding{
					File:        filePath,
					Line:        lineNum,
					Pattern:     pattern.Pattern.String(),
					Description: pattern.Description,
					Severity:    pattern.Severity,
					Code:        strings.TrimSpace(line),
				})
			}
		}
	}

	return findings, scanner.Err()
}

// analyzeFindings analyzes scan findings to determine if project is suspicious
func (s *Scanner) analyzeFindings(result *ScanResult) {
	highSeverityCount := 0
	mediumSeverityCount := 0

	for _, finding := range result.Findings {
		switch finding.Severity {
		case "high":
			highSeverityCount++
			result.SuspiciousReasons = append(result.SuspiciousReasons,
				fmt.Sprintf("%s at %s:%d", finding.Description, finding.File, finding.Line))
		case "medium":
			mediumSeverityCount++
		}
	}

	// Mark as suspicious if:
	// - Any high severity findings
	// - More than 3 medium severity findings
	if highSeverityCount > 0 || mediumSeverityCount > 3 {
		result.IsSuspicious = true
	}
}

// shouldSkipFile determines if a file should be skipped
func (s *Scanner) shouldSkipFile(path string) bool {
	// Skip common non-code files and directories
	skipPatterns := []string{
		".git/",
		"node_modules/",
		"vendor/",
		"__pycache__/",
		".pytest_cache/",
		"dist/",
		"build/",
		".next/",
		".nuxt/",
		"coverage/",
		".idea/",
		".vscode/",
		"*.min.js",
		"*.map",
		"*.lock",
		"*.log",
		"*.jpg",
		"*.png",
		"*.gif",
		"*.pdf",
		"*.zip",
		"*.tar",
		"*.gz",
	}

	for _, pattern := range skipPatterns {
		if strings.Contains(path, pattern) {
			return true
		}
	}

	return false
}
