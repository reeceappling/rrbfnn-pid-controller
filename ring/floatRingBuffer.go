package ring

import "iter"

// A Buffer is an element of a circular list, or ring.
// Rings do not have a beginning or end; a pointer to any ring element
// serves as reference to the entire ring. Empty rings are represented
// as nil Buffer pointers. The zero value for a Buffer is a one-element
// ring with a nil Value.
type Buffer struct {
	next, prev *Buffer
	Value      float64 // for use by client; untouched by this library
}

func (r *Buffer) init() *Buffer {
	r.next = r
	r.prev = r
	return r
}

// Next returns the next ring element. r must not be empty.
func (r *Buffer) Next() *Buffer {
	if r.next == nil {
		return r.init()
	}
	return r.next
}

// Prev returns the previous ring element. r must not be empty.
func (r *Buffer) Prev() *Buffer {
	if r.next == nil {
		return r.init()
	}
	return r.prev
}

// Move moves n % r.Len() elements backward (n < 0) or forward (n >= 0)
// in the ring and returns that ring element. r must not be empty.
func (r *Buffer) Move(n int) *Buffer {
	if r.next == nil {
		return r.init()
	}
	switch {
	case n < 0:
		for ; n < 0; n++ {
			r = r.prev
		}
	case n > 0:
		for ; n > 0; n-- {
			r = r.next
		}
	}
	return r
}

// New creates a ring of n elements.
func New(n int) *Buffer {
	if n <= 0 {
		return nil
	}
	r := new(Buffer)
	p := r
	for i := 1; i < n; i++ {
		p.next = &Buffer{prev: p}
		p = p.next
	}
	p.next = r
	r.prev = p
	return r
}
func NewBuffer(n int) *Buffer {
	return New(n)
}

// Link connects ring r with ring s such that r.Next()
// becomes s and returns the original value for r.Next().
// r must not be empty.
//
// If r and s point to the same ring, linking
// them removes the elements between r and s from the ring.
// The removed elements form a subring and the result is a
// reference to that subring (if no elements were removed,
// the result is still the original value for r.Next(),
// and not nil).
//
// If r and s point to different rings, linking
// them creates a single ring with the elements of s inserted
// after r. The result points to the element following the
// last element of s after insertion.
func (r *Buffer) Link(s *Buffer) *Buffer {
	n := r.Next()
	if s != nil {
		p := s.Prev()
		// Note: Cannot use multiple assignment because
		// evaluation order of LHS is not specified.
		r.next = s
		s.prev = r
		n.prev = p
		p.next = n
	}
	return n
}

// Unlink removes n % r.Len() elements from the ring r, starting
// at r.Next(). If n % r.Len() == 0, r remains unchanged.
// The result is the removed subring. r must not be empty.
func (r *Buffer) Unlink(n int) *Buffer {
	if n <= 0 {
		return nil
	}
	return r.Link(r.Move(n + 1))
}

// Len computes the number of elements in ring r.
// It executes in time proportional to the number of elements.
func (r *Buffer) Len() int {
	n := 0
	if r != nil {
		n = 1
		for p := r.Next(); p != r; p = p.next {
			n++
		}
	}
	return n
}

// Do calls function f on each element of the ring, in forward order.
// The behavior of Do is undefined if f changes *r.
func (r *Buffer) Do(f func(any)) {
	if r != nil {
		f(r.Value)
		for p := r.Next(); p != r; p = p.next {
			f(p.Value)
		}
	}
}

