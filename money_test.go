package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEvalUnits(t *testing.T) {
	ok := map[string]float64{
		"3.7":         3.7,
		"0.1+0.25":    0.35,
		"0.1 + 0.25":  0.35,
		"(2+3)*0.1":   0.5,
		"-0.5+1":      0.5,
		"2*(1+2)":     6,
		"1e-3*2":      0.002,
		"0.1+0.25*2":  0.6,
		"((0.4))/0.5": 0.8,
		"=0.1":        0.1,
		"=0.1+0.05":   0.15,
		"=0.1+0.25":   0.35,
	}
	for in, want := range ok {
		got, err := evalUnits(in)
		if err != nil || got != want {
			t.Errorf("evalUnits(%q) = %v, %v; want %v", in, got, err, want)
		}
	}
	bad := []string{"", "0.1+", "1/0", "foo", "1 2", "(1+2", "0..1"}
	for _, in := range bad {
		if v, err := evalUnits(in); err == nil {
			t.Errorf("evalUnits(%q) = %v; want error", in, v)
		}
	}
}

func TestEvalExprs(t *testing.T) {
	wb := newWorkbook()
	wb.Rows = []Row{
		{ID: "bitcoin", Units: 99, Expr: "0.1+0.25"},
		{ID: "litecoin", Units: 3},
	}
	wb.evalExprs()
	if wb.Rows[0].Units != 0.35 {
		t.Errorf("expr units = %v, want 0.35", wb.Rows[0].Units)
	}
	if wb.Rows[1].Units != 3 {
		t.Errorf("plain units = %v, want 3", wb.Rows[1].Units)
	}
}

func TestWorkbookPersistenceUsesKryFile(t *testing.T) {
	dir := t.TempDir()
	wb := &Workbook{
		Rows:  []Row{{Section: "banks", Label: "checking", ID: "USD", Units: 100}},
		Rates: map[string]float64{"USD": 1},
	}
	if err := saveWorkbook(dir, wb); err != nil {
		t.Fatalf("saveWorkbook failed: %v", err)
	}
	path := filepath.Join(dir, workbookFile)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("workbook file was not written: %v", err)
	}
	legacyRowsKey := `"ass` + `ets"`
	if strings.Contains(string(data), legacyRowsKey) || !strings.Contains(string(data), `"rows"`) {
		t.Fatalf("workbook kry schema = %s, want rows key and no legacy rows key", data)
	}
	loaded, _, err := loadWorkbook(dir)
	if err != nil {
		t.Fatalf("loadWorkbook failed: %v", err)
	}
	if len(loaded.Rows) != 1 || loaded.Rows[0].Label != "checking" || loaded.Rates["USD"] != 1 {
		t.Fatalf("loaded workbook = %#v", loaded)
	}
}

func TestLoadMigratesLegacyWorkbookJSONToKry(t *testing.T) {
	dir := t.TempDir()
	legacyPath := filepath.Join(dir, legacyWorkbookFile)
	legacy := `{"rows":[{"label":"legacy","id":"USD","units":7}],"rates":{"USD":1}}`
	if err := os.WriteFile(legacyPath, []byte(legacy), 0o600); err != nil {
		t.Fatal(err)
	}

	loaded, from, err := loadWorkbook(dir)
	if err != nil {
		t.Fatalf("load legacy workbook failed: %v", err)
	}
	if from != legacyWorkbookFile || len(loaded.Rows) != 1 || loaded.Rows[0].Label != "legacy" {
		t.Fatalf("migration result = from %q, workbook %#v", from, loaded)
	}
	if _, err := os.Stat(filepath.Join(dir, workbookFile)); err != nil {
		t.Fatalf("migrated workbook was not written: %v", err)
	}
	if _, err := os.Stat(legacyPath); !os.IsNotExist(err) {
		t.Fatalf("legacy workbook still exists after migration: %v", err)
	}
}

func TestSaveKeepsVisibleTotalFormulas(t *testing.T) {
	dir := t.TempDir()
	a := &app{
		dir: dir,
		wb: &Workbook{
			Rows: []Row{
				{Section: "banks", Label: "checking", ID: "USD", Units: 100},
				{Section: "pending", Label: "incoming", ID: "USD", Units: 50},
			},
			Rates: map[string]float64{"USD": 1},
		},
	}
	a.save("test")

	data, err := os.ReadFile(filepath.Join(dir, workbookFile))
	if err != nil {
		t.Fatalf("read workbook failed: %v", err)
	}
	got := string(data)
	for _, want := range []string{
		`"total_pending_cells"`,
		`"=SUM(G1:G2)"`,
		`"total_cells"`,
		`"=SUM(G1:G2)-SUM(G2:G2)"`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("workbook json missing %q: %s", want, got)
		}
	}
	for _, unwanted := range []string{"total_expr", "total_pending_expr"} {
		if strings.Contains(got, unwanted) {
			t.Fatalf("duplicate total field %q survived in workbook json: %s", unwanted, got)
		}
	}
}

