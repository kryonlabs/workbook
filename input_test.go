package main

import (
	"os"
	"path/filepath"
	"testing"

	rl "github.com/waozixyz/kryon/go/kryon"
)

func tableTestRowY(row int) float32 {
	return float32(tableTop + row*rowH + rowH/2)
}

func tableTestHeaderY() float32 {
	return float32(tableViewTop + rowH/2)
}

func TestRuneEditorHelpers(t *testing.T) {
	text, cur := insertAt("waozi", 5, " åé")
	if text != "waozi åé" || cur != 8 {
		t.Fatalf("insertAt unicode = %q, %d; want %q, 8", text, cur, "waozi åé")
	}

	text, cur = deleteBefore(text, cur)
	if text != "waozi å" || cur != 7 {
		t.Fatalf("deleteBefore unicode = %q, %d; want %q, 7", text, cur, "waozi å")
	}

	text, cur = insertAt(text, 6, "X")
	if text != "waozi Xå" || cur != 7 {
		t.Fatalf("middle insert = %q, %d; want %q, 7", text, cur, "waozi Xå")
	}

	text, cur = deleteAt(text, 7)
	if text != "waozi X" || cur != 7 {
		t.Fatalf("deleteAt unicode = %q, %d; want %q, 7", text, cur, "waozi X")
	}
}

func TestClickingAnotherCellLeavesEditModeAndSelectsCell(t *testing.T) {
	a := &app{
		dir: t.TempDir(),
		wb: &Workbook{
			Rows: []Row{
				{Section: "crypto", Label: "cold wallet", ID: "bitcoin", Units: 0.25},
				{Section: "crypto", Label: "exchange", ID: "ethereum", Units: 2.5},
			},
			Rates: map[string]float64{"bitcoin": 65000, "ethereum": 3200},
		},
	}
	u := &uiState{
		selRow:    0,
		selCol:    2,
		disp:      a.dispRows(),
		editing:   true,
		editText:  make([]byte, 128),
		editCur:   int32(len("edited wallet")),
		editFocus: true,
		editIdx:   0,
	}
	copy(u.editText, "edited wallet")

	host := rl.NewHost(rl.AppConfig{Width: winW, Height: winH})
	defer host.Close()
	defer rl.SetRuntime(nil)
	host.QueueTap(280, tableTestRowY(1)) // row 1, C/coin column in the native table body.
	host.Draw(func() {
		rl.BeginFrame()
		a.drawNativeTable(u)
		rl.EndFrame()
	})

	if u.editing {
		t.Fatal("clicking another table cell left the old editor active")
	}
	if got := a.wb.Rows[0].Label; got != "edited wallet" {
		t.Fatalf("committed label = %q, want edited wallet", got)
	}
	if u.selRow != 1 || u.selCol != 3 {
		t.Fatalf("selection = row %d col %d, want row 1 col 3", u.selRow, u.selCol)
	}
}

func TestEscapeLeavesEditModeAndKeepsCellSelected(t *testing.T) {
	a := &app{
		dir: t.TempDir(),
		wb: &Workbook{
			Rows: []Row{
				{Section: "crypto", Label: "cold wallet", ID: "bitcoin", Units: 0.25},
			},
			Rates: map[string]float64{"bitcoin": 65000},
		},
	}
	u := &uiState{
		selRow:    0,
		selCol:    2,
		disp:      a.dispRows(),
		editing:   true,
		editText:  make([]byte, 128),
		editCur:   int32(len("discard me")),
		editFocus: true,
		editIdx:   0,
	}
	copy(u.editText, "discard me")

	host := rl.NewHost(rl.AppConfig{Width: winW, Height: winH})
	defer host.Close()
	defer rl.SetRuntime(nil)
	host.QueueKey(rl.KeyEscape)
	host.Draw(func() {
		rl.BeginFrame()
		a.drawEditor(u)
		rl.EndFrame()
	})

	if u.editing {
		t.Fatal("escape left the editor active")
	}
	if u.editFocus {
		t.Fatal("escape left text input focus active")
	}
	if u.selRow != 0 || u.selCol != 2 {
		t.Fatalf("selection = row %d col %d, want row 0 col 2", u.selRow, u.selCol)
	}
	if got := a.wb.Rows[0].Label; got != "cold wallet" {
		t.Fatalf("escape committed label = %q, want cold wallet", got)
	}
	if _, err := os.Stat(filepath.Join(a.dir, workbookFile)); !os.IsNotExist(err) {
		t.Fatalf("escape wrote workbook file, stat err=%v", err)
	}
	if got := host.Focus(); got != 2001 {
		t.Fatalf("focus = %d, want table focus 2001", got)
	}
}

func TestEscapeLeavesEditModeDuringFullFrame(t *testing.T) {
	a := &app{
		dir: t.TempDir(),
		wb: &Workbook{
			Rows: []Row{
				{Section: "crypto", Label: "cold wallet", ID: "bitcoin", Units: 0.25},
			},
			Rates: map[string]float64{"bitcoin": 65000},
		},
	}
	u := &uiState{selRow: 0, selCol: 2, disp: a.dispRows(), editText: make([]byte, 128)}
	a.startEdit(u, u.disp[0], dataColFromTableCol(int32(u.selCol)), true)
	clear(u.editText)
	copy(u.editText, "discard me")
	u.editCur = int32(len("discard me"))

	host := rl.NewHost(rl.AppConfig{Width: winW, Height: winH})
	defer host.Close()
	defer rl.SetRuntime(nil)
	host.QueueKey(rl.KeyEscape)
	host.Draw(func() { draw(a, u) })

	if u.editing {
		t.Fatal("escape left the editor active after a full frame")
	}
	if u.quit {
		t.Fatal("escape quit the app instead of leaving edit mode")
	}
	if got := a.wb.Rows[0].Label; got != "cold wallet" {
		t.Fatalf("escape committed label = %q, want cold wallet", got)
	}
	if u.selRow != 0 || u.selCol != 2 {
		t.Fatalf("selection = row %d col %d, want row 0 col 2", u.selRow, u.selCol)
	}
}

