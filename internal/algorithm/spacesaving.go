package algorithm

import (
	"container/heap"
)

// Item represents an item in the Space-Saving algorithm.
type Item struct {
	Key   string
	Count uint64
	Error uint64
	Index int // Index in the heap
}

// SpaceSavingHeap is a min-heap of Items.
type SpaceSavingHeap []*Item

// SpaceSaving implements the Space-Saving algorithm for finding top-k frequent items.
type SpaceSaving struct {
	capacity int
	items    map[string]*Item
	heap     SpaceSavingHeap
}

// NewSpaceSaving creates a new Space-Saving instance with the given capacity.
func NewSpaceSaving(capacity int) *SpaceSaving {
	return &SpaceSaving{
		capacity: capacity,
		items:    make(map[string]*Item),
		heap:     make(SpaceSavingHeap, 0, capacity),
	}
}

// Add adds an item to the Space-Saving structure.
func (ss *SpaceSaving) Add(key string, count uint64) {
	// If the key already exists, increment its count
	if item, ok := ss.items[key]; ok {
		item.Count += count
		heap.Fix(&ss.heap, item.Index)
		return
	}

	// If we haven't reached the capacity, add the item
	if len(ss.heap) < ss.capacity {
		item := &Item{
			Key:   key,
			Count: count,
			Error: 0,
		}
		ss.items[key] = item
		heap.Push(&ss.heap, item)
		return
	}

	// Otherwise, replace the smallest item
	smallest := ss.heap[0]
	newCount := smallest.Count + count
	newError := smallest.Count

	// Remove the old item
	delete(ss.items, smallest.Key)

	// Update the item with the new key
	smallest.Key = key
	smallest.Count = newCount
	smallest.Error = newError

	// Add the new item to the map and fix the heap
	ss.items[key] = smallest
	heap.Fix(&ss.heap, 0)
}

// TopK returns the top k items.
func (ss *SpaceSaving) TopK(k int) []Item {
	// Create a copy of the heap
	copyHeap := make(SpaceSavingHeap, len(ss.heap))
	copy(copyHeap, ss.heap)

	// Sort the copy
	result := make([]Item, 0, len(copyHeap))
	for len(copyHeap) > 0 {
		item := heap.Pop(&copyHeap).(*Item)
		result = append(result, *item)
	}

	// Return the top k items (or all if k > len(result))
	if k > len(result) {
		k = len(result)
	}

	// Reverse the order (we want highest count first)
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}

	return result[:k]
}

// Len returns the length of the heap.
func (h SpaceSavingHeap) Len() int { return len(h) }

// Less returns whether the item at index i is less than the item at index j.
func (h SpaceSavingHeap) Less(i, j int) bool { return h[i].Count < h[j].Count }

// Swap swaps the items at indices i and j.
func (h SpaceSavingHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].Index = i
	h[j].Index = j
}

// Push adds an item to the heap.
func (h *SpaceSavingHeap) Push(x any) {
	n := len(*h)
	item := x.(*Item)
	item.Index = n
	*h = append(*h, item)
}

// Pop removes and returns the minimum item from the heap.
func (h *SpaceSavingHeap) Pop() any {
	old := *h
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // avoid memory leak
	item.Index = -1 // for safety
	*h = old[0 : n-1]
	return item
}

// Count returns the count for a specific key
func (ss *SpaceSaving) Count(key string) uint64 {
	if item, ok := ss.items[key]; ok {
		return item.Count
	}
	return 0
}

// Decay applies exponential decay to all counts
func (ss *SpaceSaving) Decay(factor float64) {
	for _, item := range ss.items {
		item.Count = uint64(float64(item.Count) * factor)
		item.Error = uint64(float64(item.Error) * factor)
	}
	// Re-heapify since counts have changed
	heap.Init(&ss.heap)
}

// Clear removes all items from the Space-Saving structure
func (ss *SpaceSaving) Clear() {
	ss.items = make(map[string]*Item)
	ss.heap = make(SpaceSavingHeap, 0, ss.capacity)
}
