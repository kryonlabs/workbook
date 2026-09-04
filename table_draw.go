package main

import (
	"sort"
	"strconv"
	"strings"

	rl "github.com/waozixyz/kryon/go/kryon"
)

// --- drawing -----------------------------------------------------------------

func draw(a *app, u *uiState) {
	rl.BeginDrawing()
	syncThemeColors()
	rl.ClearBackground(rl.GetThemeBackground())

	a.drawFormulaBar(u)
	a.drawNativeTable(u)
	a.drawEditor(u)
	if u.ctx {
		if rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
			m := rl.GetMousePosition()
			u.ctx = false
			if rl.CheckCollisionPointRec(m, ctxRect(u)) {
				if u.ctxHeader {
					switch rowContextAction(u, m.Y) {
					case 0:
						a.mutateColumn(u, u.ctxCol, true)
					case 1:
						a.mutateColumn(u, clampInt(u.ctxCol+1, 0, 7), true)
					case 2:
						a.mutateColumn(u, u.ctxCol, false)
					}
				} else if u.ctxIdx >= 0 && u.ctxIdx < len(a.wb.Rows) && u.ctxCol < 0 {
					switch rowContextAction(u, m.Y) {
					case 0:
						a.insertRowRelative(u, u.ctxIdx, false)
					case 1:
						a.insertRowRelative(u, u.ctxIdx, true)
					case 2:
						a.deleteRow(u, u.ctxIdx)
					}
				} else if u.ctxIdx >= 0 && u.ctxIdx < len(a.wb.Rows) {
					switch int((m.Y - ctxRect(u).Y) / 28) {
					case 0:
						a.clearRowCell(u, u.ctxIdx, u.ctxCol)
					case 1:
						a.setCellTextColor(u.ctxIdx, u.ctxCol, nextPaletteColor(a.cellFormat(u.ctxIdx, u.ctxCol).TextColor, formatPalette))
						a.save("text color")
					case 2:
						a.setCellBackgroundColor(u.ctxIdx, u.ctxCol, nextPaletteColor(a.cellFormat(u.ctxIdx, u.ctxCol).BackgroundColor, backgroundPalette))
						a.save("background color")
					case 3:
						a.toggleCellConditional(u.ctxIdx, u.ctxCol)
						a.save("conditional formatting")
					}
				}
			}
		}
		r := ctxRect(u)
		rl.DrawRectangleRec(r, colPanel)
		rl.DrawRectangleLinesEx(r, 1, colRed)
		if u.ctxHeader {
			for i, label := range []string{"insert column left", "insert column right", "delete column"} {
				txt(label, r.X+10, r.Y+7+float32(i*28), 13, colRed)
			}
		} else if u.ctxCol < 0 {
			for i, label := range []string{"insert above", "insert below", "delete row"} {
				txt(label, r.X+10, r.Y+7+float32(i*28), 13, colRed)
			}
		} else {
			for i, label := range []string{"clear cell", "text color", "background color", "conditional: sign"} {
				txt(label, r.X+10, r.Y+7+float32(i*28), 13, colRed)
			}
		}
	}
	a.drawTopChrome(u)
	rl.EndDrawing()
}

func rowContextAction(u *uiState, y float32) int {
	action := int((y - ctxRect(u).Y) / 28)
	if action < 0 || action > 2 {
		return -1
	}
	return action
}

func syncThemeColors() {
	colBG = rl.GetThemeBackground()
	colPanel = rl.GetThemeButton()
	colText = rl.GetThemeText()
	colDim = rl.GetThemeIcon()
	colAccent = rl.GetThemeLink()
	colSel = rl.GetThemeButtonHover()
}

