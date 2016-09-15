package ringbuf

import "testing"

func TestRingbuf_simple(t *testing.T) {
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

func TestRingbuf_smallbyteslice(t *testing.T) {
	ring := NewBuffer(10)
	// must get the new reader BEFORE writing! Otherwise no data available for read
	r := ring.NewReader()
	b := []byte{0, 1, 2, 3, 4, 5, 6, 7}
	n, err := ring.Write(b)
	if err != nil || n != len(b) {
		t.Fatalf("Unexpected output from ring.Write(b). n=%+v, err=%+v", n, err)
	}
	b2 := make([]byte, 0, 3)
	n, err = r.Read(b2[:cap(b2)])
	if err != nil || n != cap(b2) {
		t.Fatalf("Unexpected output from r.Read(b2). n=%+v, err=%+v", n, err)
	}
}

func TestRingbuf_tworeaders(t *testing.T) {
	ring := NewBuffer(10)
	// must get the new reader BEFORE writing! Otherwise no data available for read
	r1 := ring.NewReader()
	b1 := []byte{0, 1, 2, 3, 4}
	n, err := ring.Write(b1)
	if err != nil || n != len(b1) {
		t.Fatalf("Unexpected output from ring.Write(b). n=%+v, err=%+v", n, err)
	}
	r2 := ring.NewReader()
	b2 := []byte{5, 6, 7}
	n, err = ring.Write(b2)
	if err != nil || n != len(b2) {
		t.Fatalf("Unexpected output from ring.Write(b). n=%+v, err=%+v", n, err)
	}
	b3 := make([]byte, 0, 20)
	n, err = r1.Read(b3[:cap(b3)])
	if err != nil || n != 8 {
		t.Fatalf("Unexpected output from r.Read(b3). n=%+v, err=%+v", n, err)
	}
	b4 := make([]byte, 0, 20)
	n, err = r2.Read(b4[:cap(b4)])
	if err != nil || n != 3 {
		t.Fatalf("Unexpected output from r.Read(b4). n=%+v, err=%+v", n, err)
	}
}

func TestRingbuf_invalidate(t *testing.T) {
	ring := NewBuffer(10)
	// must get the new reader BEFORE writing! Otherwise no data available for read
	r := ring.NewReader()
	b := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}
	n, err := ring.Write(b)
	if err != nil || n != len(b) {
		t.Fatalf("Unexpected output from ring.Write(b). n=%+v, err=%+v", n, err)
	}
	b2 := make([]byte, 0, 20)
	n, err = r.Read(b2[:cap(b2)])
	// should return error 'overwrote'
	if err == nil || n != 0 {
		t.Fatalf("Unexpected output from r.Read(b2). n=%+v, err=%+v", n, err)
	}
}
