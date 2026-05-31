package game

import (
	"strconv"
	"testing"
)

func generateWordlist(size int) []string {
	words := make([]string, size)
	for i := range size {
		words[i] = "word" + strconv.Itoa(i)
	}
	return words
}

func shuffleStub(n int, swap func(i, j int)) {}

func TestNewBoard(t *testing.T) {
	t.Run("5x5 with free space (24 words)", func(t *testing.T) {
		words := generateWordlist(24)
		board, err := NewBoard(5, words, true, shuffleStub)

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		centerIndex := 12

		if board.Words[centerIndex] != "" {
			t.Errorf("expected center tile to be empty, got '%s'", board.Words[centerIndex])
		}
		if board.Marks[centerIndex] != true {
			t.Errorf("expected center tile mark to be true")
		}
		if board.Words[0] == "" {
			t.Errorf("expected tile 0 to be filled, but it was empty")
		}
	})

	t.Run("5x5 without free space (25 words)", func(t *testing.T) {
		words := generateWordlist(25)
		board, err := NewBoard(5, words, false, func(n int, swap func(i, j int)) {})

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		centerIndex := 12

		if board.Words[centerIndex] == "" {
			t.Errorf("expected center tile to have a word, but it was empty")
		}
		if board.Marks[centerIndex] == true {
			t.Errorf("expected center tile mark to be false initially")
		}
	})
}

func TestMarkTile(t *testing.T) {
	words := generateWordlist(24)

	word0 := "word0"
	word12 := "word12"

	tests := []struct {
		name         string
		initialMarks []bool
		clickIndex   int
		expectMark   bool
		expectWord   *string
	}{
		{
			name: "mark free space",
			initialMarks: []bool{
				false, false, false, false, false,
				false, false, false, false, false,
				false, false, true, false, false,
				false, false, false, false, false,
				false, false, false, false, false,
			},
			clickIndex: 12,
			expectMark: true,
			expectWord: nil,
		},
		{
			name: "mark a normal tile (index 0)",
			initialMarks: []bool{
				false, false, false, false, false,
				false, false, false, false, false,
				false, false, true, false, false,
				false, false, false, false, false,
				false, false, false, false, false,
			},
			clickIndex: 0,
			expectMark: true,
			expectWord: &word0,
		},
		{
			name: "unmark an already marked tile (index 13)",
			initialMarks: []bool{
				false, false, false, false, false,
				false, false, false, false, false,
				false, false, true, true, false,
				false, false, false, false, false,
				false, false, false, false, false,
			},
			clickIndex: 13,
			expectMark: false,
			expectWord: &word12,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			board, _ := NewBoard(5, words, true, func(n int, swap func(i, j int)) {})

			board.Marks = tt.initialMarks

			gotWord, isMarked := board.MarkTile(tt.clickIndex)

			if board.Marks[tt.clickIndex] != tt.expectMark {
				t.Errorf("expected board mark to be %v, got %v", tt.expectMark, board.Marks[tt.clickIndex])
			}

			if isMarked != tt.expectMark {
				t.Errorf("expected isMarked to be %t, got %t", tt.expectMark, isMarked)
			}

			if tt.expectWord == nil && gotWord != nil {
				t.Errorf("expected gotWord to be nil, got %q", *gotWord)
			} else if tt.expectWord != nil && gotWord == nil {
				t.Errorf("expected gotWord to be %q, got nil", *tt.expectWord)
			} else if tt.expectWord != nil && gotWord != nil && *tt.expectWord != *gotWord {
				t.Errorf("expected gotWord to be %q, got %q", *tt.expectWord, *gotWord)
			}
		})
	}
}

func TestAnalyze(t *testing.T) {
	words := generateWordlist(24)
	board, _ := NewBoard(5, words, true, func(n int, swap func(i, j int)) {})

	tests := []struct {
		name             string
		marks            []bool
		lastChangedIndex int
		expectedBingo    bool
		expectedMax      int
	}{
		{
			name: "empty board with just free space",
			marks: []bool{
				false, false, false, false, false,
				false, false, false, false, false,
				false, false, true, false, false,
				false, false, false, false, false,
				false, false, false, false, false,
			},
			lastChangedIndex: 12,
			expectedBingo:    false,
			expectedMax:      1,
		},
		{
			name: "horizontal bingo",
			marks: []bool{
				false, false, false, false, false,
				false, false, false, false, false,
				true, true, true, true, true,
				false, false, false, false, false,
				false, false, false, false, false,
			},
			lastChangedIndex: 10,
			expectedBingo:    true,
			expectedMax:      5,
		},
		{
			name: "vertical bingo",
			marks: []bool{
				true, false, false, false, false,
				true, false, false, false, false,
				true, false, false, false, false,
				true, false, false, false, false,
				true, false, false, false, false,
			},
			lastChangedIndex: 20,
			expectedBingo:    true,
			expectedMax:      5,
		},
		{
			name: "main diagonal bingo",
			marks: []bool{
				true, false, false, false, false,
				false, true, false, false, false,
				false, false, true, false, false,
				false, false, false, true, false,
				false, false, false, false, true,
			},
			lastChangedIndex: 24,
			expectedBingo:    true,
			expectedMax:      5,
		},
		{
			name: "anti diagonal bingo",
			marks: []bool{
				false, false, false, false, true,
				false, false, false, true, false,
				false, false, true, false, false,
				false, true, false, false, false,
				true, false, false, false, false,
			},
			lastChangedIndex: 4,
			expectedBingo:    true,
			expectedMax:      5,
		},
		{
			name: "one away horizontal",
			marks: []bool{
				false, false, false, false, false,
				false, false, false, false, false,
				true, true, true, true, false,
				false, false, false, false, false,
				false, false, false, false, false,
			},
			lastChangedIndex: 13,
			expectedBingo:    false,
			expectedMax:      4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			board.Marks = tt.marks
			analysis := board.Analyze(tt.lastChangedIndex)

			if analysis.HasBingo != tt.expectedBingo {
				t.Errorf("expected HasBingo %v, got %v", tt.expectedBingo, analysis.HasBingo)
			}
			if analysis.MaxInLine != tt.expectedMax {
				t.Errorf("expected MaxInLine %d, got %d", tt.expectedMax, analysis.MaxInLine)
			}
		})
	}
}
