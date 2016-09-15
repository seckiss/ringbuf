package ringbuf

import (
	"fmt"
	"log"
	"sync"
)

// Overwriting ring buffer
type Buffer struct {
	data []byte
	size int
	//write cursor
	cursor  int
	written int64
	readers []*RingReader
}

var mutex = &sync.RWMutex{}
var cond = sync.NewCond(mutex.RLocker())

func NewBuffer(size int) *Buffer {
	if size <= 0 {
		log.Fatal("Size must be positive")
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
	if n > b.size {
		buf = buf[n-b.size:]
	}

	var toend = b.size - b.cursor
	copy(b.data[b.cursor:], buf)
	if len(buf) > toend {
		copy(b.data, buf[toend:])
	}

	b.cursor = ((b.cursor + len(buf)) % b.size)

	var writeCursorAfter = b.cursor

	// invalidate stale RingReaders
	var invalid []int
	for i, reader := range b.readers {
		// if overflow or reader cursor between
		if n > b.size || b.isBetween(reader.cursor, writeCursorBefore, writeCursorAfter) {
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
func (b *Buffer) isBetween(cursor int, rangeFrom int, rangeTo int) bool {
	if rangeFrom >= b.size || rangeTo >= b.size {
		log.Fatal("rangeFrom or rangeTo out of buffer.size range")
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

func (b *Buffer) Size() int {
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
	if b.written >= int64(b.size) {
		var out = make([]byte, b.size)
		copy(out, b.data[b.cursor:])
		copy(out[b.size-b.cursor:], b.data[:b.cursor])
		return out
	} else {
		if b.written != int64(b.cursor) {
			log.Fatal("Serious inconsistency")
		}
		var out = make([]byte, b.written)
		copy(out, b.data[:b.written])
		return out
	}
}

func (b *Buffer) String() string {
	return string(b.Bytes())
}

func (b *Buffer) NewReader() *RingReader {
	return b.NewReader(0)
}

func (b *Buffer) NewReader(cursorOffset int) *RingReader {
	mutex.Lock()
	if cursorOffset < 0 || cursorOffset > b.size {
		log.Fatal("cursorOffset must be in [0, b.size] range")
	}
	defer mutex.Unlock()
	// ensure that the cursorStart is positive
	var cursorStart = (2*b.cursor - cursorOffset) % b.size
	var reader = &RingReader{b, cursorStart, true}
	b.readers = append(b.readers, reader)
	return reader
}

type RingReader struct {
	ring   *Buffer
	cursor int
	valid  bool
}

func (reader *RingReader) Read(p []byte) (n int, err error) {
	mutex.RLock()
	defer mutex.RUnlock()
	var ring = reader.ring
	for {
		if !reader.valid {
			return 0, fmt.Errorf("Reader invalidated because writer overwrote the reader cursor")
		} else {
			if ring.cursor == reader.cursor {
				// nothing to read so wait
				cond.Wait()
			} else if ring.cursor > reader.cursor {
				var avail = int(ring.cursor - reader.cursor)
				var n = len(p)
				if avail < n {
					n = avail
				}
				copy(p, ring.data[reader.cursor:reader.cursor+n])
				reader.cursor = ((reader.cursor + n) % ring.size)
				return n, nil
			} else {
				var toend = ring.size - reader.cursor
				var avail = int(toend + ring.cursor)
				var n = len(p)
				if avail < n {
					n = avail
				}
				if n > ring.size-reader.cursor {
					copy(p, ring.data[reader.cursor:])
					copy(p[toend:], ring.data[:ring.cursor])
				} else {
					copy(p, ring.data[reader.cursor:reader.cursor+n])
				}
				reader.cursor = ((reader.cursor + n) % ring.size)
				return n, nil
			}
		}
	}
}
