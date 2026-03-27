package memory

import "time"

type readyItem struct {
	jobID    string
	priority int
	sequence int64
}

type readyHeap []readyItem

func (h readyHeap) Len() int {
	return len(h)
}

func (h readyHeap) Less(i, j int) bool {
	if h[i].priority == h[j].priority {
		return h[i].sequence < h[j].sequence
	}
	return h[i].priority > h[j].priority
}

func (h readyHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

func (h *readyHeap) Push(x any) {
	*h = append(*h, x.(readyItem))
}

func (h *readyHeap) Pop() any {
	current := *h
	last := len(current) - 1
	item := current[last]
	*h = current[:last]
	return item
}

type delayedItem struct {
	jobID    string
	runAt    time.Time
	priority int
	sequence int64
}

type delayedHeap []delayedItem

func (h delayedHeap) Len() int {
	return len(h)
}

func (h delayedHeap) Less(i, j int) bool {
	if h[i].runAt.Equal(h[j].runAt) {
		if h[i].priority == h[j].priority {
			return h[i].sequence < h[j].sequence
		}
		return h[i].priority > h[j].priority
	}
	return h[i].runAt.Before(h[j].runAt)
}

func (h delayedHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

func (h *delayedHeap) Push(x any) {
	*h = append(*h, x.(delayedItem))
}

func (h *delayedHeap) Pop() any {
	current := *h
	last := len(current) - 1
	item := current[last]
	*h = current[:last]
	return item
}