func TestOnlyEnterCommitsCellEdit(t *testing.T) {
	a := &app{
		dir: t.TempDir(),
		wb: &Workbook{
			Rows: []Row{
				{Section: "crypto", Label: "cold wallet", ID: "bitcoin", Units: 0.25},
			},
			Rates: map[string]float64{"bitcoin": 65000},
		},
	}
	u := &uiState{selRow: 0, selCol: 2, disp: a.dispRows(), editText: make([]byte, 128)}
	host := rl.NewHost(rl.AppConfig{Width: winW, Height: winH})
	defer host.Close()
	defer rl.SetRuntime(nil)

	a.startEdit(u, u.disp[0], dataColFromTableCol(int32(u.selCol)), true)
	clear(u.editText)
	copy(u.editText, "discarded")
	u.editCur = int32(len("discarded"))
	host.QueueKey(rl.KeyTab)
	host.Draw(func() {
		rl.BeginFrame()
		a.drawEditor(u)
		rl.EndFrame()
	})
	if got := a.wb.Rows[0].Label; got != "cold wallet" {
		t.Fatalf("tab committed label = %q, want cold wallet", got)
	}

	u.selRow, u.selCol = 0, 2
	a.startEdit(u, u.disp[0], dataColFromTableCol(int32(u.selCol)), true)
	clear(u.editText)
	copy(u.editText, "also discarded")
	u.editCur = int32(len("also discarded"))
	u.editFocus = false
	host.Draw(func() {
		rl.BeginFrame()
		a.drawEditor(u)
		rl.EndFrame()
	})
	if got := a.wb.Rows[0].Label; got != "cold wallet" {
		t.Fatalf("focus loss committed label = %q, want cold wallet", got)
	}

	u.selRow, u.selCol = 0, 2
	a.startEdit(u, u.disp[0], dataColFromTableCol(int32(u.selCol)), true)
	clear(u.editText)
	copy(u.editText, "vault")
	u.editCur = int32(len("vault"))
	host.QueueKey(rl.KeyEnter)
	host.Draw(func() {
		rl.BeginFrame()
		a.drawEditor(u)
		rl.EndFrame()
	})
	if got := a.wb.Rows[0].Label; got != "vault" {
		t.Fatalf("enter did not commit label = %q, want vault", got)
	}
}

func TestDirectTypingIntoCellCommitsReplacement(t *testing.T) {
	a := &app{
		dir: t.TempDir(),
		wb: &Workbook{
			Rows: []Row{
				{Section: "crypto", Label: "cold wallet", ID: "bitcoin", Units: 0.25},
			},
			Rates: map[string]float64{"bitcoin": 65000},
		},
	}
	u := &uiState{selRow: 0, selCol: 2, disp: a.dispRows(), editText: make([]byte, 128)}

	a.startEdit(u, u.disp[0], dataColFromTableCol(int32(u.selCol)), false)
	copy(u.editText, "replacement")
	u.editCur = int32(len("replacement"))
	a.commitCell(u)

	if got := a.wb.Rows[0].Label; got != "replacement" {
		t.Fatalf("direct typed label = %q, want replacement", got)
	}
}

func TestFormulaUnitsEditStartsWithEquals(t *testing.T) {
	a := &app{
		dir: t.TempDir(),
		wb: &Workbook{
			Rows: []Row{
				{Section: "crypto", Label: "cold wallet", ID: "bitcoin", Units: 0.35, Expr: "0.1+0.25"},
			},
			Rates: map[string]float64{"bitcoin": 65000},
		},
	}
	u := &uiState{selRow: 0, selCol: 4, disp: a.dispRows(), editText: make([]byte, 128)}

	a.startEdit(u, u.disp[0], dataColFromTableCol(int32(u.selCol)), true)

	if got := editString(u); got != "=0.1+0.25" {
		t.Fatalf("formula edit text = %q, want =0.1+0.25", got)
	}
	if u.editCur != int32(len("=0.1+0.25")) {
		t.Fatalf("formula cursor = %d, want end of formula", u.editCur)
	}
}

func TestTableStartsWithNoSelectedColumn(t *testing.T) {
	a := &app{
		dir: t.TempDir(),
		wb: &Workbook{
			Rows: []Row{
				{Section: "crypto", Label: "cold wallet", ID: "bitcoin", Units: 0.25},
			},
			Rates: map[string]float64{"bitcoin": 65000},
		},
	}
	u := &uiState{selRow: -1, selCol: -1, disp: a.dispRows(), editText: make([]byte, 128)}
	host := rl.NewHost(rl.AppConfig{Width: winW, Height: winH})
	defer host.Close()
	defer rl.SetRuntime(nil)

	host.Draw(func() {
		rl.BeginFrame()
		a.drawNativeTable(u)
		rl.EndFrame()
	})

	for _, op := range host.FrameOps() {
		if op.Selected {
			t.Fatalf("initial table painted selected op: %#v", op)
		}
	}
}

func TestDisplayRowsContainOnlyRows(t *testing.T) {
	a := &app{
		wb: &Workbook{
			Rows: []Row{
				{Section: "crypto", Label: "cold wallet", ID: "bitcoin", Units: 0.25},
				{Section: "banks", Label: "checking", ID: "USD", Units: 100},
			},
			Rates: map[string]float64{"bitcoin": 65000, "USD": 1},
		},
	}
	rows := a.dispRows()
	if len(rows) != len(a.wb.Rows)+blankRowsBeforeTotals+2 {
		t.Fatalf("display rows = %d, want %d rows plus blank rows and totals", len(rows), len(a.wb.Rows)+blankRowsBeforeTotals+2)
	}
	for i, row := range rows[:len(a.wb.Rows)] {
		if row.kind != dispRow {
			t.Fatalf("display row %d kind = %d, want data row", i, row.kind)
		}
	}
	for i, row := range rows[len(a.wb.Rows) : len(a.wb.Rows)+blankRowsBeforeTotals] {
		if row.kind != dispBlank {
			t.Fatalf("blank row %d kind = %d, want blank row", i, row.kind)
		}
	}
	if rows[len(rows)-2].kind != dispTotalPending || rows[len(rows)-1].kind != dispTotal {
		t.Fatalf("last rows kinds = %d,%d, want total+pending,total", rows[len(rows)-2].kind, rows[len(rows)-1].kind)
	}
}

