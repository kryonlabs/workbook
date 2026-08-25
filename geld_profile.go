package main

import (
	"regexp"
	"strconv"
	"strings"
)

func (a *app) ensureProfileCells() {
	if a.isGeldProfile() {
		a.ensureGeldProfileCells()
	}
}

func (a *app) ensureGeldProfileCells() {
	a.ensureGeldRowCells()
	a.ensureGeldTotalCells()
}

func (a *app) ensureGeldRowCells() {
	for i := range a.wb.Rows {
		as := a.wb.Rows[i]
		row := strconv.Itoa(i + 1)
		if as.Rate == nil && as.RateExpr == "" {
			if rate := a.wb.Rates[as.ID]; rate > 0 {
				as.Rate = &rate
			}
		}
		if as.USD == nil && shouldFillGeldRowFormula(6, i, as.USDExpr) {
			as.USDExpr = defaultRowFormula(i, 6)
		}
		if as.DUSD == nil && shouldFillGeldRowFormula(7, i, as.DUSDExpr) {
			if p, ok := a.prev[as.ID]; ok {
				as.DUSDExpr = "(E" + row + "-" + strconv.FormatFloat(p, 'f', -1, 64) + ")*D" + row
			}
		}
		a.wb.Rows[i] = as
	}
}

func shouldFillGeldRowFormula(col, row int, raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return true
	}
	switch col {
	case 6:
		return raw == defaultRowFormula(row, col)
	case 7:
		return defaultDUSDFormulaRE(row).MatchString(raw)
	}
	return false
}

func defaultDUSDFormulaRE(row int) *regexp.Regexp {
	n := strconv.Itoa(row + 1)
	return regexp.MustCompile(`^\(?E` + n + `-[0-9.+-]+\)?\*D` + n + `$`)
}

func (a *app) ensureGeldTotalCells() {
	for _, kind := range []int{dispTotalPending, dispTotal} {
		cells := append([]string(nil), a.totalCells(kind)...)
		if len(cells) < 8 {
			next := make([]string, 8)
			copy(next, cells)
			cells = next
		}
		for col := 0; col < 8; col++ {
			if !shouldFillTotalCell(kind, col, cells[col]) {
				continue
			}
			if def := a.defaultTotalCell(kind, col); def != "" {
				cells[col] = def
			}
		}
		switch kind {
		case dispTotalPending:
			a.wb.TotalPendingCells = trimTrailingEmpty(cells)
		case dispTotal:
			a.wb.TotalCells = trimTrailingEmpty(cells)
		}
	}
}