//type StructBuffer[T any] struct {
//	next, prev *StructBuffer[T]
//	Value      float64 // for use by client; untouched by this library
//}
//
//func (r *StructBuffer[T]) init() *StructBuffer[T] {
//	r.next = r
//	r.prev = r
//	return r
//}
//
//// Next returns the next ring element. r must not be empty.
//func (r *StructBuffer[T]) Next() *StructBuffer[T] {
//	if r.next == nil {
//		return r.init()
//	}
//	return r.next
//}
//
//// Prev returns the previous ring element. r must not be empty.
//func (r *StructBuffer[T]) Prev() *StructBuffer[T] {
//	if r.next == nil {
//		return r.init()
//	}
//	return r.prev
//}
//
//// Move moves n % r.Len() elements backward (n < 0) or forward (n >= 0)
//// in the ring and returns that ring element. r must not be empty.
//func (r *StructBuffer[T]) Move(n int) *StructBuffer[T] {
//	if r.next == nil {
//		return r.init()
//	}
//	switch {
//	case n < 0:
//		for ; n < 0; n++ {
//			r = r.prev
//		}
//	case n > 0:
//		for ; n > 0; n-- {
//			r = r.next
//		}
//	}
//	return r
//}
//
//// New creates a ring of n elements.
//func NewStructBuffer[T any](n int) *StructBuffer[T] {
//	if n <= 0 {
//		return nil
//	}
//	r := new(StructBuffer[T])
//	p := r
//	for i := 1; i < n; i++ {
//		p.next = &StructBuffer[T]{
//			next:  nil,
//			prev:  p,
//			Value: 0,
//		}
//		p = p.next
//	}
//	p.next = r
//	r.prev = p
//	return r
//}
//
//// Link connects ring r with ring s such that r.Next()
//// becomes s and returns the original value for r.Next().
//// r must not be empty.
////
//// If r and s point to the same ring, linking
//// them removes the elements between r and s from the ring.
//// The removed elements form a subring and the result is a
//// reference to that subring (if no elements were removed,
//// the result is still the original value for r.Next(),
//// and not nil).
////
//// If r and s point to different rings, linking
//// them creates a single ring with the elements of s inserted
//// after r. The result points to the element following the
//// last element of s after insertion.
//func (r *StructBuffer[T]) Link(s *StructBuffer[T]) *StructBuffer[T] {
//	n := r.Next()
//	if s != nil {
//		p := s.Prev()
//		// Note: Cannot use multiple assignment because
//		// evaluation order of LHS is not specified.
//		r.next = s
//		s.prev = r
//		n.prev = p
//		p.next = n
//	}
//	return n
//}
//
//// Unlink removes n % r.Len() elements from the ring r, starting
//// at r.Next(). If n % r.Len() == 0, r remains unchanged.
//// The result is the removed subring. r must not be empty.
//func (r *StructBuffer[T]) Unlink(n int) *StructBuffer[T] {
//	if n <= 0 {
//		return nil
//	}
//	return r.Link(r.Move(n + 1))
//}
//
//// Len computes the number of elements in ring r.
//// It executes in time proportional to the number of elements.
//func (r *StructBuffer[T]) Len() int {
//	n := 0
//	if r != nil {
//		n = 1
//		for p := r.Next(); p != r; p = p.next {
//			n++
//		}
//	}
//	return n
//}
//
//// Do calls function f on each element of the ring, in forward order.
//// The behavior of Do is undefined if f changes *r.
//func (r *StructBuffer[T]) Do(f func(any)) {
//	if r != nil {
//		f(r.Value)
//		for p := r.Next(); p != r; p = p.next {
//			f(p.Value)
//		}
//	}
//}
//

type StructRing[T any] struct {
	currentIndex int
	values       []T
}

func NewStructRing[T any](size int) *StructRing[T] {
	return &StructRing[T]{
		values:       make([]T, size),
		currentIndex: 0,
	}
}
func NewStructRingWithDefault[T any](size int, def T) *StructRing[T] {
	ring := &StructRing[T]{
		values:       make([]T, size),
		currentIndex: 0,
	}
	ring.setDefault(def)
	return ring
}
func (r *StructRing[T]) nextIndex() int {
	if r.currentIndex >= len(r.values) {
		return 0
	}
	return r.currentIndex + 1
}
func (r *StructRing[T]) secondEarliestIndex() int {
	return (r.currentIndex + 2) % len(r.values)
}
func (r *StructRing[T]) setDefault(val T) {
	for i := 0; i < len(r.values); i++ {
		r.values[i] = val
	}
}
func (r *StructRing[T]) Earliest() T {
	return r.values[r.nextIndex()]
}
func (r *StructRing[T]) SecondEarliest() T {
	return r.values[r.secondEarliestIndex()]
}
func (r *StructRing[T]) Latest() T {
	return r.values[r.currentIndex]
}
func (r *StructRing[T]) Prev() T {
	return r.values[(r.currentIndex+len(r.values)-1)%len(r.values)]
}
func (r *StructRing[T]) GetXAgo(nAgo int) T {
	return r.values[(r.currentIndex+len(r.values)+nAgo)%len(r.values)]
}
func (r *StructRing[T]) ReplaceCurrent(val T) {
	r.values[r.currentIndex] = val
}
func (r *StructRing[T]) Iter() iter.Seq[T] {
	return func(yield func(T) bool) {
		for i := 0; i < len(r.values); i++ {
			if !yield(r.values[(i+1+r.currentIndex)%len(r.values)]) {
				return
			}
		}
	}
}
func (r *StructRing[T]) IterBackwards() iter.Seq[T] {
	return func(yield func(T) bool) {
		for i := 0; i < len(r.values); i++ {
			if !yield(r.values[(r.currentIndex+len(r.values)-i)%len(r.values)]) {
				return
			}
		}
	}
}
func (r *StructRing[T]) PrevPrev() T {
	return r.values[(r.currentIndex+len(r.values)-2)%len(r.values)]
}
func (r *StructRing[T]) Add(val T) (removed T) {
	nextIndex := r.nextIndex()
	removedVal := r.values[nextIndex]
	r.values[nextIndex], r.currentIndex = val, nextIndex
	return removedVal
}
