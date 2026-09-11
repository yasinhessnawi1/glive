package memory

import (
	"bufio"
	"bytes"
	"context"
	"os"
	"strings"
	"sync"
)

// BufferPool provides reusable byte buffers
var BufferPool = sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(make([]byte, 0, 4096))
	},
}

// GetBuffer retrieves a buffer from the pool
func GetBuffer() *bytes.Buffer {
	return BufferPool.Get().(*bytes.Buffer)
}

// PutBuffer returns a buffer to the pool
func PutBuffer(buf *bytes.Buffer) {
	buf.Reset()
	BufferPool.Put(buf)
}

// StringBuilderPool for string concatenation
var StringBuilderPool = sync.Pool{
	New: func() interface{} {
		return &strings.Builder{}
	},
}

// GetStringBuilder retrieves a string builder from the pool
func GetStringBuilder() *strings.Builder {
	return StringBuilderPool.Get().(*strings.Builder)
}

// PutStringBuilder returns a string builder to the pool
func PutStringBuilder(sb *strings.Builder) {
	sb.Reset()
	StringBuilderPool.Put(sb)
}

// StreamingReader reads large files without loading into memory
type StreamingReader struct {
	path      string
	chunkSize int
}

// NewStreamingReader creates a new streaming reader
func NewStreamingReader(path string, chunkSize int) *StreamingReader {
	return &StreamingReader{
		path:      path,
		chunkSize: chunkSize,
	}
}

// ProcessLines processes a file line by line without loading it entirely into memory
func (r *StreamingReader) ProcessLines(ctx context.Context, processor func(line string) error) error {
	file, err := os.Open(r.path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, r.chunkSize), r.chunkSize)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := processor(scanner.Text()); err != nil {
				return err
			}
		}
	}

	return scanner.Err()
}