func TestTableRowsShowSectionInEveryDataRow(t *testing.T) {
	a := &app{
		wb: &Workbook{
			Rows: []Row{
				{Section: "crypto", Label: "cold wallet", ID: "bitcoin", Units: 0.25},
				{Section: "crypto", Label: "exchange", ID: "ethereum", Units: 2.5},
			},
			Rates: map[string]float64{"bitcoin": 65000, "ethereum": 3200},
		},
	}
	u := &uiState{disp: a.dispRows()}
	rows := a.workbookTableRows(u)
	for i, row := range rows[:len(a.wb.Rows)] {
		if got := row.Cells[1]; got != "crypto" {
			t.Fatalf("row %d section cell = %q, want crypto", i, got)
		}
	}
}

func TestFormulaUnitsDoNotAddCoinMarker(t *testing.T) {
	a := &app{
		wb: &Workbook{
			Rows: []Row{
				{Section: "crypto", Label: "cold wallet", ID: "bitcoin", Units: 0.35, Expr: "0.1+0.25"},
			},
			Rates: map[string]float64{"bitcoin": 65000},
		},
	}
	u := &uiState{disp: a.dispRows()}
	rows := a.workbookTableRows(u)
	if got := rows[0].Cells[3]; got != "btc" {
		t.Fatalf("coin cell = %q, want btc without formula marker", got)
	}
	if got := rows[0].Cells[4]; got != "0.35" {
		t.Fatalf("units cell = %q, want evaluated units", got)
	}
}

func TestSpreadsheetHeadersSelectRowsAndColumns(t *testing.T) {
	a := &app{
		dir: t.TempDir(),
		wb: &Workbook{
			Rows: []Row{
				{Section: "crypto", Label: "cold wallet", ID: "bitcoin", Units: 0.25},
				{Section: "crypto", Label: "exchange", ID: "ethereum", Units: 2.5},
			},
			Rates: map[string]float64{"bitcoin": 65000, "ethereum": 3200},
		},
	}
	u := &uiState{selRow: -1, selCol: -1, disp: a.dispRows(), editText: make([]byte, 128)}

	host := rl.NewHost(rl.AppConfig{Width: winW, Height: winH})
	defer host.Close()
	defer rl.SetRuntime(nil)

	host.QueueTap(40, tableTestRowY(1)) // row 1 number column.
	host.Draw(func() {
		rl.BeginFrame()
		a.drawNativeTable(u)
		rl.EndFrame()
	})
	if u.selRow != 1 || u.selCol != -1 {
		t.Fatalf("row-number selection = row %d col %d, want row 1 full row", u.selRow, u.selCol)
	}

	host.QueueTap(172, tableTestHeaderY()) // B header.
	host.Draw(func() {
		rl.BeginFrame()
		a.drawNativeTable(u)
		rl.EndFrame()
	})
	if u.selRow != -1 || u.selCol != 2 {
		t.Fatalf("header selection = row %d col %d, want full column B", u.selRow, u.selCol)
	}
}

func TestArrowKeysMoveSelectedCellWhenNotEditing(t *testing.T) {
	a := &app{
		dir: t.TempDir(),
		wb: &Workbook{
			Rows: []Row{
				{Section: "crypto", Label: "cold wallet", ID: "bitcoin", Units: 0.25},
				{Section: "banks", Label: "checking", ID: "USD", Units: 100},
			},
			Rates: map[string]float64{"bitcoin": 65000, "USD": 1},
		},
	}
	u := &uiState{selRow: -1, selCol: -1, disp: a.dispRows(), editText: make([]byte, 128)}
	host := rl.NewHost(rl.AppConfig{Width: winW, Height: winH})
	defer host.Close()
	defer rl.SetRuntime(nil)

	drawTable := func() {
		host.Draw(func() {
			rl.BeginFrame()
			a.drawNativeTable(u)
			rl.EndFrame()
		})
	}

	host.QueueTap(172, tableTestRowY(0)) // B/label cell in row 0.
	drawTable()
	if u.selRow != 0 || u.selCol != 2 {
		t.Fatalf("clicked selection = row %d col %d, want row 0 col 2", u.selRow, u.selCol)
	}

	host.QueueKey(rl.KeyRight)
	drawTable()
	if u.selRow != 0 || u.selCol != 3 {
		t.Fatalf("right selection = row %d col %d, want row 0 col 3", u.selRow, u.selCol)
	}

	host.QueueKey(rl.KeyDown)
	drawTable()
	if u.selRow != 1 || u.selCol != 3 {
		t.Fatalf("down selection = row %d col %d, want row 1 col 3", u.selRow, u.selCol)
	}

	host.QueueKey(rl.KeyLeft)
	drawTable()
	if u.selRow != 1 || u.selCol != 2 {
		t.Fatalf("left selection = row %d col %d, want row 1 col 2", u.selRow, u.selCol)
	}

	host.QueueKey(rl.KeyUp)
	drawTable()
	if u.selRow != 0 || u.selCol != 2 {
		t.Fatalf("up selection = row %d col %d, want row 0 col 2", u.selRow, u.selCol)
	}
}