func (a *app) drawNativeTable(u *uiState) {
	u.normalizeRange()
	selectedRow := int32(u.selRow)
	selectedCol := int32(u.selCol)
	hadSelection := selectedRow >= 0 || selectedCol >= 0
	tableSelectedRow := selectedRow
	tableSelectedCol := selectedCol
	if u.editing {
		tableSelectedRow = -1
		tableSelectedCol = -1
	}
	scroll := int32(u.offset * rowH)
	sortCol := int32(u.sortCol)
	activatedRow, activatedCol := int32(-1), int32(-1)
	rightRow, rightCol := int32(-1), int32(-1)
	rangeStartRow, rangeStartCol := int32(u.rangeStartRow), int32(u.rangeStartCol)
	rangeEndRow, rangeEndCol := int32(u.rangeEndRow), int32(u.rangeEndCol)
	pastedText := ""
	pastedRow, pastedCol := int32(-1), int32(-1)
	props := a.workbookTableProps(u, &tableSelectedRow, &tableSelectedCol, &rangeStartRow, &rangeStartCol, &rangeEndRow, &rangeEndCol, &activatedRow, &activatedCol, &rightRow, &rightCol, &sortCol, &scroll)
	manualHeaderRightCol := int32(-1)
	if !u.editing && rl.IsMouseButtonPressed(rl.MouseButtonRight) {
		mouse := rl.GetMousePosition()
		if mouse.Y >= tableViewTop && mouse.Y < tableTop {
			x := float32(24)
			for col, width := range props.ColumnWidths {
				if mouse.X >= x && mouse.X < x+float32(width) {
					manualHeaderRightCol = int32(col)
					break
				}
				x += float32(width)
			}
		}
	}
	if !u.editing {
		if copyText, ok := a.tableCopyText(u); ok {
			props.CopyText = &copyText
		}
		props.PastedText = &pastedText
		props.PastedRow = &pastedRow
		props.PastedColumn = &pastedCol
	}
	changed := rl.TableView(props)
	if rightCol < 0 && manualHeaderRightCol >= 0 {
		rightCol = manualHeaderRightCol
	}
	if !u.editing && (pastedText != "" || pastedRow >= 0 || pastedCol >= 0) {
		u.selRow, u.selCol = workbookSelectionFromTable(tableSelectedRow, tableSelectedCol)
		a.pasteTableText(u, pastedRow, pastedCol, pastedText)
		return
	}
	if !u.editing {
		u.selRow, u.selCol = workbookSelectionFromTable(tableSelectedRow, tableSelectedCol)
		u.rangeStartRow, u.rangeStartCol = int(rangeStartRow), int(rangeStartCol)
		u.rangeEndRow, u.rangeEndCol = int(rangeEndRow), int(rangeEndCol)
		if changed == 0 {
			a.moveSelectionFromKey(u)
		}
	}
	tableClearedSelection := hadSelection && tableSelectedRow < 0 && tableSelectedCol < 0
	u.sortCol = int(sortCol)
	u.offset = int(scroll / rowH)
	u.clamp(len(u.disp))
	a.drawFormulaRefs(u, props)
	a.handleFillDrag(u, props)

	if dataCol := dataColFromTableCol(activatedCol); activatedRow >= 0 && int(activatedRow) < len(u.disp) && dataCol >= 0 && a.rowCanEdit(u.disp[activatedRow], dataCol) {
		if u.editing {
			a.commitCell(u)
		}
		u.selRow, u.selCol = int(activatedRow), int(activatedCol)
		a.startEdit(u, u.disp[u.selRow], dataCol, true)
		return
	}

	if u.editing && (tableSelectedRow >= 0 || tableSelectedCol >= 0) {
		targetRow, targetCol := workbookSelectionFromTable(tableSelectedRow, tableSelectedCol)
		if targetRow == u.selRow && targetCol == u.selCol {
			return
		}
		a.commitCell(u)
		u.editFocus = false
		u.disp = a.dispRows()
		u.clamp(len(u.disp))
		if targetRow < len(u.disp) {
			u.selRow, u.selCol = targetRow, targetCol
		}
		return
	}

	if !u.editing {
		switch {
		case rightRow < 0 && rightCol > 0:
			if dataCol := dataColFromTableCol(rightCol); dataCol >= 0 {
				m := rl.GetMousePosition()
				u.selRow, u.selCol = -1, int(rightCol)
				u.ctx, u.ctxHeader, u.ctxIdx, u.ctxCol = true, true, -1, dataCol
				u.ctxX, u.ctxY = m.X, m.Y
			}
		case rightRow >= 0 && int(rightRow) < len(u.disp) && u.disp[rightRow].kind == dispRow:
			ctxCol := -2
			if rightCol == 0 {
				ctxCol = -1
			} else if dataCol := dataColFromTableCol(rightCol); dataCol >= 0 {
				ctxCol = dataCol
			}
			if ctxCol != -2 {
				m := rl.GetMousePosition()
				u.selRow, u.selCol = workbookSelectionFromTable(rightRow, rightCol)
				u.ctx, u.ctxHeader, u.ctxIdx, u.ctxCol = true, false, u.disp[rightRow].idx, ctxCol
				u.ctxX, u.ctxY = m.X, m.Y
			}
		case rl.IsKeyPressed(rl.KeyBackspace) && u.selRow >= 0 && u.selRow < len(u.disp) && a.rowCanEdit(u.disp[u.selRow], dataColFromTableCol(int32(u.selCol))):
			a.startEdit(u, u.disp[u.selRow], dataColFromTableCol(int32(u.selCol)), true)
		case rl.IsKeyPressed(rl.KeyDelete):
			a.clearCell(u)
		case rl.IsKeyPressed(rl.KeyEscape) && !tableClearedSelection:
			u.quit = true
		default:
			if dataCol := dataColFromTableCol(int32(u.selCol)); u.selRow >= 0 && u.selRow < len(u.disp) && a.rowCanEdit(u.disp[u.selRow], dataCol) {
				if s := typedChars(); s != "" {
					a.startEdit(u, u.disp[u.selRow], dataCol, false)
					clear(u.editText)
					copy(u.editText, s)
					u.editCur = int32(len(s))
					u.editSkip = true
				}
			}
		}
	}
}

