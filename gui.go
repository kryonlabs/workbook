package main

import (
	_ "embed"
	"fmt"
	"os"
	"strconv"

	rl "github.com/waozixyz/kryon/go/kryon"
)

// The Kryon application shell, following core/gui.go: window, fonts, input
// handling and the frame loop. The dark theme nods to the old plan9-style
// boot screen run.sh used to draw.

//go:embed fonts/NotoSansMono-Regular.ttf
var notoSansMonoTTF []byte

var (
	colBG     = rl.Color{R: 10, G: 10, B: 14, A: 255}
	colPanel  = rl.Color{R: 40, G: 44, B: 52, A: 255}
	colText   = rl.Color{R: 228, G: 230, B: 235, A: 255}
	colDim    = rl.Color{R: 118, G: 124, B: 138, A: 255}
	colAccent = rl.Color{R: 86, G: 170, B: 255, A: 255}
	colSel    = rl.Color{R: 30, G: 48, B: 70, A: 255}
	colGreen  = rl.Color{R: 96, G: 226, B: 142, A: 255}
	colRed    = rl.Color{R: 255, G: 96, B: 96, A: 255}
)

func loadUIFont() {
	if rl.RegisterUIFontData("cell-mono", ".ttf", notoSansMonoTTF, nil) {
		rl.UseUIFont("cell-mono")
	}
}

type uiState struct {
	selRow  int // index into disp, -1 = none
	selCol  int // visible table column: -1 whole row · 0 row number · 1 section · 2 label · 3 coin · 4 units
	offset  int
	sortCol int
	disp    []displayRow

	editing   bool
	editText  []byte
	editCur   int32 // byte cursor in editText
	editFocus bool
	editIdx   int // workbook row index, -1 otherwise
	editSkip  bool
	editNew   bool
	editTotal int
	editBar   bool

	ctx       bool // right-click row/cell menu
	ctxIdx    int
	ctxCol    int  // -1 row action, otherwise editable data column
	ctxHeader bool // column-header action rather than a cell action
	ctxX      float32
	ctxY      float32

	filling      bool
	fillStartRow int
	fillStartCol int
	fillEndRow   int

	openMenu int32

	rangeStartRow int
	rangeStartCol int
	rangeEndRow   int
	rangeEndCol   int

	quit bool
}

// runGUI opens the window and blocks until it is closed.
func runGUI(a *app) {
	if a.info != "" {
		fmt.Println(a.commandName()+":", a.info)
	}
	rl.SetSingleInstance(false)
	rl.SetConfigFlags(rl.FlagWindowResizable)
	rl.InitWindow(winW, winH, a.windowTitle())
	if !rl.IsWindowReady() {
		fmt.Fprintln(os.Stderr, a.commandName()+": window did not initialize")
		return
	}
	rl.SetTargetFPS(30)
	rl.SetExitKey(0) // esc is a cell-edit key; quitting is handled ourselves
	loadUIFont()
	defer rl.CloseWindow()

	u := &uiState{
		selRow: -1, selCol: -1, editText: make([]byte, 1024), openMenu: -1,
		rangeStartRow: -1, rangeStartCol: -1, rangeEndRow: -1, rangeEndCol: -1,
	}
	if a.isGeldProfile() {
		a.startRefresh()
	}

	// Screenshot hook for smoke tests.
	shot := os.Getenv("WORKBOOK_SCREENSHOT")
	if shot == "" {
		shot = os.Getenv("GELD_SCREENSHOT")
	}
	shotAfter := 2.5
	delay := os.Getenv("WORKBOOK_SHOT_AFTER")
	if delay == "" {
		delay = os.Getenv("GELD_SHOT_AFTER")
	}
	if v := delay; v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
			shotAfter = f
		}
	}

	for !rl.WindowShouldClose() && !u.quit {
		if !u.editing {
			if a.isGeldProfile() {
				a.pollRefresh()
			}
		}
		u.disp = a.dispRows()
		u.clamp(len(u.disp))
		draw(a, u)
		if shot != "" && rl.GetTime() > shotAfter {
			rl.TakeScreenshot(shot)
			break
		}
	}
}

// visRows is how many table rows fit in the window.
func visRows() int { return (tableBot - tableTop) / rowH }

func (u *uiState) clamp(n int) {
	if max := n - visRows(); n > visRows() && u.offset > max {
		u.offset = max
	}
	if u.offset < 0 {
		u.offset = 0
	}
	if u.selRow > n-1 {
		u.selRow = n - 1
	}
	if u.selRow < -1 {
		u.selRow = -1
	}
	if u.selRow >= 0 {
		if u.selRow < u.offset {
			u.offset = u.selRow
		}
		if u.selRow >= u.offset+visRows() {
			u.offset = u.selRow - visRows() + 1
		}
	}
}
