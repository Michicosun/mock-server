package util

import (
	"sync"

	"github.com/gammazero/deque"
	"github.com/moznion/go-optional"
)

type BlockingQueue[T any] struct {
	max_size int
	closed   bool
	elems    *deque.Deque[T]
	mtx      *sync.Mutex
	full     *sync.Cond
	empty    *sync.Cond
}

// call under lock only
func (q *BlockingQueue[T]) isFull() bool {
	return q.max_size > 0 && q.elems.Len() >= q.max_size
}

func (q *BlockingQueue[T]) Put(el T) bool {
	q.mtx.Lock()
	defer q.mtx.Unlock()

	if !q.closed && q.isFull() {
		q.full.Wait()
	}

	if q.closed {
		return false
	}

	q.elems.PushBack(el)
	q.empty.Signal()
	return true
}

func (q *BlockingQueue[T]) Get() optional.Option[T] {
	q.mtx.Lock()
	defer q.mtx.Unlock()

	if !q.closed && q.elems.Len() == 0 {
		q.empty.Wait()
	}

	if q.elems.Len() != 0 {
		el := optional.Some(q.elems.PopFront())
		q.full.Signal()
		return el
	}

	return optional.None[T]()
}

func (q *BlockingQueue[T]) Close(clear bool) {
	q.mtx.Lock()
	defer q.mtx.Unlock()
	q.closed = true
	if clear {
		q.elems.Clear()
	}
	q.empty.Broadcast()
}

func (q *BlockingQueue[T]) IsClosed() bool {
	q.mtx.Lock()
	defer q.mtx.Unlock()
	return q.closed
}

func NewBoundedBlockingQueue[T any](max_size int) BlockingQueue[T] {
	q := BlockingQueue[T]{
		max_size: max_size,
		closed:   false,
		elems:    deque.New[T](),
		mtx:      &sync.Mutex{},
	}

	q.full = sync.NewCond(q.mtx)
	q.empty = sync.NewCond(q.mtx)

	return q
}

func NewUnboundedBlockingQueue[T any]() BlockingQueue[T] {
	return NewBoundedBlockingQueue[T](-1)
}