func (a *app) drawFormulaRefs(u *uiState, props rl.TableViewProps) {
	formula := a.selectedFormulaText(u)
	if !strings.HasPrefix(strings.TrimSpace(formula), "=") {
		return
	}
	palette := []rl.Color{
		{R: 86, G: 170, B: 255, A: 255},
		{R: 96, G: 226, B: 142, A: 255},
		{R: 255, G: 207, B: 86, A: 255},
		{R: 255, G: 96, B: 164, A: 255},
	}
	for i, group := range a.formulaRefGroups(formula) {
		color := palette[i%len(palette)]
		for _, seg := range formulaRefSegments(group.Cells) {
			tableCol := tableColFromDataCol(seg.Col)
			if tableCol < 0 {
				continue
			}
			rect := rl.TableCellRect(props, int32(seg.StartRow), int32(tableCol))
			if rect.Width <= 0 || rect.Height <= 0 {
				continue
			}
			end := rl.TableCellRect(props, int32(seg.EndRow), int32(tableCol))
			if end.Width <= 0 || end.Height <= 0 {
				continue
			}
			rect.Height = end.Y + end.Height - rect.Y
			rl.DrawRectangleLinesEx(rect, 2, color)
		}
	}
}

func (a *app) handleFillDrag(u *uiState, props rl.TableViewProps) {
	if u.editing || u.selRow < 0 || u.selCol < 1 || u.selRow >= len(u.disp) {
		u.filling = false
		return
	}
	dataCol := dataColFromTableCol(int32(u.selCol))
	if dataCol < 0 || u.disp[u.selRow].kind != dispRow {
		u.filling = false
		return
	}
	formula := a.selectedFormulaText(u)
	if !strings.HasPrefix(strings.TrimSpace(formula), "=") {
		u.filling = false
		return
	}
	cell := rl.TableCellRect(props, int32(u.selRow), int32(u.selCol))
	if cell.Width <= 0 || cell.Height <= 0 {
		u.filling = false
		return
	}
	handle := fillHandleRect(cell)
	mouse := rl.GetMousePosition()
	if !u.filling && rl.IsMouseButtonPressed(rl.MouseButtonLeft) && rl.CheckCollisionPointRec(mouse, handle) {
		u.filling = true
		u.fillStartRow = u.selRow
		u.fillStartCol = dataCol
		u.fillEndRow = u.selRow
	}
	if u.filling {
		if row := fillTargetRow(u, props, mouse); row >= 0 {
			u.fillEndRow = row
		}
		a.drawFillPreview(u, props)
		if rl.IsMouseButtonReleased(rl.MouseButtonLeft) {
			a.applyFill(u)
			u.filling = false
		}
		return
	}
	rl.DrawRectangleRec(handle, colAccent)
}

