package interactive

import (
	"github.com/sahilm/fuzzy"
	"strings"
)

// FuzzySearchFunc creates a fuzzy search function for promptui
func FuzzySearchFunc(items []string) func(input string, index int) bool {
	return func(input string, index int) bool {
		// Empty search shows all items
		if input == "" {
			return true
		}

		// Convert to lowercase for case-insensitive search
		input = strings.ToLower(input)
		item := strings.ToLower(items[index])

		// First try simple substring match
		if strings.Contains(item, input) {
			return true
		}

		// Then try fuzzy match
		pattern := fuzzy.Find(input, []string{item})
		return len(pattern) > 0
	}
}

// FuzzySearcher provides advanced fuzzy search with result ordering
type FuzzySearcher struct {
	allItems    []string
	originalMap map[int]int // Maps display index to original index
}

// NewFuzzySearcher creates a new fuzzy searcher
func NewFuzzySearcher(items []string) *FuzzySearcher {
	return &FuzzySearcher{
		allItems:    items,
		originalMap: make(map[int]int),
	}
}

// CreateSearchFunc creates a search function that filters and reorders results
func (f *FuzzySearcher) CreateSearchFunc() func(input string, index int) bool {
	return func(input string, index int) bool {
		if input == "" {
			// Reset mapping for no search
			f.originalMap = make(map[int]int)
			for i := range f.allItems {
				f.originalMap[i] = i
			}
			return true
		}

		// Perform fuzzy search
		results := fuzzy.FindFrom(input, fuzzySource(f.allItems))

		// Build new mapping
		newMap := make(map[int]int)
		for i, result := range results {
			newMap[i] = result.Index
		}
		f.originalMap = newMap

		// Check if current index is in results
		for i := range results {
			if i == index {
				return true
			}
		}
		return false
	}
}

// GetOriginalIndex maps a filtered index back to the original index
func (f *FuzzySearcher) GetOriginalIndex(filteredIndex int) int {
	if original, ok := f.originalMap[filteredIndex]; ok {
		return original
	}
	// If no mapping exists, assume direct mapping
	return filteredIndex
}

// fuzzySource implements the Source interface for fuzzy.Find
type fuzzySource []string

func (s fuzzySource) String(i int) string {
	return s[i]
}

func (s fuzzySource) Len() int {
	return len(s)
}
