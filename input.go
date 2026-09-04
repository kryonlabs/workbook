package main

import (
	"strconv"
	"strings"
	"unicode"

	rl "github.com/waozixyz/kryon/go/kryon"
)

// Spreadsheet-style interaction: click a cell, type, enter. No letter hotkeys
// so typing always goes into the cell.

// cellRect is the editable box of column col (0 section · 1 label · 2 coin ·
// 3 units) on the row whose text is drawn at y.
func cellRect(col int, y float32) rl.Rectangle {
	var x, w float32
	switch col {
	case 0:
		x, w = xSec, xLabel-xSec-12
	case 1:
		x, w = xLabel, xCoin-xLabel-12
	case 2:
		x, w = xCoin, rUnits-160-xCoin
	case 3:
		x, w = rUnits-140, 150
	}
	return rl.NewRectangle(x, y-4, w, rowH)
}

func colAtX(x float32) int {
	switch {
	case x >= xSec && x < xLabel-12:
		return 0
	case x >= xLabel && x < xCoin-12:
		return 1
	case x >= xCoin && x < rUnits-160:
		return 2
	case x >= rUnits-140 && x < rUnits+10:
		return 3
	}
	return -1
}

func dataColFromTableCol(col int32) int {
	i := int(col) - 1
	if i >= 0 && i < len(visibleDataCols) {
		return visibleDataCols[i]
	}
	return -1
}

func tableColFromDataCol(col int) int {
	for i, dataCol := range visibleDataCols {
		if dataCol == col {
			return i + 1
		}
	}
	return -1
}

func (u *uiState) normalizeRange() {
	if u.rangeStartRow == 0 && u.rangeStartCol == 0 && u.rangeEndRow == 0 && u.rangeEndCol == 0 {
		u.clearRange()
	}
}

func (u *uiState) clearRange() {
	u.rangeStartRow, u.rangeStartCol = -1, -1
	u.rangeEndRow, u.rangeEndCol = -1, -1
}

func (u *uiState) hasRange() bool {
	u.normalizeRange()
	return u.rangeStartRow >= 0 && u.rangeStartCol >= 0 && u.rangeEndRow >= 0 && u.rangeEndCol >= 0 &&
		(u.rangeStartRow != u.rangeEndRow || u.rangeStartCol != u.rangeEndCol)
}

func (u *uiState) rangeBounds() (int, int, int, int, bool) {
	if !u.hasRange() {
		return 0, 0, 0, 0, false
	}
	startRow, endRow := u.rangeStartRow, u.rangeEndRow
	startCol, endCol := u.rangeStartCol, u.rangeEndCol
	if startRow > endRow {
		startRow, endRow = endRow, startRow
	}
	if startCol > endCol {
		startCol, endCol = endCol, startCol
	}
	return startRow, startCol, endRow, endCol, true
}

// rowAt maps a y coordinate to a display row index (-1 = outside the table).
func (u *uiState) rowAt(y float32) int {
	if y < tableTop || y >= tableBot {
		return -1
	}
	i := u.offset + int((y-tableTop)/rowH)
	if i >= len(u.disp) {
		return -1
	}
	return i
}

func ctxRect(u *uiState) rl.Rectangle {
	x, y := u.ctxX, u.ctxY
	h := float32(112)
	if u.ctxCol < 0 || u.ctxHeader {
		h = 84
	}
	if x > winW-190 {
		x = winW - 190
	}
	if y > winH-h-8 {
		y = winH - h - 8
	}
	return rl.NewRectangle(x, y, 180, h)
}

func normID(s string) string {
	if isFiat(s) {
		return strings.ToUpper(s)
	}
	return strings.ToLower(s)
}

// typedChars collects the frame's printable keypresses.
func typedChars() string {
	var b strings.Builder
	for {
		c := rl.GetCharPressed()
		if c <= 0 {
			break
		}
		if unicode.IsPrint(rune(c)) {
			b.WriteRune(rune(c))
		}
	}
	return b.String()
}