func fillHandleRect(cell rl.Rectangle) rl.Rectangle {
	size := float32(7)
	return rl.NewRectangle(cell.X+cell.Width-size-1, cell.Y+cell.Height-size-1, size, size)
}

func fillTargetRow(u *uiState, props rl.TableViewProps, mouse rl.Vector2) int {
	best := -1
	for row := range u.disp {
		cell := rl.TableCellRect(props, int32(row), int32(u.selCol))
		if cell.Width <= 0 || cell.Height <= 0 {
			continue
		}
		if mouse.Y >= cell.Y && mouse.Y < cell.Y+cell.Height {
			return row
		}
		if mouse.Y >= cell.Y {
			best = row
		}
	}
	return best
}

func (a *app) drawFillPreview(u *uiState, props rl.TableViewProps) {
	start, end := u.fillStartRow, u.fillEndRow
	if start > end {
		start, end = end, start
	}
	for row := start; row <= end && row < len(u.disp); row++ {
		cell := rl.TableCellRect(props, int32(row), int32(u.selCol))
		if cell.Width <= 0 || cell.Height <= 0 {
			continue
		}
		rl.DrawRectangleLinesEx(cell, 1, colAccent)
	}
}

func (a *app) applyFill(u *uiState) {
	if !u.filling || u.fillStartRow < 0 || u.fillStartRow >= len(u.disp) {
		return
	}
	if u.disp[u.fillStartRow].kind != dispRow {
		return
	}
	sourceIdx := u.disp[u.fillStartRow].idx
	sourceFormula := a.rowCellEditText(sourceIdx, u.fillStartCol)
	if !strings.HasPrefix(strings.TrimSpace(sourceFormula), "=") {
		return
	}
	start, end := u.fillStartRow, u.fillEndRow
	if start > end {
		start, end = end, start
	}
	changed := false
	for row := start; row <= end && row < len(u.disp); row++ {
		if row == u.fillStartRow || u.disp[row].kind != dispRow {
			continue
		}
		targetIdx := u.disp[row].idx
		shifted := shiftFormulaRows(sourceFormula, targetIdx-sourceIdx)
		if a.setRowCell(targetIdx, u.fillStartCol, shifted) {
			changed = true
		}
	}
	if changed {
		a.save("formula filled")
		u.disp = a.dispRows()
		u.selRow = clampInt(u.fillEndRow, 0, len(u.disp)-1)
		u.selCol = tableColFromDataCol(u.fillStartCol)
	}
}

type formulaRefSegment struct {
	StartRow int
	EndRow   int
	Col      int
}

func formulaRefSegments(cells []formulaCellRef) []formulaRefSegment {
	if len(cells) == 0 {
		return nil
	}
	refs := append([]formulaCellRef(nil), cells...)
	sort.Slice(refs, func(i, j int) bool {
		if refs[i].Col != refs[j].Col {
			return refs[i].Col < refs[j].Col
		}
		return refs[i].Row < refs[j].Row
	})
	var out []formulaRefSegment
	cur := formulaRefSegment{StartRow: refs[0].Row, EndRow: refs[0].Row, Col: refs[0].Col}
	for _, ref := range refs[1:] {
		if ref.Col == cur.Col && ref.Row == cur.EndRow+1 {
			cur.EndRow = ref.Row
			continue
		}
		out = append(out, cur)
		cur = formulaRefSegment{StartRow: ref.Row, EndRow: ref.Row, Col: ref.Col}
	}
	return append(out, cur)
}

func (a *app) selectedFormulaText(u *uiState) string {
	if u.editing {
		return editString(u)
	}
	if u.selRow < 0 || u.selRow >= len(u.disp) {
		return ""
	}
	col := dataColFromTableCol(int32(u.selCol))
	if col < 0 {
		return ""
	}
	dr := u.disp[u.selRow]
	switch dr.kind {
	case dispRow:
		return a.rowCellEditText(dr.idx, col)
	case dispTotalPending, dispTotal:
		return a.totalCellEditText(dr.kind, col)
	}
	return ""
}