func TestArrowKeysMoveSelectionWhenTableFocusIsStale(t *testing.T) {
	a := &app{
		dir: t.TempDir(),
		wb: &Workbook{
			Rows: []Row{
				{Section: "crypto", Label: "cold wallet", ID: "bitcoin", Units: 0.25},
				{Section: "banks", Label: "checking", ID: "USD", Units: 100},
			},
			Rates: map[string]float64{"bitcoin": 65000, "USD": 1},
		},
	}
	u := &uiState{selRow: 0, selCol: 2, disp: a.dispRows(), editText: make([]byte, 128)}
	host := rl.NewHost(rl.AppConfig{Width: winW, Height: winH})
	defer host.Close()
	defer rl.SetRuntime(nil)
	host.SetFocus(3001)

	host.QueueKey(rl.KeyRight)
	host.Draw(func() {
		rl.BeginFrame()
		a.drawNativeTable(u)
		rl.EndFrame()
	})
	if u.selRow != 0 || u.selCol != 3 {
		t.Fatalf("stale-focus right selection = row %d col %d, want row 0 col 3", u.selRow, u.selCol)
	}
}

func TestDirectTypingIntoCellDoesNotDuplicateFirstCharacter(t *testing.T) {
	a := &app{
		dir: t.TempDir(),
		wb: &Workbook{
			Rows: []Row{
				{Section: "crypto", Label: "cold wallet", ID: "bitcoin", Units: 0.25},
			},
			Rates: map[string]float64{"bitcoin": 65000},
		},
	}
	u := &uiState{selRow: 0, selCol: 2, disp: a.dispRows(), editText: make([]byte, 128)}
	host := rl.NewHost(rl.AppConfig{Width: winW, Height: winH})
	defer host.Close()
	defer rl.SetRuntime(nil)
	host.SetFocus(2001)

	host.QueueText("x")
	host.Draw(func() { draw(a, u) })
	if !u.editing {
		t.Fatal("direct typing did not enter edit mode")
	}
	if got := editString(u); got != "x" {
		t.Fatalf("edit text after direct typing = %q, want x", got)
	}
}

func TestFormulaBarEditsSelectedCell(t *testing.T) {
	a := &app{
		dir: t.TempDir(),
		wb: &Workbook{
			Rows: []Row{
				{Section: "crypto", Label: "cold wallet", ID: "bitcoin", Units: 0.25},
			},
			Rates: map[string]float64{"bitcoin": 65000},
		},
	}
	u := &uiState{selRow: 0, selCol: 2, disp: a.dispRows(), editText: make([]byte, 128)}
	host := rl.NewHost(rl.AppConfig{Width: winW, Height: winH})
	defer host.Close()
	defer rl.SetRuntime(nil)

	r := formulaBarInputRect()
	host.QueueTap(r.X+10, r.Y+r.Height/2)
	host.Draw(func() {
		rl.BeginFrame()
		a.drawFormulaBar(u)
		rl.EndFrame()
	})
	if !u.editing || !u.editBar {
		t.Fatalf("formula bar edit state editing=%v bar=%v, want active bar edit", u.editing, u.editBar)
	}
	if got := editString(u); got != "cold wallet" {
		t.Fatalf("formula bar initial text = %q, want cold wallet", got)
	}

	clear(u.editText)
	copy(u.editText, "vault")
	u.editCur = int32(len("vault"))
	a.commitCell(u)
	if got := a.wb.Rows[0].Label; got != "vault" {
		t.Fatalf("formula bar committed label = %q, want vault", got)
	}
}

func TestEscapeLeavesFormulaBarEditMode(t *testing.T) {
	a := &app{
		dir: t.TempDir(),
		wb: &Workbook{
			Rows: []Row{
				{Section: "crypto", Label: "cold wallet", ID: "bitcoin", Units: 0.25},
			},
			Rates: map[string]float64{"bitcoin": 65000},
		},
	}
	u := &uiState{selRow: 0, selCol: 2, disp: a.dispRows(), editText: make([]byte, 128)}
	a.startEdit(u, u.disp[0], dataColFromTableCol(int32(u.selCol)), true)
	u.editBar = true
	clear(u.editText)
	copy(u.editText, "discard me")
	u.editCur = int32(len("discard me"))

	host := rl.NewHost(rl.AppConfig{Width: winW, Height: winH})
	defer host.Close()
	defer rl.SetRuntime(nil)
	host.QueueKey(rl.KeyEscape)
	host.Draw(func() {
		rl.BeginFrame()
		a.drawFormulaBar(u)
		rl.EndFrame()
	})

	if u.editing || u.editBar {
		t.Fatalf("escape left formula bar edit active: editing=%v bar=%v", u.editing, u.editBar)
	}
	if got := a.wb.Rows[0].Label; got != "cold wallet" {
		t.Fatalf("escape committed label = %q, want cold wallet", got)
	}
}

func TestOnlyEnterCommitsFormulaBarEdit(t *testing.T) {
	a := &app{
		dir: t.TempDir(),
		wb: &Workbook{
			Rows: []Row{
				{Section: "crypto", Label: "cold wallet", ID: "bitcoin", Units: 0.25},
			},
			Rates: map[string]float64{"bitcoin": 65000},
		},
	}
	u := &uiState{selRow: 0, selCol: 2, disp: a.dispRows(), editText: make([]byte, 128)}
	host := rl.NewHost(rl.AppConfig{Width: winW, Height: winH})
	defer host.Close()
	defer rl.SetRuntime(nil)

	a.startEdit(u, u.disp[0], dataColFromTableCol(int32(u.selCol)), true)
	u.editBar = true
	clear(u.editText)
	copy(u.editText, "discarded")
	u.editCur = int32(len("discarded"))
	host.QueueKey(rl.KeyTab)
	host.Draw(func() {
		rl.BeginFrame()
		a.drawFormulaBar(u)
		rl.EndFrame()
	})
	if got := a.wb.Rows[0].Label; got != "cold wallet" {
		t.Fatalf("formula bar tab committed label = %q, want cold wallet", got)
	}

	u.selRow, u.selCol = 0, 2
	a.startEdit(u, u.disp[0], dataColFromTableCol(int32(u.selCol)), true)
	u.editBar = true
	clear(u.editText)
	copy(u.editText, "vault")
	u.editCur = int32(len("vault"))
	host.QueueKey(rl.KeyEnter)
	host.Draw(func() {
		rl.BeginFrame()
		a.drawFormulaBar(u)
		rl.EndFrame()
	})
	if got := a.wb.Rows[0].Label; got != "vault" {
		t.Fatalf("formula bar enter did not commit label = %q, want vault", got)
	}
}

