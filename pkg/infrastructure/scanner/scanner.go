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

// getSuspiciousPatterns returns a comprehensive list of security patterns to check
func getSuspiciousPatterns() []SuspiciousPattern {
	return []SuspiciousPattern{
		// Credential patterns (CRITICAL)
		{
			Pattern:     regexp.MustCompile(`AKIA[0-9A-Z]{16}`),
			Description: "AWS Access Key ID detected",
			Severity:    "critical",
		},
		{
			Pattern:     regexp.MustCompile(`-----BEGIN (RSA |EC |DSA |OPENSSH )?PRIVATE KEY-----`),
			Description: "Private key detected",
			Severity:    "critical",
		},
		{
			Pattern:     regexp.MustCompile(`(?i)(api[_-]?key|apikey|secret[_-]?key|access[_-]?token)\s*[:=]\s*['"][^'"]{20,}['"]`),
			Description: "API key or secret detected",
			Severity:    "high",
		},
		{
			Pattern:     regexp.MustCompile(`(?i)password\s*[:=]\s*['"][^'"]{8,}['"]`),
			Description: "Hardcoded password detected",
			Severity:    "high",
		},

		// Dangerous code execution (HIGH)
		{
			Pattern:     regexp.MustCompile(`(?i)eval\s*\(`),
			Description: "Use of eval() - potential code injection",
			Severity:    "high",
		},
		{
			Pattern:     regexp.MustCompile(`(?i)exec\s*\(`),
			Description: "Use of exec() - potential command injection",
			Severity:    "high",
		},
		{
			Pattern:     regexp.MustCompile(`(?i)(os\.system|subprocess\.call|subprocess\.Popen)`),
			Description: "Subprocess execution",
			Severity:    "high",
		},
		{
			Pattern:     regexp.MustCompile(`(?i)(child_process\.exec|child_process\.spawn)`),
			Description: "Child process execution in Node.js",
			Severity:    "high",
		},

		// Malware indicators (CRITICAL)
		{
			Pattern:     regexp.MustCompile(`(?i)(\/bin\/(ba)?sh\s+-i|nc\s+-e|python\s+-c.*socket)`),
			Description: "Reverse shell detected",
			Severity:    "critical",
		},
		{
			Pattern:     regexp.MustCompile(`(?i)(stratum\+tcp|xmr|monero|coinhive|cryptonight)`),
			Description: "Cryptocurrency miner detected",
			Severity:    "critical",
		},
		{
			Pattern:     regexp.MustCompile(`(?i)(curl|wget|nc)\s+.*\$`),
			Description: "Potential data exfiltration",
			Severity:    "high",
		},

		// Dangerous file operations (HIGH)
		{
			Pattern:     regexp.MustCompile(`(?i)rm\s+-rf?\s+/`),
			Description: "Dangerous file deletion command",
			Severity:    "high",
		},
		{
			Pattern:     regexp.MustCompile(`(?i)(format|fdisk|mkfs)\s+`),
			Description: "Disk formatting command",
			Severity:    "critical",
		},
		{
			Pattern:     regexp.MustCompile(`(?i)chmod\s+777`),
			Description: "Overly permissive file permissions",
			Severity:    "medium",
		},

		// Network attacks (HIGH)
		{
			Pattern:     regexp.MustCompile(`(?i)(curl|wget).*\|\s*(bash|sh|python|node)`),
			Description: "Piping remote content to interpreter",
			Severity:    "high",
		},
		{
			Pattern:     regexp.MustCompile(`(?i)(curl|wget).*\|.*(eval|exec)`),
			Description: "Piping remote content to eval/exec",
			Severity:    "critical",
		},

		// Obfuscation (HIGH)
		{
			Pattern:     regexp.MustCompile(`(?i)base64[._-]?decode.*(exec|eval|system)`),
			Description: "Base64 encoded execution",
			Severity:    "high",
		},
		{
			Pattern:     regexp.MustCompile(`(?i)\\x[0-9a-f]{2}.*\\x[0-9a-f]{2}`),
			Description: "Hex-encoded strings",
			Severity:    "medium",
		},
		{
			Pattern:     regexp.MustCompile(`(?i)fromCharCode.*eval`),
			Description: "Obfuscated eval",
			Severity:    "high",
		},

		// Build hook attacks (CRITICAL)
		{
			Pattern:     regexp.MustCompile(`"(pre|post)install"\s*:\s*"[^"]*\b(curl|wget|node\s+-e|sh\s+-c)\b`),
			Description: "NPM postinstall hook with network execution",
			Severity:    "critical",
		},
		{
			Pattern:     regexp.MustCompile(`(?i)cmdclass\s*=.*install.*(curl|wget|exec)`),
			Description: "Pip install hook with execution",
			Severity:    "high",
		},

		// Privilege escalation (MEDIUM)
		{
			Pattern:     regexp.MustCompile(`(?i)sudo\s+`),
			Description: "Requires elevated privileges",
			Severity:    "medium",
		},
		{
			Pattern:     regexp.MustCompile(`(?i)su\s+-`),
			Description: "User switching command",
			Severity:    "medium",
		},

		// Other suspicious patterns (LOW-MEDIUM)
		{
			Pattern:     regexp.MustCompile(`(?i)__import__\s*\(\s*['"]os['"]\s*\)`),
			Description: "Dynamic import of os module",
			Severity:    "medium",
		},
		{
			Pattern:     regexp.MustCompile(`(?i)crypto\.createCipher`),
			Description: "Deprecated crypto method",
			Severity:    "low",
		},
		{
			Pattern:     regexp.MustCompile(`(?i)\.env\s*=`),
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
	criticalSeverityCount := 0
	highSeverityCount := 0
	mediumSeverityCount := 0

	for _, finding := range result.Findings {
		switch finding.Severity {
		case "critical":
			criticalSeverityCount++
			result.SuspiciousReasons = append(result.SuspiciousReasons,
				fmt.Sprintf("[CRITICAL] %s at %s:%d", finding.Description, finding.File, finding.Line))
		case "high":
			highSeverityCount++
			result.SuspiciousReasons = append(result.SuspiciousReasons,
				fmt.Sprintf("[HIGH] %s at %s:%d", finding.Description, finding.File, finding.Line))
		case "medium":
			mediumSeverityCount++
		}
	}

	// Mark as suspicious if:
	// - Any critical severity findings
	// - Any high severity findings
	// - More than 3 medium severity findings
	if criticalSeverityCount > 0 || highSeverityCount > 0 || mediumSeverityCount > 3 {
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

