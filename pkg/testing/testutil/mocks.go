package testutil

import (
	"context"
	"fmt"
	"io"
)

// Logger interface for logging
type Logger interface {
	Info(msg string)
	Error(msg string, err error)
	Debug(msg string)
	Warn(msg string, args ...interface{})
}

// NullLogger is a logger that discards all output
type NullLogger struct{}

func (n NullLogger) Info(msg string)                              {}
func (n NullLogger) Error(msg string, err error)                  {}
func (n NullLogger) Debug(msg string)                             {}
func (n NullLogger) Warn(msg string, args ...interface{})        {}

// MockWriter is a mock io.Writer for testing
type MockWriter struct {
	Written []byte
	Err     error
}

func (m *MockWriter) Write(p []byte) (n int, err error) {
	if m.Err != nil {
		return 0, m.Err
	}
	m.Written = append(m.Written, p...)
	return len(p), nil
}

// MockReader is a mock io.Reader for testing
type MockReader struct {
	Data []byte
	Err  error
	pos  int
}

func (m *MockReader) Read(p []byte) (n int, err error) {
	if m.Err != nil {
		return 0, m.Err
	}
	if m.pos >= len(m.Data) {
		return 0, io.EOF
	}
	n = copy(p, m.Data[m.pos:])
	m.pos += n
	return n, nil
}

// MockCloser is a mock io.Closer for testing
type MockCloser struct {
	Err error
}

func (m *MockCloser) Close() error {
	return m.Err
}

// MockContext is a mock context for testing
type MockContext struct {
	context.Context
	DoneCh chan struct{}
	Err    error
}

func NewMockContext() *MockContext {
	return &MockContext{
		Context: context.Background(),
		DoneCh:  make(chan struct{}),
	}
}

func (m *MockContext) Done() <-chan struct{} {
	return m.DoneCh
}

func (m *MockContext) Err() error {
	return m.Err
}

// Cancel cancels the mock context
func (m *MockContext) Cancel() {
	close(m.DoneCh)
	m.Err = context.Canceled
}

// MockError is a mock error for testing
type MockError struct {
	Message string
	Code    string
}

func (m *MockError) Error() string {
	return m.Message
}

// NewMockError creates a new mock error
func NewMockError(message string) *MockError {
	return &MockError{Message: message}
}

// TestFixture provides common test fixtures
type TestFixture struct {
	TempDir string
	Logger  Logger
	Writer  io.Writer
	Reader  io.Reader
}

// NewTestFixture creates a new test fixture
func NewTestFixture(t *testing.T) *TestFixture {
	return &TestFixture{
		TempDir: TempDir(t),
		Logger:  NullLogger{},
		Writer:  &MockWriter{},
		Reader:  &MockReader{},
	}
}

// CreateProjectStructure creates a basic project structure for testing
func (f *TestFixture) CreateProjectStructure(t *testing.T, projectType string) {
	t.Helper()

	switch projectType {
	case "nodejs":
		WriteFile(t, f.TempDir, "package.json", `{
  "name": "test-project",
  "version": "1.0.0",
  "scripts": {
    "start": "node index.js"
  }
}`)
		WriteFile(t, f.TempDir, "index.js", "console.log('Hello World');")
	case "python":
		WriteFile(t, f.TempDir, "requirements.txt", "requests==2.31.0")
		WriteFile(t, f.TempDir, "main.py", "print('Hello World')")
	case "go":
		WriteFile(t, f.TempDir, "go.mod", "module test-project\n\ngo 1.21")
		WriteFile(t, f.TempDir, "main.go", "package main\n\nfunc main() {\n\tprintln(\"Hello World\")\n}")
	default:
		t.Fatalf("unknown project type: %s", projectType)
	}
}

// AssertContains checks if a string contains a substring
func AssertContains(t *testing.T, s, substr string) {
	t.Helper()
	if !strings.Contains(s, substr) {
		t.Errorf("expected %q to contain %q", s, substr)
	}
}

// AssertNotContains checks if a string does not contain a substring
func AssertNotContains(t *testing.T, s, substr string) {
	t.Helper()
	if strings.Contains(s, substr) {
		t.Errorf("expected %q not to contain %q", s, substr)
	}
}

// MustParseURL parses a URL and fails the test if it fails
func MustParseURL(t *testing.T, urlStr string) interface{} {
	t.Helper()
	// This is a placeholder - actual implementation depends on URL type
	// Will be implemented when we know the exact URL type
	return nil
}

// MustCreatePath creates a path and fails the test if it fails
func MustCreatePath(t *testing.T, pathStr string) interface{} {
	t.Helper()
	// This is a placeholder - actual implementation depends on Path type
	// Will be implemented when we know the exact Path type
	return nil
}