func TestTypingIntoBlankRowCreatesRow(t *testing.T) {
	a := &app{
		dir: t.TempDir(),
		wb: &Workbook{
			Rows: []Row{
				{Section: "crypto", Label: "cold wallet", ID: "bitcoin", Units: 0.25},
			},
			Rates: map[string]float64{"bitcoin": 65000},
		},
	}
	u := &uiState{selRow: 1, selCol: 2, disp: a.dispRows(), editText: make([]byte, 128)}
	host := rl.NewHost(rl.AppConfig{Width: winW, Height: winH})
	defer host.Close()
	defer rl.SetRuntime(nil)
	host.SetFocus(2001)

	host.QueueText("x")
	host.Draw(func() { draw(a, u) })
	if len(a.wb.Rows) != 2 {
		t.Fatalf("rows after typing into blank row = %d, want 2", len(a.wb.Rows))
	}
	if !u.editing || u.editIdx != 1 || editString(u) != "x" {
		t.Fatalf("blank row edit state editing=%v idx=%d text=%q, want editing row 1 text x", u.editing, u.editIdx, editString(u))
	}
}

func TestEscapeFromNewBlankRowEditRemovesRow(t *testing.T) {
	a := &app{
		dir: t.TempDir(),
		wb: &Workbook{
			Rows: []Row{
				{Section: "crypto", Label: "cold wallet", ID: "bitcoin", Units: 0.25},
			},
			Rates: map[string]float64{"bitcoin": 65000},
		},
	}
	u := &uiState{selRow: 1, selCol: 2, disp: a.dispRows(), editText: make([]byte, 128)}
	host := rl.NewHost(rl.AppConfig{Width: winW, Height: winH})
	defer host.Close()
	defer rl.SetRuntime(nil)

	a.startEdit(u, u.disp[1], dataColFromTableCol(int32(u.selCol)), false)
	if len(a.wb.Rows) != 2 {
		t.Fatalf("rows after starting blank row edit = %d, want 2", len(a.wb.Rows))
	}
	host.QueueKey(rl.KeyEscape)
	host.Draw(func() {
		rl.BeginFrame()
		a.drawEditor(u)
		rl.EndFrame()
	})
	if len(a.wb.Rows) != 1 {
		t.Fatalf("rows after escape from blank row edit = %d, want 1", len(a.wb.Rows))
	}
}

func TestEscapeClearsSelectedCellWhenTableFocused(t *testing.T) {
	a := &app{
		dir: t.TempDir(),
		wb: &Workbook{
			Rows: []Row{
				{Section: "crypto", Label: "cold wallet", ID: "bitcoin", Units: 0.25},
			},
			Rates: map[string]float64{"bitcoin": 65000},
		},
	}
	u := &uiState{selRow: -1, selCol: -1, disp: a.dispRows(), editText: make([]byte, 128)}
	host := rl.NewHost(rl.AppConfig{Width: winW, Height: winH})
	defer host.Close()
	defer rl.SetRuntime(nil)

	drawTable := func() {
		host.Draw(func() {
			rl.BeginFrame()
			a.drawNativeTable(u)
			rl.EndFrame()
		})
	}

	host.QueueTap(172, tableTestRowY(0))
	drawTable()
	if u.selRow != 0 || u.selCol != 2 {
		t.Fatalf("clicked selection = row %d col %d, want row 0 col 2", u.selRow, u.selCol)
	}

	host.QueueKey(rl.KeyEscape)
	drawTable()
	if u.quit {
		t.Fatal("escape quit the app instead of clearing the table selection")
	}
	if u.selRow != -1 || u.selCol != -1 {
		t.Fatalf("escape selection = row %d col %d, want no selection", u.selRow, u.selCol)
	}
}

func TestRightClickRowNumberAndCellContexts(t *testing.T) {
	a := &app{
		dir: t.TempDir(),
		wb: &Workbook{
			Rows: []Row{
				{Section: "crypto", Label: "cold wallet", ID: "bitcoin", Units: 0.25},
			},
			Rates: map[string]float64{"bitcoin": 65000},
		},
	}
	u := &uiState{selRow: -1, selCol: -1, disp: a.dispRows(), editText: make([]byte, 128)}

	host := rl.NewHost(rl.AppConfig{Width: winW, Height: winH})
	defer host.Close()
	defer rl.SetRuntime(nil)

	host.Runtime().(interface {
		QueueMouseButton(int32, float32, float32)
	}).QueueMouseButton(rl.MouseButtonRight, 40, tableTestRowY(0)) // row-number column.
	host.Draw(func() {
		rl.BeginFrame()
		a.drawNativeTable(u)
		rl.EndFrame()
	})
	if !u.ctx || u.ctxCol != -1 {
		t.Fatalf("row-number context = open %v col %d, want delete-row context", u.ctx, u.ctxCol)
	}

	u.ctx = false
	host.Runtime().(interface {
		QueueMouseButton(int32, float32, float32)
	}).QueueMouseButton(rl.MouseButtonRight, 172, tableTestRowY(0)) // B/label cell.
	host.Draw(func() {
		rl.BeginFrame()
		a.drawNativeTable(u)
		rl.EndFrame()
	})
	if !u.ctx || u.ctxCol != 1 {
		t.Fatalf("cell context = open %v col %d, want clear label cell context", u.ctx, u.ctxCol)
	}
}

