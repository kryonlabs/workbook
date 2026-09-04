package main

import (
	"fmt"
	"strconv"
	"strings"

	rl "github.com/waozixyz/kryon/go/kryon"
)

var formatPalette = []rl.Color{
	{R: 228, G: 230, B: 235, A: 255},
	{R: 86, G: 170, B: 255, A: 255},
	{R: 96, G: 226, B: 142, A: 255},
	{R: 255, G: 207, B: 86, A: 255},
	{R: 255, G: 96, B: 164, A: 255},
}

var backgroundPalette = []rl.Color{
	{R: 30, G: 48, B: 70, A: 255},
	{R: 46, G: 67, B: 50, A: 255},
	{R: 72, G: 55, B: 28, A: 255},
	{R: 74, G: 36, B: 56, A: 255},
	{R: 50, G: 52, B: 60, A: 255},
}

func cellFormatKey(row, col int) string {
	return strconv.Itoa(row) + ":" + strconv.Itoa(col)
}

func splitCellFormatKey(key string) (int, int, bool) {
	parts := strings.Split(key, ":")
	if len(parts) != 2 {
		return 0, 0, false
	}
	row, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, false
	}
	col, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, false
	}
	return row, col, true
}

func colorHex(c rl.Color) string {
	return fmt.Sprintf("#%02x%02x%02x%02x", c.R, c.G, c.B, c.A)
}

func parseColorHex(s string) (rl.Color, bool) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "#")
	if len(s) != 6 && len(s) != 8 {
		return rl.Color{}, false
	}
	v, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return rl.Color{}, false
	}
	if len(s) == 6 {
		return rl.Color{R: uint8(v >> 16), G: uint8(v >> 8), B: uint8(v), A: 255}, true
	}
	return rl.Color{R: uint8(v >> 24), G: uint8(v >> 16), B: uint8(v >> 8), A: uint8(v)}, true
}

func (a *app) cellFormat(row, col int) CellFormat {
	if a.wb.CellFormats == nil {
		return CellFormat{}
	}
	return a.wb.CellFormats[cellFormatKey(row, col)]
}

func (a *app) setCellTextColor(row, col int, color rl.Color) {
	a.ensureCellFormats()
	key := cellFormatKey(row, col)
	format := a.wb.CellFormats[key]
	format.TextColor = colorHex(color)
	a.setCellFormat(key, format)
}

func (a *app) setCellBackgroundColor(row, col int, color rl.Color) {
	a.ensureCellFormats()
	key := cellFormatKey(row, col)
	format := a.wb.CellFormats[key]
	format.BackgroundColor = colorHex(color)
	a.setCellFormat(key, format)
}

func (a *app) clearCellFormat(row, col int) {
	if a.wb.CellFormats == nil {
		return
	}
	delete(a.wb.CellFormats, cellFormatKey(row, col))
}

func (a *app) setCellFormat(key string, format CellFormat) {
	if strings.TrimSpace(format.TextColor) == "" && strings.TrimSpace(format.BackgroundColor) == "" && strings.TrimSpace(format.Conditional) == "" {
		delete(a.wb.CellFormats, key)
		return
	}
	a.wb.CellFormats[key] = format
}

func (a *app) toggleCellConditional(row, col int) {
	a.ensureCellFormats()
	key := cellFormatKey(row, col)
	format := a.wb.CellFormats[key]
	if format.Conditional == "sign" {
		format.Conditional = ""
	} else {
		format.Conditional = "sign"
	}
	a.setCellFormat(key, format)
}

func (a *app) ensureCellFormats() {
	if a.wb.CellFormats == nil {
		a.wb.CellFormats = map[string]CellFormat{}
	}
}

func (a *app) shiftCellFormatsForInsert(pos int) {
	if len(a.wb.CellFormats) == 0 {
		return
	}
	next := map[string]CellFormat{}
	for key, format := range a.wb.CellFormats {
		row, col, ok := splitCellFormatKey(key)
		if !ok {
			continue
		}
		if row >= pos {
			row++
		}
		next[cellFormatKey(row, col)] = format
	}
	a.wb.CellFormats = next
}

func (a *app) shiftCellFormatsForDelete(pos int) {
	if len(a.wb.CellFormats) == 0 {
		return
	}
	next := map[string]CellFormat{}
	for key, format := range a.wb.CellFormats {
		row, col, ok := splitCellFormatKey(key)
		if !ok || row == pos {
			continue
		}
		if row > pos {
			row--
		}
		next[cellFormatKey(row, col)] = format
	}
	a.wb.CellFormats = next
}

func (a *app) shiftCellFormatsForColumnInsert(pos int) {
	if len(a.wb.CellFormats) == 0 {
		return
	}
	next := map[string]CellFormat{}
	for key, format := range a.wb.CellFormats {
		row, col, ok := splitCellFormatKey(key)
		if !ok {
			continue
		}
		if col >= pos {
			col++
		}
		if col < 8 {
			next[cellFormatKey(row, col)] = format
		}
	}
	a.wb.CellFormats = next
}

func (a *app) shiftCellFormatsForColumnDelete(pos int) {
	if len(a.wb.CellFormats) == 0 {
		return
	}
	next := map[string]CellFormat{}
	for key, format := range a.wb.CellFormats {
		row, col, ok := splitCellFormatKey(key)
		if !ok || col == pos {
			continue
		}
		if col > pos {
			col--
		}
		next[cellFormatKey(row, col)] = format
	}
	a.wb.CellFormats = next
}

func nextPaletteColor(current string, palette []rl.Color) rl.Color {
	if len(palette) == 0 {
		return colAccent
	}
	if current == "" {
		return palette[0]
	}
	for i, color := range palette {
		if strings.EqualFold(current, colorHex(color)) {
			return palette[(i+1)%len(palette)]
		}
	}
	return palette[0]
}
