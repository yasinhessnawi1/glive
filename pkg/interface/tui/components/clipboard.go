//go:build !noclipboard

package components

import (
	"golang.design/x/clipboard"
)

// initClipboard initializes the clipboard
func initClipboard() error {
	return clipboard.Init()
}

// readClipboard reads text from the clipboard
func readClipboard() []byte {
	return clipboard.Read(clipboard.FmtText)
}
