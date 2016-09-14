package ringbuf

import "sync"

// Overwriting ring buffer
type Buffer struct {
	data    []byte
	size    int64
	cursor  int64
	written int64
}

var mutex = &sync.Mutex{}

func NewBuffer(size int64) *Buffer {
	if size <= 0 {
		panic("Size must be positive")
	}
	return &Buffer{
		size: size,
		data: make([]byte, size),
	}
}

// Write len(buf) bytes to the internal ring,
// overriding older data if necessary.
func (b *Buffer) Write(buf []byte) (int, error) {
	mutex.Lock()
	defer mutex.Unlock()

	// Account for total bytes written
	n := len(buf)
	b.written += int64(n)

	// If the buffer is larger than ring, then truncate to last b.size bytes
	if int64(n) > b.size {
		buf = buf[int64(n)-b.size:]
	}

	var toend = b.size - b.cursor
	copy(b.data[b.cursor:], buf)
	if int64(len(buf)) > toend {
		copy(b.data, buf[toend:])
	}

	b.cursor = ((b.cursor + int64(len(buf))) % b.size)
	return n, nil
}

func (b *Buffer) Size() int64 {
	mutex.Lock()
	defer mutex.Unlock()
	return b.size
}

func (b *Buffer) TotalWritten() int64 {
	mutex.Lock()
	defer mutex.Unlock()
	return b.written
}

// Return copy slice of the unwinded ring
func (b *Buffer) Bytes() []byte {
	mutex.Lock()
	defer mutex.Unlock()
	if b.written >= b.size {
		var out = make([]byte, b.size)
		copy(out, b.data[b.cursor:])
		copy(out[b.size-b.cursor:], b.data[:b.cursor])
		return out
	} else {
		if b.written != b.cursor {
			panic("Serious inconsistency")
		}
		var out = make([]byte, b.written)
		copy(out, b.data[:b.written])
		return out
	}
}

func (b *Buffer) Reset() {
	mutex.Lock()
	defer mutex.Unlock()
	b.cursor = 0
	b.written = 0
}

func (b *Buffer) String() string {
	mutex.Lock()
	defer mutex.Unlock()
	return string(b.Bytes())
}
