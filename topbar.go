package main

import (
	"fmt"

	rl "github.com/waozixyz/kryon/go/kryon"
)

const (
	menuIDSave int32 = 5001 + iota
	menuIDQuit
	menuIDInsertAbove
	menuIDInsertBelow
	menuIDDeleteRow
	menuIDClearCell
	menuIDTextColor
	menuIDBackgroundColor
	menuIDClearFormat
)

const (
	toolbarActionSave = iota
	toolbarActionInsert
	toolbarActionDelete
	toolbarActionTextColor
	toolbarActionBackgroundColor
	toolbarActionClearFormat
)

func (a *app) drawTopChrome(u *uiState) {
	if u.openMenu == 0 {
		u.openMenu = -1
	}
	actions := []rl.ToolbarAction{
		{IconType: rl.UIIconTypeSave},
		{IconType: rl.UIIconTypePlus},
		{IconType: rl.UIIconTypeTrash},
		{IconType: rl.UIIconTypeWorkbookTextColor},
		{IconType: rl.UIIconTypeWorkbookFillColor},
		{IconType: rl.UIIconTypeWorkbookClearFormatting},
	}
	toolbar := rl.Toolbar(rl.ToolbarProps{
		ID:                4100,
		X:                 0,
		Y:                 toolbarTop,
		Width:             int32(rl.GetScreenWidth()),
		Height:            toolbarH,
		Actions:           actions,
		ActionCount:       int32(len(actions)),
		ActionIconSize:    18,
		ActionIconPadding: 6,
		ActionGap:         8,
		SidePadding:       18,
	})
	if toolbar.ClickedAction >= 0 {
		a.handleToolbarAction(u, int(toolbar.ClickedAction))
	}

	status, sc := statusLine(a)
	rtxt(ellip(status, 360, 13), winW-280, float32(toolbarTop+14), 13, sc)
	if a.info != "" {
		txt(ellip(a.info, 420, 13), 24, float32(toolbarTop+14), 13, colAccent)
	}

	menus := a.topMenus(u)
	menu := rl.MenuBar(4000, rl.NewRectangle(0, 0, float32(rl.GetScreenWidth()), float32(menuBarH)), menus, &u.openMenu)
	if menu.ActivatedID != 0 {
		a.handleMenuAction(u, menu.ActivatedID)
	}
}

func (a *app) topMenus(u *uiState) []rl.Menu {
	canRow := selectedWorkbookRow(u) >= 0
	canCell := selectedDataCell(u) != nil
	active := "active: " + a.commandName()
	return []rl.Menu{
		{
			Label: "File",
			Items: []rl.MenuItem{
				{Kind: rl.MenuCommand, Label: "Save", Accelerator: "Ctrl+S", ID: menuIDSave},
				{Kind: rl.MenuSeparator},
				{Kind: rl.MenuCommand, Label: "Quit", Accelerator: "Esc", ID: menuIDQuit},
			},
		},
		{
			Label: "Edit",
			Items: []rl.MenuItem{
				{Kind: rl.MenuCommand, Label: "Insert row above", ID: menuIDInsertAbove, Disabled: !canRow},
				{Kind: rl.MenuCommand, Label: "Insert row below", ID: menuIDInsertBelow, Disabled: !canRow},
				{Kind: rl.MenuCommand, Label: "Delete row", ID: menuIDDeleteRow, Disabled: !canRow},
				{Kind: rl.MenuSeparator},
				{Kind: rl.MenuCommand, Label: "Clear cell", ID: menuIDClearCell, Disabled: !canCell},
			},
		},
		{
			Label: "Format",
			Items: []rl.MenuItem{
				{Kind: rl.MenuCommand, Label: "Text color", ID: menuIDTextColor, Disabled: !canCell},
				{Kind: rl.MenuCommand, Label: "Background color", ID: menuIDBackgroundColor, Disabled: !canCell},
				{Kind: rl.MenuCommand, Label: "Clear formatting", ID: menuIDClearFormat, Disabled: !canCell},
			},
		},
		{
			Label: "Profile",
			Items: []rl.MenuItem{
				{Kind: rl.MenuCommand, Label: active, Disabled: true},
			},
		},
	}
}

