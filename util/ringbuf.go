package util

// RingBuf is a generic circular buffer backed
// by a slice that automatically grows.
//
// The zero value represents an empty RingBuf.
type RingBuf[T any] struct {
	rp   int
	wp   int
	size int
	b    []T
}

// Increases the capacity by n.
func (rb *RingBuf[T]) GrowCap(n int) {
	oldCap := len(rb.b)
	rb.b = append(rb.b, make([]T, n)...)
	if rb.size == 0 {
		return
	}
	if rb.rp >= rb.wp {
		shift := len(rb.b) - oldCap
		copy(rb.b[rb.rp+shift:], rb.b[rb.rp:oldCap])
		rb.rp += shift
	}
}

// Returns the maximum number of elements
// the RingBuf can hold without growing
// the backing slice.
func (rb *RingBuf[T]) Cap() int {
	return len(rb.b)
}

// Returns the number of readable elements.
func (rb *RingBuf[T]) Len() int {
	return rb.size
}

// Writes the given items into the RingBuf.
func (rb *RingBuf[T]) Write(v ...T) {
	if overshoot := rb.Len() + len(v) - len(rb.b); overshoot > 0 {
		// Double the capacity at least
		rb.GrowCap(max(overshoot, len(rb.b)))
	}
	if rb.wp < rb.rp {
		copy(rb.b[rb.wp:rb.rp], v)
		rb.wp += len(v)
	} else {
		n := copy(rb.b[rb.wp:], v)
		rb.wp += n
		if rb.wp >= len(rb.b) { // wrap around
			rb.wp = copy(rb.b, v[n:])
		}
	}
	rb.size += len(v)
}

// Reads a single item.
func (rb *RingBuf[T]) Read() T {
	if rb.size == 0 {
		panic("out of bounds read (read ptr write ptr collision)")
	}
	v := rb.b[rb.rp]
	rb.rp++
	if rb.rp == len(rb.b) {
		rb.rp = 0
	}
	rb.size--
	return v
}

// Reads all readable data as up to 2 read-only buffers
// without any copying. The entire data is the concatenation
// of the two returned buffers.
//
// Warning: Don't just append bufA and bufB as it may
// alter the RingBuf's internal slice data.
//
// The returned slices are only valid until before the next
// read or write operation.
//
// This function does not allocate or copy any memory and
// should be pretty fast.
func (rb *RingBuf[T]) ReadAll() (bufA []T, bufB []T) {
	bufA, bufB = rb.PeekAll()
	rb.rp = rb.wp
	rb.size = 0
	return
}

// Like [RingBuf.ReadAll], but doesn't change the read
// position.
func (rb *RingBuf[T]) PeekAll() (bufA []T, bufB []T) {
	if rb.size == 0 {
		return
	}
	if rb.rp < rb.wp {
		bufA = rb.b[rb.rp:rb.wp]
	} else {
		bufA = rb.b[rb.rp:]
		bufB = rb.b[:rb.wp]
	}
	return
}
