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

func RenderMiniTileOOB(playerID string, index int, marked bool) string {
	class := "mini-tile"
	if marked {
		class = "mini-tile marked"
	}
	return fmt.Sprintf(`<div id="mini-tile-%s-%d" class="%s" hx-swap-oob="true"></div>`, template.HTMLEscapeString(playerID), index, class)
}

func RenderMiniTile(playerID string, index int, marked bool) string {
	class := "mini-tile"
	if marked {
		class = "mini-tile marked"
	}
	return fmt.Sprintf(`<div id="mini-tile-%s-%d" class="%s"></div>`, template.HTMLEscapeString(playerID), index, class)
}

func RenderEvent(e Event, receiverID string, size int) string {
	switch e.Type {
	case BoardStateEvent:
		if e.PlayerID != receiverID {
			return ""
		}
		var buf bytes.Buffer
		fmt.Fprintf(&buf, `<div id="game-board-container" class="board" style="grid-template-columns: repeat(%d, 1fr);" hx-swap-oob="true">`, size)
		for i, word := range e.Board.Words {
			buf.WriteString(RenderTile(i, word, e.Board.Marks[i], false))
		}
		buf.WriteString(`</div>`)
		return buf.String()

	case OpponentBoardsStateEvent:
		if len(e.Opponents) == 0 {
			return `<div id="opponents-sidebar" class="opponents-sidebar" hx-swap-oob="true" style="display: none;"></div>`
		}
		if len(e.Opponents) > 5 {
			return `<div id="opponents-sidebar" class="opponents-sidebar" hx-swap-oob="true" style="display: none;"></div>`
		}

		var buf bytes.Buffer
		buf.WriteString(`<div id="opponents-sidebar" class="opponents-sidebar" hx-swap-oob="true">`)
		for _, opp := range e.Opponents {
			label := opp.PlayerName
			if label == "" {
				label = opp.PlayerID
				if len(label) > 10 {
					label = "Player " + label[len(label)-4:]
				}
			}
			buf.WriteString(`<div class="mini-board-wrapper">`)
			fmt.Fprintf(&buf, `<div class="mini-board-label">%s</div>`, template.HTMLEscapeString(label))
			fmt.Fprintf(&buf, `<div id="mini-board-%s" class="mini-board" style="grid-template-columns: repeat(%d, 1fr);">`, template.HTMLEscapeString(opp.PlayerID), size)
			for i := range opp.Board.Words {
				buf.WriteString(RenderMiniTile(opp.PlayerID, i, opp.Board.Marks[i]))
			}
			buf.WriteString(`</div></div>`)
		}
		buf.WriteString(`</div>`)
		return buf.String()

	case TileMarkedEvent:
		res := ""
		if e.PlayerID == receiverID || e.PlayerID == "global" {
			res += RenderTile(*e.TileIndex, *e.TileWord, true, true)
		} else {
			res += RenderMiniTileOOB(e.PlayerID, *e.TileIndex, true)
		}

		if e.PlayerID != receiverID {
			label := e.PlayerName
			if label == "" {
				label = e.PlayerID
			}
			msg := template.HTMLEscapeString(label) + " marked " + template.HTMLEscapeString(*e.TileWord)
			res += fmt.Sprintf(`<div id="event-log" hx-swap-oob="afterbegin"><div class="log-entry">%s<script>highlightWord("%s");</script></div></div>`, msg, template.JSEscapeString(*e.TileWord))
		}
		return res

	case TileUnmarkedEvent:
		res := ""
		if e.PlayerID == receiverID || e.PlayerID == "global" {
			res += RenderTile(*e.TileIndex, *e.TileWord, false, true)
		} else {
			res += RenderMiniTileOOB(e.PlayerID, *e.TileIndex, false)
		}
		return res

	case BingoEvent:
		if e.PlayerID == "global" {
			return ""
		}
		msg := "BINGO!"
		if e.PlayerID != receiverID {
			label := e.PlayerName
			if label == "" {
				label = e.PlayerID
			}
			msg = template.HTMLEscapeString(label) + " got BINGO!"
		}
		return fmt.Sprintf(`<div id="event-log" hx-swap-oob="afterbegin">
			<div class="log-entry bingo">%s</div>
		</div>`, msg)

	case OneToBingoEvent:
		if e.PlayerID == "global" {
			return ""
		}
		label := "You are"
		if e.PlayerID != receiverID {
			name := e.PlayerName
			if name == "" {
				name = e.PlayerID
			}
			label = template.HTMLEscapeString(name) + " is"
		}
		return fmt.Sprintf(`<div id="event-log" hx-swap-oob="afterbegin">
			<div class="log-entry one-to-bingo">%s one tile away from BINGO!</div>
		</div>`, label)

	default:
		return ""
	}
}
