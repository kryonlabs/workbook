package main

import (
	"regexp"
	"strconv"
	"strings"
)

var sumRangeRewriteRE = regexp.MustCompile(`(?i)SUM\(([A-H])([0-9]+):([A-H])([0-9]+)\)`)

// rewriteFormulaRows applies spreadsheet structural-edit rules. References to
// rows which move are moved with them; a direct reference to a deleted row is
// invalid, while a range containing that row contracts around it.
func rewriteFormulaRows(formula string, pos int, insert bool) string {
	if !strings.HasPrefix(strings.TrimSpace(formula), "=") && formula == "" {
		return formula
	}
	placeholders := []string{}
	formula = sumRangeRewriteRE.ReplaceAllStringFunc(formula, func(call string) string {
		m := sumRangeRewriteRE.FindStringSubmatch(call)
		a, _ := strconv.Atoi(m[2])
		b, _ := strconv.Atoi(m[4])
		edit := pos + 1
		if insert {
			if edit <= a {
				a++
				b++
			} else if edit <= b {
				b++
			}
		} else {
			if a == b && a == edit {
				call = "SUM(#REF!)"
			} else if edit < a {
				a--
				b--
			} else if edit >= a && edit <= b {
				b--
			}
			if call != "SUM(#REF!)" {
				call = "SUM(" + strings.ToUpper(m[1]) + strconv.Itoa(a) + ":" + strings.ToUpper(m[3]) + strconv.Itoa(b) + ")"
			}
		}
		if insert {
			call = "SUM(" + strings.ToUpper(m[1]) + strconv.Itoa(a) + ":" + strings.ToUpper(m[3]) + strconv.Itoa(b) + ")"
		}
		marker := "§" + strconv.Itoa(len(placeholders)) + "§"
		placeholders = append(placeholders, call)
		return marker
	})
	formula = cellRefRE.ReplaceAllStringFunc(formula, func(ref string) string {
		m := cellRefRE.FindStringSubmatch(ref)
		row, _ := strconv.Atoi(m[2])
		edit := pos + 1
		if insert && row >= edit {
			row++
		} else if !insert {
			if row == edit {
				return "#REF!"
			}
			if row > edit {
				row--
			}
		}
		return strings.ToUpper(m[1]) + strconv.Itoa(row)
	})
	for i, value := range placeholders {
		formula = strings.ReplaceAll(formula, "§"+strconv.Itoa(i)+"§", value)
	}
	return formula
}

func rewriteFormulaColumns(formula string, pos int, insert bool) string {
	return cellRefRE.ReplaceAllStringFunc(formula, func(ref string) string {
		m := cellRefRE.FindStringSubmatch(ref)
		col := int(strings.ToUpper(m[1])[0] - 'A')
		if insert && col >= pos {
			col++
		} else if !insert {
			if col == pos {
				return "#REF!"
			}
			if col > pos {
				col--
			}
		}
		if col < 0 || col > 7 {
			return "#REF!"
		}
		return string(rune('A'+col)) + m[2]
	})
}

func (a *app) rewriteWorkbookFormulaRows(pos int, insert bool) {
	for i := range a.wb.Rows {
		r := &a.wb.Rows[i]
		for _, expr := range []*string{&r.Expr, &r.RateExpr, &r.PctExpr, &r.USDExpr, &r.DUSDExpr} {
			if *expr != "" {
				*expr = strings.TrimPrefix(rewriteFormulaRows("="+*expr, pos, insert), "=")
			}
		}
	}
	a.rewriteExtraFormulas(func(s string) string { return rewriteFormulaRows(s, pos, insert) })
}

func (a *app) rewriteWorkbookFormulaColumns(pos int, insert bool) {
	for i := range a.wb.Rows {
		r := &a.wb.Rows[i]
		for _, expr := range []*string{&r.Expr, &r.RateExpr, &r.PctExpr, &r.USDExpr, &r.DUSDExpr} {
			if *expr != "" {
				*expr = strings.TrimPrefix(rewriteFormulaColumns("="+*expr, pos, insert), "=")
			}
		}
	}
	a.rewriteExtraFormulas(func(s string) string { return rewriteFormulaColumns(s, pos, insert) })
}

func (a *app) rewriteExtraFormulas(rewrite func(string) string) {
	for _, cells := range [][]string{a.wb.TotalPendingCells, a.wb.TotalCells} {
		for i := range cells {
			if strings.HasPrefix(strings.TrimSpace(cells[i]), "=") {
				cells[i] = rewrite(cells[i])
			}
		}
	}
	for key, raw := range a.wb.CellValues {
		if strings.HasPrefix(strings.TrimSpace(raw), "=") {
			a.wb.CellValues[key] = rewrite(raw)
		}
	}
}

func (a *app) shiftCellValuesForRowInsert(pos int) { a.shiftCellValueRows(pos, true) }
func (a *app) shiftCellValuesForRowDelete(pos int) { a.shiftCellValueRows(pos, false) }
func (a *app) shiftCellValueRows(pos int, insert bool) {
	next := map[string]string{}
	for key, value := range a.wb.CellValues {
		row, col, ok := splitCellFormatKey(key)
		if !ok {
			continue
		}
		if insert && row >= pos {
			row++
		} else if !insert {
			if row == pos {
				continue
			}
			if row > pos {
				row--
			}
		}
		next[cellFormatKey(row, col)] = value
	}
	a.wb.CellValues = next
}

func (a *app) mutateColumn(u *uiState, pos int, insert bool) {
	if pos < 0 || pos > 7 {
		return
	}
	values := make([][8]string, len(a.wb.Rows))
	for row := range a.wb.Rows {
		for col := 0; col < 8; col++ {
			values[row][col] = a.rowCellEditText(row, col)
		}
	}
	a.wb.CellValues = map[string]string{}
	for row := range a.wb.Rows {
		a.wb.Rows[row] = Row{}
		for col := 0; col < 8; col++ {
			source := col
			if insert {
				if col == pos {
					continue
				}
				if col > pos {
					source = col - 1
				}
			} else {
				if col >= pos {
					source = col + 1
				}
				if source > 7 {
					continue
				}
			}
			_ = a.setRowCell(row, col, values[row][source])
		}
	}
	shiftTotals := func(cells []string) []string {
		next := make([]string, 8)
		for col := 0; col < 8; col++ {
			source := col
			if insert {
				if col == pos {
					continue
				}
				if col > pos {
					source = col - 1
				}
			} else {
				if col >= pos {
					source = col + 1
				}
				if source > 7 {
					continue
				}
			}
			if source < len(cells) {
				next[col] = cells[source]
			}
		}
		return trimTrailingEmpty(next)
	}
	a.wb.TotalPendingCells = shiftTotals(a.wb.TotalPendingCells)
	a.wb.TotalCells = shiftTotals(a.wb.TotalCells)
	if insert {
		a.shiftCellFormatsForColumnInsert(pos)
	} else {
		a.shiftCellFormatsForColumnDelete(pos)
	}
	a.rewriteWorkbookFormulaColumns(pos, insert)
	a.ensureProfileCells()
	a.save(map[bool]string{true: "column inserted", false: "column deleted"}[insert])
	u.disp = a.dispRows()
	u.selRow = -1
	u.selCol = tableColFromDataCol(pos)
}
