package main

import (
	rl "github.com/waozixyz/kryon/go/kryon"
)

const (
	winW                  = 1180
	winH                  = 720
	rowH                  = 26
	menuBarH              = 30
	toolbarTop            = menuBarH
	toolbarH              = 44
	formulaBarTop         = toolbarTop + toolbarH + 10
	formulaBarH           = 30
	tableViewTop          = formulaBarTop + formulaBarH + 10
	tableTop              = tableViewTop + 40
	tableBot              = 694
	blankRowsBeforeTotals = 4
)

// column anchors: x of left-aligned columns, right edge of right-aligned ones
const (
	xSec   = 24
	xLabel = 112
	xCoin  = 228
	rUnits = 610
	rRate  = 730
	rPct   = 806
	rUsd   = 926
	rEur   = 1042
	rDusd  = 1156
)

var (
	visibleDataCols     = []int{0, 1, 2, 3, 4, 6, 7}
	visibleTableColumns = []string{"", "A", "B", "C", "D", "E", "G", "H"}
)

// displayRow is one rendered table row backed by a workbook row.
type displayRow struct {
	kind int
	sec  string
	idx  int // workbook row index for kind 0
}

const (
	dispRow = iota
	dispBlank
	dispTotalPending
	dispTotal
)

func (a *app) dispRows() []displayRow {
	var out []displayRow
	for i := range a.wb.Rows {
		out = append(out, displayRow{dispRow, a.wb.Rows[i].Section, i})
	}
	for i := 0; i < blankRowsBeforeTotals; i++ {
		out = append(out, displayRow{kind: dispBlank, idx: len(a.wb.Rows) + i})
	}
	out = append(out, displayRow{kind: dispTotalPending, idx: -1})
	out = append(out, displayRow{kind: dispTotal, idx: -1})
	return out
}

// --- text helpers (embedded fonts, float32 like core/gui_theme.go) ----------

func txt(s string, x, y, size float32, c rl.Color) {
	rl.Text(s, int32(x), int32(y), int32(size), c)
}

func btxt(s string, x, y, size float32, c rl.Color) {
	rl.Text(s, int32(x), int32(y), int32(size), c)
}

func wtxt(s string, size float32) float32 {
	return rl.MeasureTextEx(rl.Font{}, s, size, 1).X
}

func rtxt(s string, right, y, size float32, c rl.Color) { txt(s, right-wtxt(s, size), y, size, c) }

// ellip truncates text with a trailing "…" so it fits maxW.
func ellip(s string, maxW, size float32) string {
	if s == "" || wtxt(s, size) <= maxW {
		return s
	}
	r := []rune(s)
	for len(r) > 1 && wtxt(string(r)+"…", size) > maxW {
		r = r[:len(r)-1]
	}
	return string(r) + "…"
}

func signColor(v float64) rl.Color {
	switch {
	case v > 0:
		return colGreen
	case v < 0:
		return colRed
	}
	return colDim
}
