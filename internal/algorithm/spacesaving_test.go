package algorithm

import (
	"fmt"
	"testing"
)

func TestSpaceSaving_NewAndBasicOperations(t *testing.T) {
	ss := NewSpaceSaving(3)

	// Add some items
	ss.Add("apple", 5)
	ss.Add("banana", 3)
	ss.Add("apple", 2) // apple total: 7

	// Get top items
	topItems := ss.TopK(3)

	// Should have 2 unique items
	if len(topItems) != 2 {
		t.Errorf("Expected 2 items, got %d", len(topItems))
	}

	// Find apple in results
	found := false
	for _, item := range topItems {
		if item.Key == "apple" && item.Count >= 7 {
			found = true
			break
		}
	}

	if !found {
		t.Error("apple with count >= 7 not found in top items")
	}
}

func TestSpaceSaving_CapacityLimit(t *testing.T) {
	capacity := 2
	ss := NewSpaceSaving(capacity)

	// Add more unique items than capacity
	ss.Add("first", 10)
	ss.Add("second", 5)
	ss.Add("third", 1) // This should replace the least frequent

	topItems := ss.TopK(10)

	// Should not exceed capacity
	if len(topItems) > capacity {
		t.Errorf("Items count %d exceeds capacity %d", len(topItems), capacity)
	}

	// First item should still be there (highest count)
	found := false
	for _, item := range topItems {
		if item.Key == "first" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Highest count item should not be evicted")
	}
}

func TestSpaceSaving_TopKOrdering(t *testing.T) {
	ss := NewSpaceSaving(5)

	// Add items with known frequencies
	items := map[string]uint64{
		"most":   100,
		"second": 50,
		"third":  25,
		"fourth": 10,
		"fifth":  5,
	}

	for key, count := range items {
		ss.Add(key, count)
	}

	// Test different k values
	tests := []struct {
		k        int
		expected int
	}{
		{1, 1},
		{3, 3},
		{5, 5},
		{10, 5}, // Should return all 5 items even if k > actual count
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("k=%d", tt.k), func(t *testing.T) {
			topItems := ss.TopK(tt.k)

			if len(topItems) != tt.expected {
				t.Errorf("TopK(%d) returned %d items, expected %d",
					tt.k, len(topItems), tt.expected)
			}

			// Verify items are sorted in descending order of count
			for i := 0; i < len(topItems)-1; i++ {
				if topItems[i].Count < topItems[i+1].Count {
					t.Errorf("Items not sorted: index %d count %d < index %d count %d",
						i, topItems[i].Count, i+1, topItems[i+1].Count)
				}
			}
		})
	}
}

func TestSpaceSaving_ErrorTracking(t *testing.T) {
	ss := NewSpaceSaving(2) // Small capacity to force replacements

	// Add enough items to cause replacements
	ss.Add("first", 10)
	ss.Add("second", 5)
	ss.Add("third", 1) // This should replace an existing item

	topItems := ss.TopK(10)

	// Check that error values are tracked correctly
	for _, item := range topItems {
		if item.Key == "third" {
			// Third item should have error since it replaced another item
			if item.Error == 0 {
				t.Error("Replaced item should have non-zero error")
			}
		}
	}
}