// startEdit opens the in-place cell editor. existing=true preloads the cell's
// current content (second click / enter / f2); typing starts with empty=false.
func (a *app) startEdit(u *uiState, dr displayRow, col int, existing bool) {
	if col < 0 || col > 7 {
		return
	}
	if dr.kind == dispTotalPending || dr.kind == dispTotal {
		u.editing = true
		clear(u.editText)
		u.editCur = 0
		u.editFocus = true
		u.editIdx = -1
		u.editNew = false
		u.editTotal = dr.kind
		u.editBar = false
		value := a.totalCellEditText(dr.kind, col)
		copy(u.editText, value)
		u.editCur = int32(len(value))
		return
	}
	if dr.kind != dispRow && dr.kind != dispBlank {
		return
	}
	u.editing = true
	clear(u.editText)
	u.editCur = 0
	u.editFocus = true
	u.editIdx = -1
	u.editNew = false
	u.editTotal = 0
	u.editBar = false
	value := ""
	if dr.kind == dispBlank {
		idx := clampInt(dr.idx, 0, len(a.wb.Rows))
		a.wb.Rows = append(a.wb.Rows, Row{})
		copy(a.wb.Rows[idx+1:], a.wb.Rows[idx:])
		a.wb.Rows[idx] = Row{}
		u.disp = a.dispRows()
		dr = displayRow{kind: dispRow, idx: idx}
		u.selRow = idx
		u.editNew = true
	}
	if dr.kind == dispRow {
		u.editIdx = dr.idx
	}
	if existing && dr.kind == dispRow {
		value = a.rowCellEditText(dr.idx, col)
	}
	copy(u.editText, value)
	u.editCur = int32(len(value))
}

// commitCell applies the edited cell.
func (a *app) commitCell(u *uiState) {
	if !u.editing {
		return
	}
	u.editing = false
	u.editBar = false
	col := dataColFromTableCol(int32(u.selCol))
	if col < 0 {
		return
	}
	if u.editTotal != 0 {
		a.setTotalCell(u.editTotal, col, editString(u))
		a.save("cell saved")
		u.editTotal = 0
		return
	}

	if u.editIdx < 0 || u.editIdx >= len(a.wb.Rows) {
		return
	}
	if !a.setRowCell(u.editIdx, col, editString(u)) {
		return
	}
	u.editNew = false
	as := a.wb.Rows[u.editIdx]
	a.save(tableColName(col) + " saved")
	a.selectRow(u, as)
}

// clearCell wipes the selected cell with Delete.
func (a *app) clearCell(u *uiState) {
	if a.clearSelectedRangeCells(u) {
		return
	}
	col := dataColFromTableCol(int32(u.selCol))
	if u.selRow < 0 || u.selRow >= len(u.disp) || col < 0 {
		return
	}
	switch u.disp[u.selRow].kind {
	case dispRow:
		a.clearRowCell(u, u.disp[u.selRow].idx, col)
	case dispTotalPending, dispTotal:
		a.setTotalCell(u.disp[u.selRow].kind, col, "")
		a.save("cell cleared")
	}
}

func (a *app) selectedRangeDataCells(u *uiState) []selectedCell {
	startRow, startCol, endRow, endCol, ok := u.rangeBounds()
	if !ok {
		if cell := selectedDataCell(u); cell != nil {
			return []selectedCell{*cell}
		}
		return nil
	}
	cells := []selectedCell{}
	for row := startRow; row <= endRow && row < len(u.disp); row++ {
		if row < 0 || u.disp[row].kind != dispRow {
			continue
		}
		for tableCol := startCol; tableCol <= endCol; tableCol++ {
			dataCol := dataColFromTableCol(int32(tableCol))
			if dataCol >= 0 {
				cells = append(cells, selectedCell{row: u.disp[row].idx, col: dataCol})
			}
		}
	}
	return cells
}

func (a *app) clearSelectedRangeCells(u *uiState) bool {
	if !u.hasRange() {
		return false
	}
	cells := a.selectedRangeDataCells(u)
	if len(cells) == 0 {
		return false
	}
	for _, cell := range cells {
		a.clearRowCell(u, cell.row, cell.col)
	}
	a.save("cells cleared")
	return true
}

func (a *app) clearRowCell(u *uiState, idx, col int) {
	if idx < 0 || idx >= len(a.wb.Rows) || col < 0 || col > 7 {
		return
	}
	as := a.wb.Rows[idx]
	delete(a.wb.CellValues, cellFormatKey(idx, col))
	switch col {
	case 0:
		as.Section = ""
	case 1:
		as.Label = ""
	case 2:
		as.ID = ""
	case 3:
		as.Units, as.Expr = 0, ""
	case 4:
		as.Rate, as.RateExpr = nil, ""
	case 5:
		as.Pct, as.PctExpr = nil, ""
	case 6:
		as.USD, as.USDExpr = nil, ""
	case 7:
		as.DUSD, as.DUSDExpr = nil, ""
	}
	a.wb.Rows[idx] = as
	a.save("cell cleared")
	a.selectRow(u, as)
}

