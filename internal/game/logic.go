package game

import (
	"errors"
)

type Board struct {
	Size  int      `json:"size"`
	Words []string `json:"words"`
	Marks []bool   `json:"marks"`
}

type ShuffleFunc func(n int, swap func(i, j int))

func NewBoard(size int, wordlist []string, freeSpace bool, shuffle ShuffleFunc) (*Board, error) {
	if size%2 == 0 {
		return nil, errors.New("board must have odd size")
	}
	length := size * size

	requiredLength := length
	if freeSpace {
		requiredLength--
	}

	if len(wordlist) < requiredLength {
		return nil, errors.New("wordlist isn't enough for the board")
	}

	words := make([]string, length)
	marks := make([]bool, length)

	pool := make([]string, len(wordlist))
	copy(pool, wordlist)
	shuffle(len(pool), func(i, j int) {
		pool[i], pool[j] = pool[j], pool[i]
	})

	centerIndex := length / 2
	poolIndex := 0
	for i := range length {
		if freeSpace && (i == centerIndex) {
			words[i] = ""
			marks[i] = true
			continue
		}

		words[i] = pool[poolIndex]
		poolIndex++
	}

	return &Board{
		size,
		words,
		marks,
	}, nil
}

func (b *Board) MarkTile(index int) (*string, bool) {
	if b.Words[index] == "" {
		return nil, true
	}

	b.Marks[index] = !b.Marks[index]
	return &b.Words[index], b.Marks[index]
}

type BoardAnalysis struct {
	HasBingo  bool
	MaxInLine int
}

func (b *Board) Analyze(lastChangedIndex int) BoardAnalysis {
	row := lastChangedIndex / b.Size
	col := lastChangedIndex % b.Size

	onMain := (row == col)
	onAnti := (row+col == b.Size-1)

	rowCount, colCount, mainDiagCount, antiDiagCount := 0, 0, 0, 0

	for i := range b.Size {
		if b.Marks[(row*b.Size)+i] {
			rowCount++
		}

		if b.Marks[(i*b.Size)+col] {
			colCount++
		}

		if onMain && b.Marks[(i*b.Size+i)] {
			mainDiagCount++
		}

		if onAnti && b.Marks[i*b.Size+b.Size-1-i] {
			antiDiagCount++
		}
	}

	return BoardAnalysis{
		HasBingo:  (rowCount == b.Size) || (colCount == b.Size) || (mainDiagCount == b.Size) || (antiDiagCount == b.Size),
		MaxInLine: max(rowCount, colCount, mainDiagCount, antiDiagCount),
	}
}