func TestSaveRefreshesDefaultTotalFormulasWhenRowsChange(t *testing.T) {
	dir := t.TempDir()
	a := &app{
		dir: dir,
		wb: &Workbook{
			Rows: []Row{
				{Section: "banks", Label: "checking", ID: "USD", Units: 100},
				{Section: "pending", Label: "incoming", ID: "USD", Units: 50},
			},
			Rates: map[string]float64{"USD": 1},
		},
	}
	a.save("initial")
	a.wb.Rows = append(a.wb.Rows, Row{Section: "", Label: "incoming 2", ID: "USD", Units: 25})
	a.save("changed")

	if got, want := a.wb.TotalPendingCells[6], "=SUM(G1:G3)"; got != want {
		t.Fatalf("total + pending formula = %q, want %q", got, want)
	}
	if got, want := a.wb.TotalCells[6], "=SUM(G1:G3)-SUM(G2:G3)"; got != want {
		t.Fatalf("total formula = %q, want %q", got, want)
	}
}

func TestSavePreservesCustomTotalFormula(t *testing.T) {
	dir := t.TempDir()
	a := &app{
		dir: dir,
		wb: &Workbook{
			Rows: []Row{
				{Section: "banks", Label: "checking", ID: "USD", Units: 100},
			},
			Rates:      map[string]float64{"USD": 1},
			TotalCells: []string{"", "total", "", "", "", "", "=G1+25"},
		},
	}
	a.save("custom")

	if got := a.wb.TotalCells[6]; got != "=G1+25" {
		t.Fatalf("custom total formula = %q", got)
	}
}

func TestCellFormatsRoundTripAndShiftWithRows(t *testing.T) {
	dir := t.TempDir()
	a := &app{
		dir: dir,
		wb: &Workbook{
			Rows: []Row{
				{Section: "banks", Label: "checking", ID: "USD", Units: 100},
				{Section: "cash", Label: "wallet", ID: "EUR", Units: 50},
			},
			Rates: map[string]float64{"USD": 1, "EUR": 1.1},
		},
	}
	a.setCellTextColor(1, 1, colAccent)
	a.setCellBackgroundColor(1, 1, colSel)
	a.shiftCellFormatsForInsert(1)
	a.save("formatted")

	loaded, _, err := loadWorkbook(dir)
	if err != nil {
		t.Fatalf("load formatted workbook failed: %v", err)
	}
	format := loaded.CellFormats[cellFormatKey(2, 1)]
	if format.TextColor == "" || format.BackgroundColor == "" {
		t.Fatalf("shifted cell format missing: %#v", loaded.CellFormats)
	}

	loadedApp := &app{wb: loaded}
	loadedApp.shiftCellFormatsForDelete(1)
	if _, ok := loaded.CellFormats[cellFormatKey(2, 1)]; ok {
		t.Fatalf("old shifted format key survived delete: %#v", loaded.CellFormats)
	}
	if _, ok := loaded.CellFormats[cellFormatKey(1, 1)]; !ok {
		t.Fatalf("format did not shift back after delete: %#v", loaded.CellFormats)
	}
}

func TestFmtUnits(t *testing.T) {
	cases := map[float64]string{
		0:        "0",
		3.7:      "3.7",
		0.35:     "0.35",
		1211:     "1,211",
		9123.171: "9123.171",
		2:        "2",
	}
	for in, want := range cases {
		if got := fmtUnits(in); got != want {
			t.Errorf("fmtUnits(%v) = %q, want %q", in, got, want)
		}
	}
}

func TestGroupSections(t *testing.T) {
	wb := &Workbook{Rows: []Row{
		{Section: "banks", ID: "demo-checking"},
		{Section: "crypto", ID: "bitcoin"},
		{Section: "", ID: "legacy"},
		{Section: "banks", ID: "travel-savings"},
	}}
	wb.groupSections()
	got := make([]string, len(wb.Rows))
	for i, as := range wb.Rows {
		got[i] = as.ID
	}
	want := []string{"demo-checking", "travel-savings", "bitcoin", "legacy"} // banks first
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("grouped order = %v, want %v", got, want)
		}
	}
}

func TestCommaf(t *testing.T) {
	cases := map[float64]string{
		0:          "0.00",
		1234567.89: "1,234,567.89",
		-42.5:      "-42.50",
		999.999:    "1,000.00",
	}
	for in, want := range cases {
		if got := commaf(in); got != want {
			t.Errorf("commaf(%v) = %q, want %q", in, got, want)
		}
	}
}
