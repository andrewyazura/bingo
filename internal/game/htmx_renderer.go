package game

import (
	"bytes"
	"fmt"
	"html/template"
)

func RenderTile(index int, word string, marked bool, isOOB bool) string {
	class := "tile"
	if marked {
		class = "tile marked"
	}

	if word == "" {
		word = "FREE SPACE"
	}

	oob := ""
	if isOOB {
		oob = ` hx-swap-oob="true"`
	}

	return fmt.Sprintf(`
		<button id="tile-%d" class="%s" ws-send hx-vals='{"tileIndex": %d}'%s>
			%s
		</button>
	`, index, class, index, oob, template.HTMLEscapeString(word))
}

func RenderEvent(e Event, size int) string {
	switch e.Type {
	case BoardStateEvent:
		var buf bytes.Buffer
		fmt.Fprintf(&buf, `<div id="game-board-container" class="board" style="grid-template-columns: repeat(%d, 1fr);" hx-swap-oob="true">`, size)
		for i, word := range e.Board.Words {
			buf.WriteString(RenderTile(i, word, e.Board.Marks[i], false))
		}
		buf.WriteString(`</div>`)
		return buf.String()

	case TileMarkedEvent:
		return RenderTile(*e.TileIndex, *e.TileWord, true, true)

	case TileUnmarkedEvent:
		return RenderTile(*e.TileIndex, *e.TileWord, false, true)

	case BingoEvent:
		return `
		<div id="victory-modal" hx-swap-oob="true">
			<div class="victory-text">BINGO!</div>
			<button class="cs-btn" onclick="document.getElementById('victory-modal').remove()">Keep Playing</button>
		</div>`

	default:
		return ""
	}
}
