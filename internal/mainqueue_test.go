package internal

import "testing"

func TestOpsQueueFIFO(t *testing.T) {
	q := newOpsQueue()
	var got []int
	for i := range 3 {
		i := i
		if !q.push(func() { got = append(got, i) }) {
			t.Fatalf("push %d dropped on an open queue", i)
		}
	}
	q.drain()
	if len(got) != 3 || got[0] != 0 || got[1] != 1 || got[2] != 2 {
		t.Fatalf("ops ran out of order: %v", got)
	}
}

func TestOpsQueueDrainAll(t *testing.T) {
	q := newOpsQueue()
	ran := 0
	for range 70 { // more than the buffer would matter only if push blocked
		if !q.push(func() { ran++ }) {
			t.Fatal("push dropped on an open queue")
		}
		q.drain()
	}
	for range 5 {
		q.push(func() { ran++ })
	}
	q.drain()
	if ran != 75 {
		t.Fatalf("drain missed ops: ran %d of 75", ran)
	}
}

func TestOpsQueueClosedDropsOps(t *testing.T) {
	q := newOpsQueue()
	ran := 0
	q.push(func() { ran++ })
	q.close()
	if q.push(func() { ran++ }) {
		t.Fatal("push on a closed queue was accepted")
	}
	q.drain()
	if ran != 0 {
		t.Fatal("queued op ran after close")
	}
}