func TestRowContextInsertsBelow(t *testing.T) {
	a := &app{
		dir: t.TempDir(),
		wb: &Workbook{
			Rows: []Row{
				{Section: "crypto", Label: "cold wallet", ID: "bitcoin", Units: 0.25},
				{Section: "banks", Label: "checking", ID: "USD", Units: 100},
			},
			Rates: map[string]float64{"bitcoin": 65000, "USD": 1},
		},
	}
	u := &uiState{selRow: -1, selCol: -1, disp: a.dispRows(), editText: make([]byte, 128)}
	host := rl.NewHost(rl.AppConfig{Width: winW, Height: winH})
	defer host.Close()
	defer rl.SetRuntime(nil)

	host.Runtime().(interface {
		QueueMouseButton(int32, float32, float32)
	}).QueueMouseButton(rl.MouseButtonRight, 40, tableTestRowY(0))
	host.Draw(func() { draw(a, u) })
	if !u.ctx || u.ctxCol != -1 {
		t.Fatalf("row context = open %v col %d, want row context", u.ctx, u.ctxCol)
	}

	host.QueueTap(u.ctxX+10, u.ctxY+36) // insert below.
	host.Draw(func() { draw(a, u) })
	if len(a.wb.Rows) != 3 {
		t.Fatalf("rows after insert below = %d, want 3", len(a.wb.Rows))
	}
	if got := a.wb.Rows[1]; got != (Row{USDExpr: "D2*E2"}) {
		t.Fatalf("inserted row = %#v, want row with default G formula", got)
	}
	if u.selRow != 1 || u.selCol != -1 {
		t.Fatalf("selection after insert = row %d col %d, want inserted full row", u.selRow, u.selCol)
	}
}

func TestTableClipboardCopyPasteUsesEditableValues(t *testing.T) {
	a := &app{
		dir: t.TempDir(),
		wb: &Workbook{
			Rows: []Row{
				{Section: "crypto", Label: "cold wallet", ID: "bitcoin", Units: 0.25},
				{Section: "crypto", Label: "exchange", ID: "ethereum", Units: 2.5},
			},
			Rates: map[string]float64{"bitcoin": 65000, "ethereum": 3200, "USD": 1, "EUR": 1.1},
		},
	}
	u := &uiState{selRow: 0, selCol: 3, disp: a.dispRows(), editText: make([]byte, 128)}
	host := rl.NewHost(rl.AppConfig{Width: winW, Height: winH})
	defer host.Close()
	defer rl.SetRuntime(nil)
	host.SetFocus(2001)

	host.QueueShortcut(rl.KeyC)
	host.Draw(func() {
		rl.BeginFrame()
		a.drawNativeTable(u)
		rl.EndFrame()
	})
	if got := host.ClipboardText(); got != "bitcoin" {
		t.Fatalf("coin copy = %q, want editable id bitcoin", got)
	}

	u.selRow, u.selCol = 1, -1
	host.SetClipboardText("banks\tchecking\tUSD\t100\ncash\twallet\tEUR\t=40+10")
	host.QueueShortcut(rl.KeyV)
	host.Draw(func() {
		rl.BeginFrame()
		a.drawNativeTable(u)
		rl.EndFrame()
	})

	if len(a.wb.Rows) != 3 {
		t.Fatalf("rows after paste = %d, want 3", len(a.wb.Rows))
	}
	if got := a.wb.Rows[1]; got.Section != "banks" || got.Label != "checking" || got.ID != "USD" || got.Units != 100 || got.Expr != "" {
		t.Fatalf("first pasted row = %#v", got)
	}
	if got := a.wb.Rows[2]; got.Section != "cash" || got.Label != "wallet" || got.ID != "EUR" || got.Units != 50 || got.Expr != "40+10" {
		t.Fatalf("second pasted row = %#v", got)
	}
	if u.selRow != 1 || u.selCol != 1 {
		t.Fatalf("selection after paste = row %d col %d, want top-left pasted cell", u.selRow, u.selCol)
	}
}

func TestToolbarAppliesCellFormatting(t *testing.T) {
	a := &app{
		profile: profileWorkbook,
		dir:     t.TempDir(),
		wb: &Workbook{
			Rows: []Row{
				{Section: "notes", Label: "todo", ID: "USD", Units: 1},
			},
			Rates: map[string]float64{"USD": 1},
		},
	}
	u := &uiState{selRow: 0, selCol: 2, disp: a.dispRows(), editText: make([]byte, 128), openMenu: -1}
	host := rl.NewHost(rl.AppConfig{Width: winW, Height: winH})
	defer host.Close()
	defer rl.SetRuntime(nil)

	host.QueueTap(1033, 52) // text color toolbar action.
	host.Draw(func() {
		rl.BeginFrame()
		a.drawTopChrome(u)
		rl.EndFrame()
	})

	format := a.cellFormat(0, 1)
	if format.TextColor == "" {
		t.Fatalf("toolbar did not save text color format: %#v", a.wb.CellFormats)
	}
	rows := a.workbookTableRows(u)
	if rows[0].TextColors[2].A == 0 {
		t.Fatalf("formatted table row did not expose text color: %#v", rows[0].TextColors)
	}
}

