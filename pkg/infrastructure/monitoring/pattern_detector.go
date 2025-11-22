package monitoring

import (
	"regexp"
)

// PatternDetector detects error patterns in command output
type PatternDetector struct {
	patterns []ErrorPattern
}

// ErrorPattern defines a detectable error pattern
type ErrorPattern struct {
	Name       string
	Regex      *regexp.Regexp
	Category   ErrorCategory
	Severity   Severity
	Confidence float64
	Extractor  func(line string, matches []string) map[string]string
}

// NewPatternDetector creates a detector with common patterns
func NewPatternDetector() *PatternDetector {
	return &PatternDetector{
		patterns: buildCommonPatterns(),
	}
}

// buildCommonPatterns creates patterns for common errors
func buildCommonPatterns() []ErrorPattern {
	return []ErrorPattern{
		// Node.js / npm patterns
		{
			Name:       "npm_module_not_found",
			Regex:      regexp.MustCompile(`(?i)Cannot find module ['"]([^'"]+)['"]`),
			Category:   CategoryDependency,
			Severity:   SeverityError,
			Confidence: 0.95,
			Extractor: func(line string, matches []string) map[string]string {
				if len(matches) > 1 {
					return map[string]string{"module": matches[1]}
				}
				return nil
			},
		},
		{
			Name:       "npm_package_not_found",
			Regex:      regexp.MustCompile(`(?i)npm ERR! 404.*'([^']+)'`),
			Category:   CategoryDependency,
			Severity:   SeverityError,
			Confidence: 0.90,
			Extractor: func(line string, matches []string) map[string]string {
				if len(matches) > 1 {
					return map[string]string{"package": matches[1]}
				}
				return nil
			},
		},
		{
			Name:       "npm_permission_denied",
			Regex:      regexp.MustCompile(`(?i)npm ERR!.*EACCES.*permission denied`),
			Category:   CategoryPermission,
			Severity:   SeverityCritical,
			Confidence: 0.95,
		},
		{
			Name:       "npm_network_error",
			Regex:      regexp.MustCompile(`(?i)npm ERR!.*network`),
			Category:   CategoryNetwork,
			Severity:   SeverityWarning,
			Confidence: 0.85,
		},

		// Python / pip patterns
		{
			Name:       "python_module_not_found",
			Regex:      regexp.MustCompile(`(?i)ModuleNotFoundError: No module named ['"]([^'"]+)['"]`),
			Category:   CategoryDependency,
			Severity:   SeverityError,
			Confidence: 0.95,
			Extractor: func(line string, matches []string) map[string]string {
				if len(matches) > 1 {
					return map[string]string{"module": matches[1]}
				}
				return nil
			},
		},
		{
			Name:       "pip_package_not_found",
			Regex:      regexp.MustCompile(`(?i)ERROR: Could not find.*matching ([^\s]+)`),
			Category:   CategoryDependency,
			Severity:   SeverityError,
			Confidence: 0.90,
			Extractor: func(line string, matches []string) map[string]string {
				if len(matches) > 1 {
					return map[string]string{"package": matches[1]}
				}
				return nil
			},
		},
		{
			Name:       "python_syntax_error",
			Regex:      regexp.MustCompile(`(?i)SyntaxError:`),
			Category:   CategorySyntax,
			Severity:   SeverityError,
			Confidence: 0.95,
		},
		{
			Name:       "pip_permission_denied",
			Regex:      regexp.MustCompile(`(?i)PermissionError|permission denied`),
			Category:   CategoryPermission,
			Severity:   SeverityCritical,
			Confidence: 0.90,
		},

		// Go patterns
		{
			Name:       "go_package_not_found",
			Regex:      regexp.MustCompile(`(?i)cannot find package "([^"]+)"`),
			Category:   CategoryDependency,
			Severity:   SeverityError,
			Confidence: 0.95,
			Extractor: func(line string, matches []string) map[string]string {
				if len(matches) > 1 {
					return map[string]string{"package": matches[1]}
				}
				return nil
			},
		},
		{
			Name:       "go_build_failed",
			Regex:      regexp.MustCompile(`(?i)build failed`),
			Category:   CategoryBuildFailure,
			Severity:   SeverityError,
			Confidence: 0.85,
		},
		{
			Name:       "go_import_error",
			Regex:      regexp.MustCompile(`(?i)undefined:.*import`),
			Category:   CategoryDependency,
			Severity:   SeverityError,
			Confidence: 0.90,
		},

		// Generic patterns
		{
			Name:       "command_not_found",
			Regex:      regexp.MustCompile(`(?i)(command not found|is not recognized|'[^']+' is not recognized)`),
			Category:   CategoryEnvironment,
			Severity:   SeverityError,
			Confidence: 0.90,
		},
		{
			Name:       "permission_denied",
			Regex:      regexp.MustCompile(`(?i)permission denied`),
			Category:   CategoryPermission,
			Severity:   SeverityCritical,
			Confidence: 0.95,
		},
		{
			Name:       "network_timeout",
			Regex:      regexp.MustCompile(`(?i)(timeout|timed out|connection refused|connection reset)`),
			Category:   CategoryNetwork,
			Severity:   SeverityWarning,
			Confidence: 0.80,
		},
		{
			Name:       "port_in_use",
			Regex:      regexp.MustCompile(`(?i)address already in use.*:(\d+)|port.*(\d+).*already in use`),
			Category:   CategoryConfiguration,
			Severity:   SeverityError,
			Confidence: 0.90,
			Extractor: func(line string, matches []string) map[string]string {
				if len(matches) > 1 {
					port := matches[1]
					if port == "" && len(matches) > 2 {
						port = matches[2]
					}
					if port != "" {
						return map[string]string{"port": port}
					}
				}
				return nil
			},
		},
		{
			Name:       "out_of_memory",
			Regex:      regexp.MustCompile(`(?i)(out of memory|OOM|killed|memory allocation failed)`),
			Category:   CategoryRuntime,
			Severity:   SeverityCritical,
			Confidence: 0.95,
		},
		{
			Name:       "file_not_found",
			Regex:      regexp.MustCompile(`(?i)(file not found|cannot find.*file|no such file)`),
			Category:   CategoryConfiguration,
			Severity:   SeverityError,
			Confidence: 0.85,
		},
		{
			Name:       "version_conflict",
			Regex:      regexp.MustCompile(`(?i)(version conflict|incompatible version|requires.*version)`),
			Category:   CategoryVersionConflict,
			Severity:   SeverityWarning,
			Confidence: 0.80,
		},
		// APT (Debian/Ubuntu) patterns
		{
			Name:       "apt_lock_error",
			Regex:      regexp.MustCompile(`(?i)(Could not get lock|Resource temporarily unavailable|Unable to acquire the dpkg frontend lock)`),
			Category:   CategoryEnvironment,
			Severity:   SeverityError,
			Confidence: 0.95,
		},
		{
			Name:       "apt_package_not_found",
			Regex:      regexp.MustCompile(`(?i)Unable to locate package ([^\s]+)`),
			Category:   CategoryDependency,
			Severity:   SeverityError,
			Confidence: 0.95,
			Extractor: func(line string, matches []string) map[string]string {
				if len(matches) > 1 {
					return map[string]string{"package": matches[1]}
				}
				return nil
			},
		},
		{
			Name:       "apt_permission_denied",
			Regex:      regexp.MustCompile(`(?i)(are you root|permission denied)`),
			Category:   CategoryPermission,
			Severity:   SeverityCritical,
			Confidence: 0.95,
		},

		// Chocolatey patterns
		{
			Name:       "choco_access_denied",
			Regex:      regexp.MustCompile(`(?i)Access to the path.*is denied`),
			Category:   CategoryPermission,
			Severity:   SeverityCritical,
			Confidence: 0.90,
		},
		{
			Name:       "choco_package_not_found",
			Regex:      regexp.MustCompile(`(?i)The package was not found with the source\(s\) listed`),
			Category:   CategoryDependency,
			Severity:   SeverityError,
			Confidence: 0.90,
		},
		{
			Name:       "choco_checksum_mismatch",
			Regex:      regexp.MustCompile(`(?i)Checksums do not match`),
			Category:   CategoryBuildFailure,
			Severity:   SeverityError,
			Confidence: 0.95,
		},

		// Yarn patterns
		{
			Name:       "yarn_network_error",
			Regex:      regexp.MustCompile(`(?i)There appears to be trouble with your network connection`),
			Category:   CategoryNetwork,
			Severity:   SeverityWarning,
			Confidence: 0.90,
		},
		{
			Name:       "yarn_integrity_error",
			Regex:      regexp.MustCompile(`(?i)integrity check failed`),
			Category:   CategoryBuildFailure,
			Severity:   SeverityError,
			Confidence: 0.95,
		},
	}
}

// Detect finds patterns in a line
func (pd *PatternDetector) Detect(line string) []PatternMatch {
	var matches []PatternMatch

	for _, pattern := range pd.patterns {
		if submatches := pattern.Regex.FindStringSubmatch(line); submatches != nil {
			match := PatternMatch{
				Pattern:    pattern.Name,
				Confidence: pattern.Confidence,
				Category:   pattern.Category,
				Severity:   pattern.Severity,
			}

			if pattern.Extractor != nil {
				match.Context = pattern.Extractor(line, submatches)
			}

			matches = append(matches, match)
		}
	}

	return matches
}

// AddCustomPattern adds a custom pattern
func (pd *PatternDetector) AddCustomPattern(pattern ErrorPattern) {
	pd.patterns = append(pd.patterns, pattern)
}
