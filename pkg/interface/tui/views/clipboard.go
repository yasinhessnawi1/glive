package views

// initClipboard is a no-op - clipboard functionality not available
// The golang.design/x/clipboard package requires CGO and X11 on Linux,
// which prevents cross-compilation. Clipboard support was removed.
func initClipboard() error {
	return nil
}

// readClipboard returns empty - clipboard functionality not available
func readClipboard() []byte {
	return nil
}

// clipboardAvailable returns false - clipboard functionality not available
func clipboardAvailable() bool {
	return false
}
