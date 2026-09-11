package logging

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// RotatingLogger manages log file rotation based on size, age, and backup limits
type RotatingLogger struct {
	basePath   string
	maxSize    int64 // bytes
	maxAge     int   // days
	maxBackups int
	current    *rotatingFile
	mu         sync.Mutex
}

// rotatingFile represents a rotating file writer
type rotatingFile struct {
	file *os.File
	path string
	size int64
	mu   sync.Mutex
}

// RotatingConfig holds configuration for log rotation
type RotatingConfig struct {
	BasePath   string
	MaxSize    int64 // bytes
	MaxAge     int   // days
	MaxBackups int
}

// NewRotatingLogger creates a new rotating logger
func NewRotatingLogger(config RotatingConfig) (*RotatingLogger, error) {
	if config.BasePath == "" {
		return nil, fmt.Errorf("base path cannot be empty")
	}

	// Ensure directory exists
	dir := filepath.Dir(config.BasePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	rl := &RotatingLogger{
		basePath:   config.BasePath,
		maxSize:    config.MaxSize,
		maxAge:     config.MaxAge,
		maxBackups: config.MaxBackups,
	}

	// Open or create the current log file
	if err := rl.openCurrentFile(); err != nil {
		return nil, err
	}

	return rl, nil
}

// Write implements io.Writer interface
func (rl *RotatingLogger) Write(p []byte) (n int, err error) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Check if rotation is needed
	if err := rl.rotateIfNeeded(); err != nil {
		return 0, err
	}

	// Write to current file
	rl.current.mu.Lock()
	defer rl.current.mu.Unlock()

	n, err = rl.current.file.Write(p)
	rl.current.size += int64(n)
	return n, err
}

// Close closes the rotating logger
func (rl *RotatingLogger) Close() error {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if rl.current != nil && rl.current.file != nil {
		return rl.current.file.Close()
	}
	return nil
}

// openCurrentFile opens or creates the current log file
func (rl *RotatingLogger) openCurrentFile() error {
	file, err := os.OpenFile(rl.basePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	info, err := file.Stat()
	if err != nil {
		file.Close()
		return fmt.Errorf("failed to stat log file: %w", err)
	}

	rl.current = &rotatingFile{
		file: file,
		path: rl.basePath,
		size: info.Size(),
	}

	return nil
}

// rotateIfNeeded checks if rotation is needed and performs it
func (rl *RotatingLogger) rotateIfNeeded() error {
	if rl.current == nil {
		return rl.openCurrentFile()
	}

	rl.current.mu.Lock()
	defer rl.current.mu.Unlock()

	needsRotation := rl.maxSize > 0 && rl.current.size >= rl.maxSize

	// Check size

	// Check age
	if rl.maxAge > 0 {
		info, err := rl.current.file.Stat()
		if err == nil {
			age := time.Since(info.ModTime())
			if age >= time.Duration(rl.maxAge)*24*time.Hour {
				needsRotation = true
			}
		}
	}

	if !needsRotation {
		return nil
	}

	return rl.rotate()
}

// rotate performs the actual rotation
func (rl *RotatingLogger) rotate() error {
	// Close current file
	if rl.current != nil && rl.current.file != nil {
		rl.current.file.Close()
	}

	// Rotate existing backups
	rl.rotateBackups()

	// Rename current file to backup
	timestamp := time.Now().Format("20060102-150405")
	backupPath := rl.basePath + "." + timestamp
	if err := os.Rename(rl.basePath, backupPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to rotate log file: %w", err)
	}

	// Clean up old backups
	rl.cleanupOldBackups()

	// Open new current file
	return rl.openCurrentFile()
}

// rotateBackups rotates existing backup files
func (rl *RotatingLogger) rotateBackups() {
	if rl.maxBackups <= 0 {
		return
	}

	// Find all backup files
	backups := rl.findBackupFiles()
	if len(backups) >= rl.maxBackups {
		// Sort by modification time (oldest first)
		sort.Slice(backups, func(i, j int) bool {
			return backups[i].ModTime().Before(backups[j].ModTime())
		})

		// Remove oldest backups
		for i := 0; i < len(backups)-rl.maxBackups+1; i++ {
			os.Remove(backups[i].Name())
		}
	}
}

// cleanupOldBackups removes backup files older than maxAge
func (rl *RotatingLogger) cleanupOldBackups() {
	if rl.maxAge <= 0 {
		return
	}

	cutoffTime := time.Now().Add(-time.Duration(rl.maxAge) * 24 * time.Hour)
	backups := rl.findBackupFiles()

	for _, backup := range backups {
		if backup.ModTime().Before(cutoffTime) {
			os.Remove(backup.Name())
		}
	}
}

// findBackupFiles finds all backup files matching the base path pattern
func (rl *RotatingLogger) findBackupFiles() []os.FileInfo {
	dir := filepath.Dir(rl.basePath)
	baseName := filepath.Base(rl.basePath)

	var backups []os.FileInfo
	entries, err := os.ReadDir(dir)
	if err != nil {
		return backups
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if strings.HasPrefix(name, baseName+".") && name != baseName {
			if info, err := entry.Info(); err == nil {
				backups = append(backups, info)
			}
		}
	}

	return backups
}

// RotatingWriter wraps a RotatingLogger to implement io.Writer
type RotatingWriter struct {
	*RotatingLogger
}

// NewRotatingWriter creates a new rotating writer
func NewRotatingWriter(config RotatingConfig) (io.Writer, error) {
	rl, err := NewRotatingLogger(config)
	if err != nil {
		return nil, err
	}
	return &RotatingWriter{RotatingLogger: rl}, nil
}
