//go:build noclipboard

package views

// initClipboard is a no-op stub when clipboard is disabled
func initClipboard() error {
	return nil
}

// readClipboard returns empty when clipboard is disabled
func readClipboard() []byte {
	return nil
}

// clipboardAvailable returns false when clipboard is disabled
func clipboardAvailable() bool {
	return false
}
