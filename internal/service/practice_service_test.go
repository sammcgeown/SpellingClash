package service

import (
	"spellingclash/internal/models"
	"testing"
)

func TestWordsToIDString(t *testing.T) {
	tests := []struct {
		name     string
		words    []models.Word
		expected string
	}{
		{
			name:     "empty slice",
			words:    []models.Word{},
			expected: "",
		},
		{
			name: "single word",
			words: []models.Word{
				{ID: 1, WordText: "cat"},
			},
			expected: "1",
		},
		{
			name: "multiple words",
			words: []models.Word{
				{ID: 1, WordText: "cat"},
				{ID: 2, WordText: "dog"},
				{ID: 3, WordText: "bird"},
			},
			expected: "1,2,3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := wordsToIDString(tt.words)
			if result != tt.expected {
				t.Errorf("wordsToIDString() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestReorderWordsByIDs(t *testing.T) {
	words := []models.Word{
		{ID: 1, WordText: "cat"},
		{ID: 2, WordText: "dog"},
		{ID: 3, WordText: "bird"},
	}

	tests := []struct {
		name     string
		words    []models.Word
		idString string
		expected []string
	}{
		{
			name:     "empty string returns original order",
			words:    words,
			idString: "",
			expected: []string{"cat", "dog", "bird"},
		},
		{
			name:     "reorder words",
			words:    words,
			idString: "3,1,2",
			expected: []string{"bird", "cat", "dog"},
		},
		{
			name:     "partial reorder",
			words:    words,
			idString: "2,3",
			expected: []string{"dog", "bird"},
		},
		{
			name:     "invalid ID ignored",
			words:    words,
			idString: "1,999,2",
			expected: []string{"cat", "dog"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := reorderWordsByIDs(tt.words, tt.idString)
			if len(result) != len(tt.expected) {
				t.Fatalf("length mismatch: got %d, want %d", len(result), len(tt.expected))
			}
			for i, word := range result {
				if word.WordText != tt.expected[i] {
					t.Errorf("position %d: got %v, want %v", i, word.WordText, tt.expected[i])
				}
			}
		})
	}
}