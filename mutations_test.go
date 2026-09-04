package main

import "testing"

func TestFreeTextInNumericCell(t *testing.T) {
	a := &app{profile: profileWorkbook, wb: newWorkbook()}
	a.wb.Rows = []Row{{}}
	if !a.setRowCell(0, 3, "delivery next week") {
		t.Fatal("free text was rejected")
	}
	if got := a.rowCellEditText(0, 3); got != "delivery next week" {
		t.Fatalf("edit text = %q", got)
	}
	rows := a.workbookTableRows(&uiState{disp: a.dispRows()})
	if got := rows[0].Cells[4]; got != "delivery next week" {
		t.Fatalf("display text = %q", got)
	}
}

func TestDeleteRowRewritesMovedFormulaReferences(t *testing.T) {
	a := &app{profile: profileWorkbook, wb: newWorkbook()}
	a.wb.Rows = []Row{{}, {}, {USDExpr: "A3+B3"}, {USDExpr: "A3+B4"}}
	u := &uiState{disp: a.dispRows()}
	a.deleteRow(u, 1)
	if got := a.wb.Rows[1].USDExpr; got != "A2+B2" {
		t.Fatalf("moved formula = %q", got)
	}
	if got := a.wb.Rows[2].USDExpr; got != "A2+B3" {
		t.Fatalf("following formula = %q", got)
	}
}

func TestStructuralFormulaRewriteContractsRanges(t *testing.T) {
	if got := rewriteFormulaRows("=SUM(A10:A14)+B13", 11, false); got != "=SUM(A10:A13)+B12" {
		t.Fatalf("row delete rewrite = %q", got)
	}
	if got := rewriteFormulaRows("=A12+B13", 11, false); got != "=#REF!+B12" {
		t.Fatalf("deleted reference rewrite = %q", got)
	}
	if got := rewriteFormulaColumns("=C4+D4", 1, false); got != "=B4+C4" {
		t.Fatalf("column delete rewrite = %q", got)
	}
}

func TestDeleteColumnMovesCellsAndRewritesFormula(t *testing.T) {
	a := &app{dir: t.TempDir(), profile: profileWorkbook, wb: newWorkbook()}
	a.wb.Rows = []Row{{Section: "first", Label: "second", ID: "third", DUSDExpr: "C1+D1"}}
	u := &uiState{disp: a.dispRows()}
	a.mutateColumn(u, 1, false)
	if got := a.rowCellEditText(0, 1); got != "third" {
		t.Fatalf("shifted B cell = %q", got)
	}
	if got := a.rowCellEditText(0, 6); got != "=B1+C1" {
		t.Fatalf("shifted formula = %q", got)
	}
}

func TestConditionalSignFormatting(t *testing.T) {
	v := -2.0
	a := &app{wb: &Workbook{Rows: []Row{{USD: &v}}, Rates: map[string]float64{}, CellFormats: map[string]CellFormat{}, CellValues: map[string]string{}}}
	a.toggleCellConditional(0, 6)
	rows := a.workbookTableRows(&uiState{disp: a.dispRows()})
	if got := rows[0].TextColors[6]; got != colRed {
		t.Fatalf("conditional color = %#v", got)
	}
}