func (a *app) rowCanEdit(dr displayRow, col int) bool {
	if col < 0 || col > 7 {
		return false
	}
	switch dr.kind {
	case dispRow, dispBlank:
		return true
	case dispTotalPending, dispTotal:
		return true
	}
	return false
}

func (a *app) moveSelectionFromKey(u *uiState) {
	if u.selRow < 0 || u.selCol < 0 {
		return
	}
	oldRow, oldCol := u.selRow, u.selCol
	switch {
	case rl.IsKeyPressed(rl.KeyUp):
		u.selRow = clampInt(u.selRow-1, 0, len(u.disp)-1)
	case rl.IsKeyPressed(rl.KeyDown):
		u.selRow = clampInt(u.selRow+1, 0, len(u.disp)-1)
	case rl.IsKeyPressed(rl.KeyLeft):
		u.selCol = clampInt(u.selCol-1, 0, len(visibleDataCols))
	case rl.IsKeyPressed(rl.KeyRight):
		u.selCol = clampInt(u.selCol+1, 0, len(visibleDataCols))
	}
	if u.selRow != oldRow || u.selCol != oldCol {
		u.clearRange()
	}
}

func workbookSelectionFromTable(row, col int32) (int, int) {
	switch {
	case row >= 0 && col == 0:
		return int(row), -1
	default:
		return int(row), int(col)
	}
}

func (a *app) workbookTableProps(u *uiState, selectedRow, selectedCol, rangeStartRow, rangeStartCol, rangeEndRow, rangeEndCol, activatedRow, activatedCol, rightRow, rightCol, sortCol, scroll *int32) rl.TableViewProps {
	return rl.TableViewProps{
		Bounds:               rl.NewRectangle(24, tableViewTop, winW-48, tableBot-tableViewTop),
		ID:                   2001,
		Columns:              visibleTableColumns,
		Rows:                 a.workbookTableRows(u),
		ColumnWidths:         []int32{44, 84, 112, 238, 140, 150, 180, 184},
		SelectedRow:          selectedRow,
		SelectedColumn:       selectedCol,
		SelectionStartRow:    rangeStartRow,
		SelectionStartColumn: rangeStartCol,
		SelectionEndRow:      rangeEndRow,
		SelectionEndColumn:   rangeEndCol,
		ActivatedRow:         activatedRow,
		ActivatedColumn:      activatedCol,
		RightClickedRow:      rightRow,
		RightClickedColumn:   rightCol,
		SortColumn:           sortCol,
		ScrollOffset:         scroll,
		RowHeight:            rowH,
	}
}

func formulaBarInputRect() rl.Rectangle {
	return rl.NewRectangle(86, formulaBarTop, winW-110, formulaBarH)
}

func formulaBarAddressRect() rl.Rectangle {
	return rl.NewRectangle(24, formulaBarTop, 54, formulaBarH)
}

func (a *app) selectedCellAddress(u *uiState) string {
	col := dataColFromTableCol(int32(u.selCol))
	if u.selRow < 0 || col < 0 {
		return ""
	}
	return dataColLetter(col) + strconv.Itoa(u.selRow+1)
}

func (a *app) canEditSelection(u *uiState) bool {
	col := dataColFromTableCol(int32(u.selCol))
	return u.selRow >= 0 && u.selRow < len(u.disp) && col >= 0 && a.rowCanEdit(u.disp[u.selRow], col)
}

func copyFormulaBarText(dst []byte, s string) {
	clear(dst)
	copy(dst, s)
}

