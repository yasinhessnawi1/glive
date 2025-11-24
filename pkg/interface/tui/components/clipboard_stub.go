//go:build noclipboard

package components

// initClipboard is a no-op stub when clipboard is disabled
func initClipboard() error {
	return nil
}

// readClipboard returns empty when clipboard is disabled
func readClipboard() []byte {
	return nil
}
