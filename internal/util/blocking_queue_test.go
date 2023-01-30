package util

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestSimple(t *testing.T) {
	q := NewBoundedBlockingQueue[int](10)

	q.Put(1)
	q.Put(2)
	q.Put(3)

	for x := 1; x <= 3; x += 1 {
		el := q.Get().Unwrap()
		if el != x {
			t.Errorf("x != el <=> %d != %d", x, el)
		}
	}

	c := q.IsClosed()
	if c {
		t.Errorf("queue is closed")
	}

	q.Close(false)
	el := q.Get()
	if !el.IsNone() {
		t.Errorf("queue close not working")
	}
}

func TestConcurrent(t *testing.T) {
	elements_count := 10000000
	concurrency_level := 10
	q := NewUnboundedBlockingQueue[int]() // unbounded

	c := int64(0)
	b := make(chan struct{})
	var wg sync.WaitGroup

	start_func := func() {
		<-b
		for {
			el := q.Get()
			if el.IsNone() {
				wg.Done()
				return
			}
			atomic.AddInt64(&c, 1)
		}
	}

	for i := 0; i < concurrency_level; i += 1 {
		wg.Add(1)
		go start_func()
	}

	close(b)

	for i := 0; i < elements_count; i += 1 {
		q.Put(i)
	}

	q.Close(false)
	wg.Wait()

	if c != int64(elements_count) {
		t.Errorf("concurrency broken")
	}
}

func TestClose(t *testing.T) {
	concurrency_level := 10
	q := NewUnboundedBlockingQueue[int]() // unbounded

	b := make(chan struct{})
	var wg sync.WaitGroup

	start_func := func() {
		<-b
		el := q.Get()
		if !el.IsNone() {
			t.Errorf("close not working")
		}
		wg.Done()
	}

	for i := 0; i < concurrency_level; i += 1 {
		wg.Add(1)
		go start_func()
	}

	close(b)

	time.Sleep(time.Millisecond * 500)

	q.Close(true)
	wg.Wait()
}