func TestTableDragSelectionCopiesAndFormatsRange(t *testing.T) {
	a := &app{
		profile: profileWorkbook,
		dir:     t.TempDir(),
		wb: &Workbook{
			Rows: []Row{
				{Section: "assets", Label: "checking", ID: "USD", Units: 10},
				{Section: "assets", Label: "wallet", ID: "EUR", Units: 20},
			},
			Rates: map[string]float64{"USD": 1, "EUR": 1.1},
		},
	}
	u := &uiState{selRow: -1, selCol: -1, disp: a.dispRows(), editText: make([]byte, 128), openMenu: -1, rangeStartRow: -1, rangeStartCol: -1, rangeEndRow: -1, rangeEndCol: -1}
	host := rl.NewHost(rl.AppConfig{Width: winW, Height: winH})
	defer host.Close()
	defer rl.SetRuntime(nil)

	host.QueueMouseButtonDown(rl.MouseButtonLeft, 172, tableTestRowY(0)) // B/label row 0.
	host.Draw(func() {
		rl.BeginFrame()
		a.drawNativeTable(u)
		rl.EndFrame()
	})
	host.QueueMouseMove(520, tableTestRowY(1)) // D/units row 1.
	host.Draw(func() {
		rl.BeginFrame()
		a.drawNativeTable(u)
		rl.EndFrame()
	})
	host.QueueMouseButtonUp(rl.MouseButtonLeft, 366, tableTestRowY(1))
	host.Draw(func() {
		rl.BeginFrame()
		a.drawNativeTable(u)
		rl.EndFrame()
	})

	if !u.hasRange() {
		t.Fatalf("drag did not create range: %#v", u)
	}
	if got, want := u.rangeStartRow, 0; got != want {
		t.Fatalf("range start row = %d, want %d", got, want)
	}
	if got, want := u.rangeEndRow, 1; got != want {
		t.Fatalf("range end row = %d, want %d", got, want)
	}
	text, ok := a.tableCopyText(u)
	if !ok || text != "checking\tUSD\t10\nwallet\tEUR\t20" {
		t.Fatalf("range copy = %q ok=%v", text, ok)
	}
	a.applyNextBackgroundColor(u)
	for _, key := range []string{cellFormatKey(0, 1), cellFormatKey(0, 2), cellFormatKey(0, 3), cellFormatKey(1, 1), cellFormatKey(1, 2), cellFormatKey(1, 3)} {
		if a.wb.CellFormats[key].BackgroundColor == "" {
			t.Fatalf("missing background format for %s: %#v", key, a.wb.CellFormats)
		}
	}
}

func TestComputedColumnsAreEditableAndFormulaBased(t *testing.T) {
	a := &app{
		dir: t.TempDir(),
		wb: &Workbook{
			Rows: []Row{
				{Section: "crypto", Label: "cold wallet", ID: "bitcoin", Units: 2},
			},
			Rates: map[string]float64{"bitcoin": 10},
		},
	}
	u := &uiState{selRow: 0, selCol: 6, disp: a.dispRows(), editText: make([]byte, 128)}

	if got := a.rowCellEditText(0, 6); got != "=D1*E1" {
		t.Fatalf("default usd edit text = %q, want =D1*E1", got)
	}
	if !a.setRowCell(0, 6, "=D1*E1+5") {
		t.Fatalf("setting usd formula failed: %s", a.err)
	}
	if got := a.rowCellEditText(0, 6); got != "=D1*E1+5" {
		t.Fatalf("usd formula edit text = %q", got)
	}
	rows := a.workbookTableRows(u)
	if got := rows[0].Cells[6]; got != "25.00" {
		t.Fatalf("selected usd formula display = %q, want value 25.00", got)
	}
	if !a.setRowCell(0, 4, "12") {
		t.Fatalf("setting rate override failed: %s", a.err)
	}
	if got := a.rowCellEditText(0, 4); got != "12" {
		t.Fatalf("rate override edit text = %q", got)
	}
}

func TestFillHandleCopiesFormulaWithShiftedRows(t *testing.T) {
	a := &app{
		dir: t.TempDir(),
		wb: &Workbook{
			Rows: []Row{
				{Section: "crypto", Label: "cold wallet", ID: "bitcoin", Units: 2},
				{Section: "crypto", Label: "exchange", ID: "bitcoin", Units: 3},
			},
			Rates: map[string]float64{"bitcoin": 10},
		},
	}
	u := &uiState{selRow: 0, selCol: 6, disp: a.dispRows(), editText: make([]byte, 128)}
	host := rl.NewHost(rl.AppConfig{Width: winW, Height: winH})
	defer host.Close()
	defer rl.SetRuntime(nil)
	host.SetFocus(2001)

	scroll := int32(0)
	props := a.workbookTableProps(u, ptr32(0), ptr32(6), ptr32(-1), ptr32(-1), ptr32(-1), ptr32(-1), ptr32(-1), ptr32(-1), ptr32(-1), ptr32(-1), ptr32(0), &scroll)
	handle := fillHandleRect(rl.TableCellRect(props, 0, 6))
	target := rl.TableCellRect(props, 1, 6)
	startX, startY := handle.X+handle.Width/2, handle.Y+handle.Height/2
	endX, endY := target.X+target.Width/2, target.Y+target.Height/2

	drawTable := func() {
		host.Draw(func() {
			rl.BeginFrame()
			a.drawNativeTable(u)
			rl.EndFrame()
		})
	}

	host.QueueMouseButtonDown(rl.MouseButtonLeft, startX, startY)
	drawTable()
	host.QueueMouseMove(endX, endY)
	drawTable()
	host.QueueMouseButtonUp(rl.MouseButtonLeft, endX, endY)
	drawTable()

	if got := a.wb.Rows[1].USDExpr; got != "D2*E2" {
		t.Fatalf("filled USD formula = %q, want D2*E2", got)
	}
	if got, ok := a.cellNumber(1, 6, map[string]bool{}); !ok || got != 30 {
		t.Fatalf("filled USD value = %v ok=%v, want 30 true", got, ok)
	}
}

