package internal

import "sync/atomic"

// opsQueue buffers callbacks that a worker goroutine wants applied on
// the platform's main thread, for platforms where the UI API is not
// thread-safe (macOS AppKit). Push blocks when the buffer is full, like
// a channel send; the platform dispatches drain onto the main thread.
// Once closed (the tray is being destroyed), pushes are dropped and
// drain discards queued ops instead of running them, so nothing touches
// released objects.
type opsQueue struct {
	ops    chan func()
	closed atomic.Bool
}

const opsQueueSize = 64

func newOpsQueue() *opsQueue {
	return &opsQueue{ops: make(chan func(), opsQueueSize)}
}

// push queues op for the next drain. It reports whether op was queued;
// on a closed queue op is dropped and never runs.
func (q *opsQueue) push(op func()) bool {
	if q.closed.Load() {
		return false
	}
	q.ops <- op
	return true
}

// drain runs every queued op in FIFO order until the queue is empty.
// On a closed queue it discards the ops without running them.
// Must be called on the platform's main thread.
func (q *opsQueue) drain() {
	for {
		select {
		case op := <-q.ops:
			if q.closed.Load() {
				continue
			}
			op()
		default:
			return
		}
	}
}

// close marks the queue closed: queued and future ops are dropped.
func (q *opsQueue) close() {
	q.closed.Store(true)
}