func (a *app) drawFormulaBar(u *uiState) {
	addrRect := formulaBarAddressRect()
	inputRect := formulaBarInputRect()
	rl.DrawRectangleRec(addrRect, colPanel)
	rl.DrawRectangleLinesEx(addrRect, 1, colDim)
	rl.DrawRectangleRec(inputRect, colBG)
	rl.DrawRectangleLinesEx(inputRect, 1, colDim)

	addr := a.selectedCellAddress(u)
	if addr != "" {
		rtxt(addr, addrRect.X+addrRect.Width-8, addrRect.Y+7, 14, colDim)
	}

	if rl.IsMouseButtonPressed(rl.MouseButtonLeft) && rl.CheckCollisionPointRec(rl.GetMousePosition(), inputRect) && a.canEditSelection(u) {
		if !u.editing {
			col := dataColFromTableCol(int32(u.selCol))
			a.startEdit(u, u.disp[u.selRow], col, true)
		}
		u.editBar = true
		u.editFocus = true
	}

	if u.editing && u.editBar {
		if rl.IsKeyPressed(rl.KeyEscape) {
			a.cancelEdit(u)
			rl.SetFocus(2001)
			return
		}
		commit := false
		rl.TextField(rl.TextFieldProps{
			Bounds:         inputRect,
			Text:           u.editText,
			CursorPosition: &u.editCur,
			Focused:        &u.editFocus,
			MaxCodepoints:  int32(len(u.editText) - 1),
			Font:           rl.Text16,
			FocusID:        3002,
			CommitPressed:  &commit,
		})
		if commit {
			a.commitCell(u)
			u.editFocus = false
			rl.SetFocus(2001)
			a.advanceAfterEditKey(u)
		} else if !u.editFocus || rl.IsKeyPressed(rl.KeyTab) {
			a.cancelEdit(u)
			rl.SetFocus(2001)
			a.advanceAfterEditKey(u)
		}
		return
	}

	barText := a.selectedFormulaText(u)
	if barText == "" {
		return
	}
	buf := make([]byte, 1024)
	copyFormulaBarText(buf, barText)
	cursor := int32(len(barText))
	focused := false
	rl.TextField(rl.TextFieldProps{
		Bounds:         inputRect,
		Text:           buf,
		CursorPosition: &cursor,
		Focused:        &focused,
		MaxCodepoints:  int32(len(buf) - 1),
		Font:           rl.Text16,
		FocusID:        3003,
	})
}

func (a *app) workbookTableRows(u *uiState) []rl.UITableRow {
	a.ensureProfileCells()
	rows := make([]rl.UITableRow, len(u.disp))
	for i, dr := range u.disp {
		rowNumber := strconv.Itoa(i + 1)
		switch dr.kind {
		case dispRow:
			row := a.workbookDataRow(dr)
			row.Cells = append([]string{rowNumber}, row.Cells...)
			rows[i] = row
		case dispBlank:
			cells := make([]string, len(visibleTableColumns))
			cells[0] = rowNumber
			rows[i] = rl.UITableRow{Cells: cells}
		case dispTotalPending, dispTotal:
			rows[i] = a.workbookTotalRow(dr.kind, rowNumber)
		}
	}
	return rows
}

func (a *app) workbookTotalRow(kind int, rowNumber string) rl.UITableRow {
	cells := make([]string, len(visibleDataCols))
	for i, col := range visibleDataCols {
		cells[i] = a.totalCellDisplayText(kind, col)
	}
	return rl.UITableRow{Cells: append([]string{rowNumber}, cells...)}
}

func (a *app) workbookDataRow(dr displayRow) rl.UITableRow {
	as := a.wb.Rows[dr.idx]
	units, _ := a.cellNumber(dr.idx, 3, map[string]bool{})
	rate, _ := a.cellNumber(dr.idx, 4, map[string]bool{})
	usd, usdOK := a.cellNumber(dr.idx, 6, map[string]bool{})
	section := as.Section
	name := coinSymbol(as.ID)
	if a.stale[as.ID] && rate > 0 {
		name += " *"
	}
	usdText := "-"
	if usdOK {
		usdText = commaf(usd)
	}
	dusdText := "-"
	if dusd, ok := a.cellNumber(dr.idx, 7, map[string]bool{}); ok {
		dusdText = signed(commaf(dusd))
	}
	unitsText := fmtUnits(units)
	rateText := fmtRate(rate)
	row := rl.UITableRow{Cells: []string{
		section,
		as.Label,
		name,
		unitsText,
		rateText,
		usdText,
		dusdText,
	}}
	for i, dataCol := range visibleDataCols {
		if raw, ok := a.wb.CellValues[cellFormatKey(dr.idx, dataCol)]; ok {
			if strings.HasPrefix(strings.TrimSpace(raw), "=") {
				if value, ok := a.cellNumber(dr.idx, dataCol, map[string]bool{}); ok {
					row.Cells[i] = strconv.FormatFloat(value, 'f', -1, 64)
				} else {
					row.Cells[i] = "#REF!"
				}
			} else {
				row.Cells[i] = raw
			}
		}
	}
	row.TextColors = make([]rl.Color, len(visibleTableColumns))
	row.BackgroundColors = make([]rl.Color, len(visibleTableColumns))
	for i, dataCol := range visibleDataCols {
		format := a.cellFormat(dr.idx, dataCol)
		if color, ok := parseColorHex(format.TextColor); ok {
			row.TextColors[i+1] = color
		}
		if color, ok := parseColorHex(format.BackgroundColor); ok {
			row.BackgroundColors[i+1] = color
		}
		if format.Conditional == "sign" {
			if value, ok := a.cellNumber(dr.idx, dataCol, map[string]bool{}); ok {
				row.TextColors[i+1] = signColor(value)
			}
		}
	}
	return row
}