func (a *app) insertRowRelative(u *uiState, idx int, below bool) {
	if idx < 0 || idx >= len(a.wb.Rows) {
		return
	}
	pos := idx
	if below {
		pos++
	}
	a.rewriteWorkbookFormulaRows(pos, true)
	a.shiftCellFormatsForInsert(pos)
	a.shiftCellValuesForRowInsert(pos)
	a.wb.Rows = append(a.wb.Rows, Row{})
	copy(a.wb.Rows[pos+1:], a.wb.Rows[pos:])
	a.wb.Rows[pos] = Row{}
	a.save("row inserted")
	u.disp = a.dispRows()
	u.selCol = -1
	u.selRow = -1
	for row, dr := range u.disp {
		if dr.kind == dispRow && dr.idx == pos {
			u.selRow = row
			break
		}
	}
}

func (a *app) deleteRow(u *uiState, idx int) {
	if idx < 0 || idx >= len(a.wb.Rows) {
		return
	}
	id := a.wb.Rows[idx].ID
	a.rewriteWorkbookFormulaRows(idx, false)
	a.shiftCellFormatsForDelete(idx)
	a.shiftCellValuesForRowDelete(idx)
	a.wb.Rows = append(a.wb.Rows[:idx], a.wb.Rows[idx+1:]...)
	u.selRow = -1
	u.disp = a.dispRows()
	a.save("deleted " + id)
}

func (a *app) setRowCell(idx, col int, raw string) bool {
	if idx < 0 || idx >= len(a.wb.Rows) || col < 0 || col > 7 {
		return false
	}
	formula := strings.HasPrefix(strings.TrimSpace(raw), "=")
	s := stripExpr(raw)
	as := a.wb.Rows[idx]
	key := cellFormatKey(idx, col)
	if a.wb.CellValues == nil {
		a.wb.CellValues = map[string]string{}
	}
	previousRaw, hadPreviousRaw := a.wb.CellValues[key]
	delete(a.wb.CellValues, key)
	switch col {
	case 0:
		if formula {
			a.wb.CellValues[key] = strings.TrimSpace(raw)
			as.Section = ""
		} else {
			as.Section = s
		}
	case 1:
		if formula {
			a.wb.CellValues[key] = strings.TrimSpace(raw)
			as.Label = ""
		} else {
			as.Label = s
		}
	case 2:
		if formula {
			a.wb.CellValues[key] = strings.TrimSpace(raw)
			as.ID = ""
		} else {
			as.ID = normID(s)
		}
	case 3:
		if strings.TrimSpace(s) == "" {
			as.Units, as.Expr = 0, ""
			a.wb.Rows[idx] = as
			return true
		}
		var v float64
		var err error
		if formula {
			v, err = a.evalTableFormula(s, idx, map[string]bool{})
		} else {
			v, err = evalUnits(s)
		}
		if err != nil {
			if !formula {
				a.wb.CellValues[key] = raw
				as.Units, as.Expr = 0, ""
				break
			}
			if hadPreviousRaw {
				a.wb.CellValues[key] = previousRaw
			}
			a.err = "bad units: " + raw
			return false
		}
		as.Units, as.Expr = v, ""
		if _, e := strconv.ParseFloat(s, 64); e != nil || formula {
			as.Expr = s
		}
	case 4:
		if !setNumericCell(raw, &as.Rate, &as.RateExpr) {
			a.wb.CellValues[key] = raw
		}
	case 5:
		if !setPctCell(raw, &as.Pct, &as.PctExpr) {
			a.wb.CellValues[key] = raw
		}
	case 6:
		if !setNumericCell(raw, &as.USD, &as.USDExpr) {
			a.wb.CellValues[key] = raw
		}
	case 7:
		if !setNumericCell(raw, &as.DUSD, &as.DUSDExpr) {
			a.wb.CellValues[key] = raw
		}
	}
	a.wb.Rows[idx] = as
	return true
}

func setNumericCell(raw string, out **float64, expr *string) bool {
	raw = strings.TrimSpace(raw)
	*out = nil
	*expr = ""
	if raw == "" {
		return true
	}
	if strings.HasPrefix(raw, "=") {
		*expr = stripExpr(raw)
		return true
	}
	v, err := strconv.ParseFloat(strings.ReplaceAll(raw, ",", ""), 64)
	if err != nil {
		return false
	}
	*out = &v
	return true
}

func setPctCell(raw string, out **float64, expr *string) bool {
	raw = strings.TrimSpace(raw)
	if strings.HasSuffix(raw, "%") && !strings.HasPrefix(raw, "=") {
		v, err := strconv.ParseFloat(strings.TrimSuffix(raw, "%"), 64)
		if err != nil {
			return false
		}
		v /= 100
		*out = &v
		*expr = ""
		return true
	}
	return setNumericCell(raw, out, expr)
}

