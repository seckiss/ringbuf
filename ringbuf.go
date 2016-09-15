package ringbuf

import (
	"fmt"
	"sync"
)

// Overwriting ring buffer
type Buffer struct {
	data []byte
	size int64
	//write cursor
	cursor  int64
	written int64
	readers []*RingReader
}

var mutex = &sync.RWMutex{}
var cond = sync.NewCond(mutex)

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

	var writeCursorBefore = b.cursor

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

	var writeCursorAfter = b.cursor

	// invalidate stale RingReaders
	var invalid []int
	for i, reader := range b.readers {
		if b.isBetween(reader.cursor, writeCursorBefore, writeCursorAfter) {
			reader.valid = false
			invalid = append(invalid, i)
		}
	}
	// delete invalidated RingReaders from collection
	if len(invalid) > 0 {
		var newreaders []*RingReader
		for i, reader := range b.readers {
			if !intcontains(invalid, i) {
				newreaders = append(newreaders, reader)
			}
		}
		b.readers = newreaders
	}

	cond.Broadcast()
	return n, nil
}

// circular between (excluding left endpoint, including right endpoint)
func (b *Buffer) isBetween(cursor int64, rangeFrom int64, rangeTo int64) bool {
	if rangeFrom >= b.size || rangeTo >= b.size {
		panic("rangeFrom or rangeTo out of buffer.size range")
	}
	if rangeTo > rangeFrom {
		return cursor > rangeFrom && cursor <= rangeTo
	}
	if rangeTo < rangeFrom {
		return cursor > rangeFrom || cursor <= rangeTo
	}
	return false
}

func intcontains(s []int, e int) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func (b *Buffer) Size() int64 {
	mutex.RLock()
	defer mutex.RUnlock()
	return b.size
}

func (b *Buffer) TotalWritten() int64 {
	mutex.RLock()
	defer mutex.RUnlock()
	return b.written
}

// Return copy slice of the unwinded ring
func (b *Buffer) Bytes() []byte {
	mutex.RLock()
	defer mutex.RUnlock()
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
	return string(b.Bytes())
}

func (b *Buffer) NewReader() *RingReader {
	var reader = &RingReader{b, 0, true}
	b.readers = append(b.readers, reader)
	return reader
}

type RingReader struct {
	ring   *Buffer
	cursor int64
	valid  bool
}

func (reader *RingReader) Read(p []byte) (n int, err error) {
	mutex.RLock()
	if !reader.valid {
		return 0, fmt.Errorf("Reader invalidated because writer overwrote the reader cursor")
	}
	defer mutex.RUnlock()
	cond.Wait()
	if !reader.valid {
		return 0, fmt.Errorf("Reader invalidated because writer overwrote the reader cursor")
	} else {
		var ring = reader.ring
		if ring.cursor == reader.cursor {
			// nothing to read but state ok
			return 0, nil
		} else if ring.cursor > reader.cursor {
			var avail = int(ring.cursor - reader.cursor)
			var n = len(p)
			if avail < n {
				n = avail
			}
			copy(p, ring.data[reader.ring.cursor:reader.cursor])
			return n, nil
		} else {
			var toend = ring.size - reader.cursor
			var avail = int(toend + ring.cursor)
			var n = len(p)
			if avail < n {
				n = avail
			}
			copy(p, ring.data[reader.cursor:])
			copy(p[toend:], ring.data[:ring.cursor])
			return n, nil
		}
	}
}
