package ringbuf

import "testing"

func TestRingbuf_rw1(t *testing.T) {
	ring := NewBuffer(10)
	// must get the new reader BEFORE writing! Otherwise no data available for read
	r := ring.NewReader()
	b := []byte{0, 1, 2, 3, 4, 5, 6, 7}
	n, err := ring.Write(b)
	if err != nil || n != len(b) {
		t.Fatalf("Unexpected output from ring.Write(b). n=%+v, err=%+v", n, err)
	}
	b2 := make([]byte, 0, 20)
	n, err = r.Read(b2[:cap(b2)])
	if err != nil || n != len(b) {
		t.Fatalf("Unexpected output from r.Read(b2). n=%+v, err=%+v", n, err)
	}
}