func (a *app) tableCopyText(u *uiState) (string, bool) {
	if text, ok := a.rangeCopyText(u); ok {
		return text, true
	}
	switch {
	case u.selRow >= 0 && u.selRow < len(u.disp) && u.selCol < 0:
		if u.disp[u.selRow].kind == dispBlank {
			return "", true
		}
		if u.disp[u.selRow].kind != dispRow {
			return a.tableDisplayCopyText(u)
		}
		idx := u.disp[u.selRow].idx
		cells := make([]string, len(visibleDataCols))
		for i, col := range visibleDataCols {
			cells[i] = a.rowCellEditText(idx, col)
		}
		return strings.Join(cells, "\t"), true
	case u.selRow >= 0 && u.selRow < len(u.disp):
		if u.disp[u.selRow].kind == dispBlank {
			return "", true
		}
		if u.disp[u.selRow].kind != dispRow {
			return a.tableDisplayCopyText(u)
		}
		col := dataColFromTableCol(int32(u.selCol))
		if col < 0 {
			return "", false
		}
		return a.rowCellEditText(u.disp[u.selRow].idx, col), true
	case u.selRow < 0:
		col := dataColFromTableCol(int32(u.selCol))
		if col < 0 {
			return "", false
		}
		cells := make([]string, 0, len(u.disp))
		for _, dr := range u.disp {
			if dr.kind == dispRow {
				cells = append(cells, a.rowCellEditText(dr.idx, col))
			}
		}
		return strings.Join(cells, "\n"), true
	}
	return "", false
}

func (a *app) rangeCopyText(u *uiState) (string, bool) {
	startRow, startCol, endRow, endCol, ok := u.rangeBounds()
	if !ok {
		return "", false
	}
	tableRows := a.workbookTableRows(u)
	lines := []string{}
	for row := startRow; row <= endRow && row < len(u.disp); row++ {
		if row < 0 {
			continue
		}
		fields := []string{}
		for tableCol := startCol; tableCol <= endCol; tableCol++ {
			dataCol := dataColFromTableCol(int32(tableCol))
			if dataCol < 0 {
				fields = append(fields, "")
				continue
			}
			if u.disp[row].kind == dispRow {
				fields = append(fields, a.rowCellEditText(u.disp[row].idx, dataCol))
				continue
			}
			if row < len(tableRows) && tableCol < len(tableRows[row].Cells) {
				fields = append(fields, tableRows[row].Cells[tableCol])
			} else {
				fields = append(fields, "")
			}
		}
		lines = append(lines, strings.Join(fields, "\t"))
	}
	return strings.Join(lines, "\n"), true
}

func (a *app) tableDisplayCopyText(u *uiState) (string, bool) {
	rows := a.workbookTableRows(u)
	if u.selRow < 0 || u.selRow >= len(rows) {
		return "", false
	}
	cells := rows[u.selRow].Cells
	if u.selCol >= 0 {
		if u.selCol >= len(cells) {
			return "", false
		}
		return cells[u.selCol], true
	}
	if len(cells) <= 1 {
		return "", true
	}
	return strings.Join(cells[1:], "\t"), true
}

func (a *app) rowCellEditText(idx, col int) string {
	if idx < 0 || idx >= len(a.wb.Rows) {
		return ""
	}
	if raw, ok := a.wb.CellValues[cellFormatKey(idx, col)]; ok {
		return raw
	}
	as := a.wb.Rows[idx]
	switch col {
	case 0:
		return as.Section
	case 1:
		return as.Label
	case 2:
		return as.ID
	case 3:
		if as.Expr != "" {
			return "=" + as.Expr
		}
		return strconv.FormatFloat(as.Units, 'f', -1, 64)
	case 4:
		if as.RateExpr != "" {
			return "=" + as.RateExpr
		}
		if s := optionalFloatText(as.Rate); s != "" {
			return s
		}
		v, _ := a.cellNumber(idx, 4, map[string]bool{})
		return strconv.FormatFloat(v, 'f', -1, 64)
	case 5:
		if as.PctExpr != "" {
			return "=" + as.PctExpr
		}
		if as.Pct != nil {
			return strconv.FormatFloat(*as.Pct*100, 'f', -1, 64) + "%"
		}
		v, _ := a.cellNumber(idx, 5, map[string]bool{})
		return strconv.FormatFloat(v*100, 'f', -1, 64) + "%"
	case 6:
		if as.USDExpr != "" {
			return "=" + as.USDExpr
		}
		if s := optionalFloatText(as.USD); s != "" {
			return s
		}
		return "=" + defaultRowFormula(idx, col)
	case 7:
		if as.DUSDExpr != "" {
			return "=" + as.DUSDExpr
		}
		if s := optionalFloatText(as.DUSD); s != "" {
			return s
		}
		p, ok := a.prev[as.ID]
		if !ok {
			return ""
		}
		row := strconv.Itoa(idx + 1)
		return "=(E" + row + "-" + strconv.FormatFloat(p, 'f', -1, 64) + ")*D" + row
	}
	return ""
}