func TestTotalFormulaCanBeEdited(t *testing.T) {
	a := &app{
		dir: t.TempDir(),
		wb: &Workbook{
			Rows: []Row{
				{Section: "banks", Label: "checking", ID: "USD", Units: 100},
			},
			Rates: map[string]float64{"USD": 1},
		},
	}
	u := &uiState{selRow: len(a.wb.Rows) + blankRowsBeforeTotals + 1, selCol: 6, disp: a.dispRows(), editText: make([]byte, 128)}

	a.startEdit(u, u.disp[u.selRow], dataColFromTableCol(int32(u.selCol)), true)
	if !u.editing {
		t.Fatal("total formula did not enter edit mode")
	}
	if got := editString(u); got != `=SUM(G1:G1)` {
		t.Fatalf("initial total formula edit text = %q", got)
	}
	clear(u.editText)
	copy(u.editText, "=G1+25")
	u.editCur = int32(len("=G1+25"))
	a.commitCell(u)

	if got := a.wb.TotalCells[6]; got != "=G1+25" {
		t.Fatalf("stored total formula = %q", got)
	}
	rows := a.workbookTableRows(u)
	if got := rows[len(rows)-1].Cells[6]; got != "125.00" {
		t.Fatalf("edited total value = %q, want 125.00", got)
	}
}

func TestFormulaReferenceGroupsHighlightPendingCorrectly(t *testing.T) {
	a := &app{
		wb: &Workbook{
			Rows: []Row{
				{Section: "banks", ID: "USD", Units: 100},
				{Section: "pending", ID: "USD", Units: 50},
				{Section: "crypto", ID: "bitcoin", Units: 1},
			},
			Rates: map[string]float64{"USD": 1, "bitcoin": 10},
		},
	}

	all := a.formulaRefGroups("=SUM(G1:G3)")
	if len(all) != 1 || len(all[0].Cells) != 3 {
		t.Fatalf("SUM refs = %#v, want one group with all 3 rows", all)
	}
	if got := a.defaultTotalFormula(dispTotal, 6); got != `=SUM(G1:G3)-SUM(G2:G2)` {
		t.Fatalf("default total formula = %q", got)
	}
	total := a.formulaRefGroups(a.defaultTotalFormula(dispTotal, 6))
	if len(total) != 2 {
		t.Fatalf("total formula groups = %#v, want all and pending groups", total)
	}
	if len(total[0].Cells) != 3 {
		t.Fatalf("total all refs = %#v, want all 3 rows", total[0].Cells)
	}
	if got, want := total[1].Cells, []formulaCellRef{{Row: 1, Col: 6}}; len(got) != 1 || got[0] != want[0] {
		t.Fatalf("total pending refs = %#v, want %#v", got, want)
	}
}

func TestDefaultTotalFormulaSubtractsPendingSectionBlock(t *testing.T) {
	a := &app{
		wb: &Workbook{
			Rows: []Row{
				{Section: "banks", ID: "USD", Units: 100},
				{Section: "", ID: "USD", Units: 20},
				{Section: "pending", ID: "", Units: 0},
				{Section: "", ID: "USD", Units: 50},
				{Section: "", ID: "USD", Units: 30},
				{Section: "debts", ID: "", Units: 0},
				{Section: "", ID: "USD", Units: -10},
			},
		},
	}

	if got := a.defaultTotalFormula(dispTotal, 6); got != `=SUM(G1:G7)-SUM(G3:G5)` {
		t.Fatalf("default total formula = %q", got)
	}
	total := a.formulaRefGroups(a.defaultTotalFormula(dispTotal, 6))
	if len(total) != 2 {
		t.Fatalf("total formula groups = %#v, want all and pending groups", total)
	}
	if got, want := total[1].Cells, []formulaCellRef{
		{Row: 2, Col: 6},
		{Row: 3, Col: 6},
		{Row: 4, Col: 6},
	}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
		t.Fatalf("total pending refs = %#v, want %#v", got, want)
	}
}

func TestEveryTotalFieldCanEnterEditMode(t *testing.T) {
	a := &app{
		dir: t.TempDir(),
		wb: &Workbook{
			Rows:  []Row{{Section: "banks", Label: "checking", ID: "USD", Units: 100}},
			Rates: map[string]float64{"USD": 1},
		},
	}
	row := len(a.wb.Rows) + blankRowsBeforeTotals
	for tableCol := 1; tableCol <= len(visibleDataCols); tableCol++ {
		u := &uiState{selRow: row, selCol: tableCol, disp: a.dispRows(), editText: make([]byte, 128)}
		a.startEdit(u, u.disp[u.selRow], dataColFromTableCol(int32(u.selCol)), true)
		if !u.editing {
			t.Fatalf("total row table col %d did not enter edit mode", tableCol)
		}
	}
}

func TestTableShowsComputedTotalRows(t *testing.T) {
	a := &app{
		wb: &Workbook{
			Rows: []Row{
				{Section: "crypto", Label: "cold wallet", ID: "bitcoin", Units: 0.25},
				{Section: "pending", Label: "incoming", ID: "USD", Units: 50},
				{Section: "banks", Label: "checking", ID: "USD", Units: 100},
			},
			Rates: map[string]float64{"bitcoin": 100, "USD": 1},
		},
		prev: map[string]float64{"bitcoin": 80, "USD": 1},
	}
	u := &uiState{disp: a.dispRows()}
	rows := a.workbookTableRows(u)
	totalPending := rows[len(rows)-2].Cells
	total := rows[len(rows)-1].Cells

	if totalPending[2] != "total + pending" || totalPending[4] != "" || totalPending[6] != "175.00" {
		t.Fatalf("total + pending row = %#v", totalPending)
	}
	if total[2] != "total" || total[4] != "" || total[6] != "125.00" {
		t.Fatalf("total row = %#v", total)
	}

	u.selRow = len(rows) - 2
	u.selCol = 6
	rows = a.workbookTableRows(u)
	if got := rows[len(rows)-2].Cells[6]; got != "175.00" {
		t.Fatalf("selected total + pending value = %q", got)
	}
}
