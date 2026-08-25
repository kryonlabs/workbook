package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const workbookFile = "workbook.json"

// fiat codes accepted as row ids, besides coingecko ids.
var fiatCodes = map[string]bool{
	"EUR": true, "USD": true, "GBP": true, "JPY": true, "BRL": true,
	"CHF": true, "AUD": true, "CAD": true, "SGD": true, "IDR": true, "THB": true,
	"PYG": true,
}

func isFiat(id string) bool { return fiatCodes[strings.ToUpper(id)] }

// Row is one holding: a coingecko id (or fiat code) with a number of units.
type Row struct {
	Section  string   `json:"section"` // group the row belongs to: crypto, banks…
	Label    string   `json:"label"`   // wallet, account, or note
	ID       string   `json:"id"`      // coingecko id or fiat code (EUR, USD, ...)
	Units    float64  `json:"units"`
	Expr     string   `json:"units_expr,omitempty"` // formula units come from, e.g. "0.1+0.25"
	Rate     *float64 `json:"rate,omitempty"`
	RateExpr string   `json:"rate_expr,omitempty"`
	Pct      *float64 `json:"pct,omitempty"`
	PctExpr  string   `json:"pct_expr,omitempty"`
	USD      *float64 `json:"usd,omitempty"`
	USDExpr  string   `json:"usd_expr,omitempty"`
	DUSD     *float64 `json:"dusd,omitempty"`
	DUSDExpr string   `json:"dusd_expr,omitempty"`
}

type CellFormat struct {
	TextColor       string `json:"text_color,omitempty"`
	BackgroundColor string `json:"background_color,omitempty"`
}

// evalExprs recomputes Units from Expr where set, so milestone formulas stay
// authoritative on load.
func (wb *Workbook) evalExprs() {
	for i := range wb.Rows {
		if wb.Rows[i].Expr != "" {
			if v, err := evalUnits(wb.Rows[i].Expr); err == nil {
				wb.Rows[i].Units = v
			}
		}
	}
}

// Workbook is everything persisted in workbook.json.
type Workbook struct {
	Rows              []Row                 `json:"rows"`
	Rates             map[string]float64    `json:"rates"` // id -> usd per unit, last known
	CellFormats       map[string]CellFormat `json:"cell_formats,omitempty"`
	TotalPendingCells []string              `json:"total_pending_cells,omitempty"`
	TotalCells        []string              `json:"total_cells,omitempty"`
	Updated           time.Time             `json:"updated"`
}

func newWorkbook() *Workbook {
	return &Workbook{Rates: map[string]float64{}, CellFormats: map[string]CellFormat{}}
}

// loadWorkbook loads workbook.json, creating an empty private workbook on first
// run.
func loadWorkbook(dir string) (*Workbook, string, error) {
	path := filepath.Join(dir, workbookFile)
	if data, err := os.ReadFile(path); err == nil {
		wb := newWorkbook()
		if err := json.Unmarshal(data, wb); err != nil {
			return nil, "", fmt.Errorf("%s: %w", path, err)
		}
		if wb.Rates == nil {
			wb.Rates = map[string]float64{}
		}
		if wb.CellFormats == nil {
			wb.CellFormats = map[string]CellFormat{}
		}
		wb.evalExprs()
		return wb, "", nil
	} else if !os.IsNotExist(err) {
		return nil, "", err
	}

	wb := newWorkbook()
	if err := saveWorkbook(dir, wb); err != nil {
		return nil, "", err
	}
	return wb, "", nil
}

// groupSections stable-groups rows by section in first-seen order, so the
// table shows one block per section (crypto, banks, ...).
func (wb *Workbook) groupSections() {
	if len(wb.Rows) < 2 {
		return
	}
	idx := map[string]int{}
	var order []string
	for _, as := range wb.Rows {
		if _, ok := idx[as.Section]; !ok {
			idx[as.Section] = len(order)
			order = append(order, as.Section)
		}
	}
	groups := make([][]Row, len(order))
	for _, as := range wb.Rows {
		i := idx[as.Section]
		groups[i] = append(groups[i], as)
	}
	out := make([]Row, 0, len(wb.Rows))
	for _, g := range groups {
		out = append(out, g...)
	}
	wb.Rows = out
}

// saveWorkbook writes workbook.json atomically, keeping the previous copy in
// workbook.json.prev.
func saveWorkbook(dir string, wb *Workbook) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	path := filepath.Join(dir, workbookFile)
	if _, err := os.Stat(path); err == nil {
		_ = os.Rename(path, path+".prev") // *.prev is gitignored
	}
	data, err := json.MarshalIndent(wb, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