func (a *app) handleToolbarAction(u *uiState, action int) {
	switch action {
	case toolbarActionSave:
		a.save("saved")
	case toolbarActionInsert:
		if row := selectedWorkbookRow(u); row >= 0 {
			a.insertRowRelative(u, row, true)
		}
	case toolbarActionDelete:
		if row := selectedWorkbookRow(u); row >= 0 {
			a.deleteRow(u, row)
		}
	case toolbarActionTextColor:
		a.applyNextTextColor(u)
	case toolbarActionBackgroundColor:
		a.applyNextBackgroundColor(u)
	case toolbarActionClearFormat:
		a.clearSelectedFormat(u)
	}
}

func (a *app) handleMenuAction(u *uiState, id int32) {
	switch id {
	case menuIDSave:
		a.save("saved")
	case menuIDQuit:
		u.quit = true
	case menuIDInsertAbove:
		if row := selectedWorkbookRow(u); row >= 0 {
			a.insertRowRelative(u, row, false)
		}
	case menuIDInsertBelow:
		if row := selectedWorkbookRow(u); row >= 0 {
			a.insertRowRelative(u, row, true)
		}
	case menuIDDeleteRow:
		if row := selectedWorkbookRow(u); row >= 0 {
			a.deleteRow(u, row)
		}
	case menuIDClearCell:
		a.clearCell(u)
	case menuIDTextColor:
		a.applyNextTextColor(u)
	case menuIDBackgroundColor:
		a.applyNextBackgroundColor(u)
	case menuIDClearFormat:
		a.clearSelectedFormat(u)
	}
}

type selectedCell struct {
	row int
	col int
}

func selectedDataCell(u *uiState) *selectedCell {
	if u.selRow < 0 || u.selRow >= len(u.disp) || u.disp[u.selRow].kind != dispRow {
		return nil
	}
	col := dataColFromTableCol(int32(u.selCol))
	if col < 0 {
		return nil
	}
	return &selectedCell{row: u.disp[u.selRow].idx, col: col}
}

func selectedWorkbookRow(u *uiState) int {
	if u.selRow < 0 || u.selRow >= len(u.disp) || u.disp[u.selRow].kind != dispRow {
		return -1
	}
	return u.disp[u.selRow].idx
}

func (a *app) applyNextTextColor(u *uiState) {
	cells := a.selectedRangeDataCells(u)
	if len(cells) == 0 {
		return
	}
	format := a.cellFormat(cells[0].row, cells[0].col)
	color := nextPaletteColor(format.TextColor, formatPalette)
	for _, cell := range cells {
		a.setCellTextColor(cell.row, cell.col, color)
	}
	a.save(fmt.Sprintf("%s text color", a.selectionLabel(u)))
}

func (a *app) applyNextBackgroundColor(u *uiState) {
	cells := a.selectedRangeDataCells(u)
	if len(cells) == 0 {
		return
	}
	format := a.cellFormat(cells[0].row, cells[0].col)
	color := nextPaletteColor(format.BackgroundColor, backgroundPalette)
	for _, cell := range cells {
		a.setCellBackgroundColor(cell.row, cell.col, color)
	}
	a.save(fmt.Sprintf("%s background", a.selectionLabel(u)))
}

func (a *app) clearSelectedFormat(u *uiState) {
	cells := a.selectedRangeDataCells(u)
	if len(cells) == 0 {
		return
	}
	for _, cell := range cells {
		a.clearCellFormat(cell.row, cell.col)
	}
	a.save(fmt.Sprintf("%s format cleared", a.selectionLabel(u)))
}

func (a *app) selectionLabel(u *uiState) string {
	startRow, startCol, endRow, endCol, ok := u.rangeBounds()
	if !ok {
		return a.selectedCellAddress(u)
	}
	startDataCol := dataColFromTableCol(int32(startCol))
	endDataCol := dataColFromTableCol(int32(endCol))
	if startDataCol < 0 || endDataCol < 0 {
		return a.selectedCellAddress(u)
	}
	return dataColLetter(startDataCol) + fmt.Sprint(startRow+1) + ":" + dataColLetter(endDataCol) + fmt.Sprint(endRow+1)
}