func (a *app) pasteTableText(u *uiState, row, tableCol int32, text string) {
	if row < 0 {
		row = 0
	}
	startVisibleCol := 0
	if tableCol >= 1 {
		startVisibleCol = int(tableCol) - 1
	}
	if startVisibleCol < 0 || startVisibleCol >= len(visibleDataCols) {
		return
	}
	rows := clipboardRows(text)
	if len(rows) == 0 {
		return
	}
	startRow := int(row)
	if startRow > len(a.wb.Rows) {
		startRow = len(a.wb.Rows)
	}
	for len(a.wb.Rows) < startRow+len(rows) {
		a.wb.Rows = append(a.wb.Rows, Row{})
	}
	orig := append([]Row(nil), a.wb.Rows...)
	for r, fields := range rows {
		for c, field := range fields {
			visibleCol := startVisibleCol + c
			if visibleCol >= len(visibleDataCols) {
				break
			}
			col := visibleDataCols[visibleCol]
			if !a.setRowCell(startRow+r, col, field) {
				a.wb.Rows = orig
				u.disp = a.dispRows()
				return
			}
		}
	}
	a.save("cells pasted")
	u.disp = a.dispRows()
	u.selRow = clampInt(startRow, 0, len(u.disp)-1)
	u.selCol = tableColFromDataCol(visibleDataCols[startVisibleCol])
}

func clipboardRows(text string) [][]string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	text = strings.TrimSuffix(text, "\n")
	lines := strings.Split(text, "\n")
	rows := make([][]string, 0, len(lines))
	for _, line := range lines {
		rows = append(rows, strings.Split(line, "\t"))
	}
	return rows
}

// selectRow points the selection back at the same workbook row after a save.
func (a *app) selectRow(u *uiState, as Row) {
	u.disp = a.dispRows()
	idx := -1
	for i := range a.wb.Rows {
		if a.wb.Rows[i] == as {
			idx = i
			break
		}
	}
	if idx < 0 {
		return
	}
	for i, d := range u.disp {
		if d.kind == dispRow && d.idx == idx {
			u.selRow = i
			return
		}
	}
}

func tableColName(col int) string {
	switch col {
	case 0:
		return "section"
	case 1:
		return "label"
	case 2:
		return "coin"
	case 3:
		return "units"
	case 4:
		return "rate"
	case 5:
		return "percent"
	case 6:
		return "usd"
	case 7:
		return "delta usd"
	}
	return "cell"
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func runeLen(s string) int {
	return len([]rune(s))
}

func editString(u *uiState) string {
	for i, b := range u.editText {
		if b == 0 {
			return string(u.editText[:i])
		}
	}
	return string(u.editText)
}

func insertAt(s string, cursor int, ins string) (string, int) {
	r := []rune(s)
	add := []rune(ins)
	cursor = clampInt(cursor, 0, len(r))
	if len(add) == 0 {
		return s, cursor
	}
	out := make([]rune, 0, len(r)+len(add))
	out = append(out, r[:cursor]...)
	out = append(out, add...)
	out = append(out, r[cursor:]...)
	if len(out) > 1024 {
		out = out[:1024]
	}
	cursor += len(add)
	if cursor > len(out) {
		cursor = len(out)
	}
	return string(out), cursor
}

func deleteBefore(s string, cursor int) (string, int) {
	r := []rune(s)
	cursor = clampInt(cursor, 0, len(r))
	if cursor == 0 {
		return s, cursor
	}
	out := append(append([]rune{}, r[:cursor-1]...), r[cursor:]...)
	return string(out), cursor - 1
}

func deleteAt(s string, cursor int) (string, int) {
	r := []rune(s)
	cursor = clampInt(cursor, 0, len(r))
	if cursor >= len(r) {
		return s, cursor
	}
	out := append(append([]rune{}, r[:cursor]...), r[cursor+1:]...)
	return string(out), cursor
}