// drawEditor paints the in-place cell editor over the edited cell.
func (a *app) drawEditor(u *uiState) {
	if !u.editing || u.editBar || u.selRow < u.offset || u.selRow >= len(u.disp) {
		return
	}
	if u.editSkip {
		u.editSkip = false
		return
	}
	scroll := int32(u.offset * rowH)
	props := a.workbookTableProps(u, ptr32(int32(u.selRow)), ptr32(int32(u.selCol)), ptr32(-1), ptr32(-1), ptr32(-1), ptr32(-1), ptr32(-1), ptr32(-1), ptr32(-1), ptr32(-1), ptr32(int32(u.sortCol)), &scroll)
	r := rl.TableCellRect(props, int32(u.selRow), int32(u.selCol))
	if r.Width <= 0 || r.Height <= 0 {
		return
	}
	if rl.IsKeyPressed(rl.KeyEscape) {
		a.cancelEdit(u)
		rl.SetFocus(2001)
		return
	}
	commit := false
	rl.TextField(rl.TextFieldProps{
		Bounds:         r,
		Text:           u.editText,
		CursorPosition: &u.editCur,
		Focused:        &u.editFocus,
		MaxCodepoints:  int32(len(u.editText) - 1),
		Font:           rl.Text16,
		FocusID:        3001,
		CommitPressed:  &commit,
	})
	if commit {
		a.commitCell(u)
		u.editFocus = false
		rl.SetFocus(2001)
		a.advanceAfterEditKey(u)
	} else if !u.editFocus || rl.IsKeyPressed(rl.KeyTab) {
		a.cancelEdit(u)
		rl.SetFocus(2001)
		a.advanceAfterEditKey(u)
	}
}

func (a *app) cancelEdit(u *uiState) {
	if u.editNew && u.editIdx >= 0 && u.editIdx < len(a.wb.Rows) {
		a.wb.Rows = append(a.wb.Rows[:u.editIdx], a.wb.Rows[u.editIdx+1:]...)
		u.disp = a.dispRows()
		u.selRow = clampInt(u.editIdx, 0, len(u.disp)-1)
	}
	u.editing = false
	u.editFocus = false
	u.editNew = false
	u.editTotal = 0
	u.editBar = false
}

func (a *app) advanceAfterEditKey(u *uiState) {
	if rl.IsKeyPressed(rl.KeyEnter) {
		u.selRow++
	} else if rl.IsKeyPressed(rl.KeyTab) {
		if u.selCol < len(visibleDataCols) {
			u.selCol++
		} else {
			u.selCol, u.selRow = 1, u.selRow+1
		}
	}
}

func visibleEditText(s string, cursor int, maxW, size float32) (text, caretPrefix string) {
	r := []rune(s)
	cursor = clampInt(cursor, 0, len(r))
	start := 0
	for start < cursor && wtxt(string(r[start:cursor]), size) > maxW {
		start++
	}
	end := len(r)
	for end > cursor && wtxt(string(r[start:end]), size) > maxW {
		end--
	}
	return string(r[start:end]), string(r[start:cursor])
}

func ptr32(v int32) *int32 {
	return &v
}
