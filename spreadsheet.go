package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	cellRefRE  = regexp.MustCompile(`\b([A-H])([0-9]+)\b`)
	sumRangeRE = regexp.MustCompile(`(?i)SUM\(([A-H])([0-9]+):([A-H])([0-9]+)\)`)
)

type formulaRefGroup struct {
	Cells []formulaCellRef
}

type formulaCellRef struct {
	Row int
	Col int
}

func dataColLetter(col int) string {
	if col < 0 || col > 7 {
		return ""
	}
	return string(rune('A' + col))
}

func defaultRowFormula(row, col int) string {
	n := strconv.Itoa(row + 1)
	switch col {
	case 6:
		return "D" + n + "*E" + n
	}
	return ""
}

func shiftFormulaRows(formula string, delta int) string {
	if delta == 0 {
		return formula
	}
	return cellRefRE.ReplaceAllStringFunc(formula, func(ref string) string {
		col := ref[:1]
		row, err := strconv.Atoi(ref[1:])
		if err != nil {
			return ref
		}
		row += delta
		if row < 1 {
			row = 1
		}
		return col + strconv.Itoa(row)
	})
}

func sumRangeTerm(col, startRow, endRow int) string {
	letter := dataColLetter(col)
	if letter == "" || startRow <= 0 || endRow < startRow {
		return "0"
	}
	return "SUM(" + letter + strconv.Itoa(startRow) + ":" + letter + strconv.Itoa(endRow) + ")"
}

func (a *app) pendingSumTerms(col int) []string {
	var terms []string
	currentSection := ""
	start := -1

	closePending := func(endRow int) {
		if start >= 0 {
			terms = append(terms, sumRangeTerm(col, start, endRow))
			start = -1
		}
	}

	for i, as := range a.wb.Rows {
		if section := strings.ToLower(strings.TrimSpace(as.Section)); section != "" && section != currentSection {
			if currentSection == "pending" {
				closePending(i)
			}
			currentSection = section
			if currentSection == "pending" {
				start = i + 1
			}
		}
	}
	if currentSection == "pending" {
		closePending(len(a.wb.Rows))
	}
	return terms
}

func (a *app) defaultTotalFormula(kind, col int) string {
	all := sumRangeTerm(col, 1, len(a.wb.Rows))
	if kind == dispTotalPending {
		return "=" + all
	}
	expr := all
	for _, term := range a.pendingSumTerms(col) {
		expr += "-" + term
	}
	return "=" + expr
}

func (a *app) defaultTotalCell(kind, col int) string {
	switch kind {
	case dispTotalPending:
		switch col {
		case 1:
			return "total + pending"
		case 6:
			return a.defaultTotalFormula(kind, col)
		case 7:
			return a.defaultTotalFormula(kind, col)
		}
	case dispTotal:
		switch col {
		case 1:
			return "total"
		case 6:
			return a.defaultTotalFormula(kind, col)
		case 7:
			return a.defaultTotalFormula(kind, col)
		}
	}
	return ""
}

func (a *app) totalCellEditText(kind, col int) string {
	cells := a.totalCells(kind)
	if col >= 0 && col < len(cells) && cells[col] != "" {
		return cells[col]
	}
	return a.defaultTotalCell(kind, col)
}

func (a *app) setTotalCell(kind, col int, raw string) {
	raw = strings.TrimSpace(raw)
	cells := append([]string(nil), a.totalCells(kind)...)
	if len(cells) < 8 {
		next := make([]string, 8)
		copy(next, cells)
		cells = next
	}
	if col >= 0 && col < len(cells) {
		cells[col] = raw
	}
	switch kind {
	case dispTotalPending:
		a.wb.TotalPendingCells = trimTrailingEmpty(cells)
	case dispTotal:
		a.wb.TotalCells = trimTrailingEmpty(cells)
	}
}

func shouldFillTotalCell(kind, col int, raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return true
	}
	switch col {
	case 1:
		return raw == defaultTotalLabel(kind)
	case 6, 7:
		return isDefaultTotalFormulaShape(kind, col, raw)
	default:
		return false
	}
}

func defaultTotalLabel(kind int) string {
	switch kind {
	case dispTotalPending:
		return "total + pending"
	case dispTotal:
		return "total"
	default:
		return ""
	}
}

func isDefaultTotalFormulaShape(kind, col int, raw string) bool {
	letter := dataColLetter(col)
	if letter == "" {
		return false
	}
	compact := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(raw), " ", ""))
	allRange := `SUM\(` + letter + `1:` + letter + `[0-9]+\)`
	var pattern string
	switch kind {
	case dispTotalPending:
		pattern = `^=` + allRange + `$`
	case dispTotal:
		pattern = `^=` + allRange + `(-SUM\(` + letter + `[0-9]+:` + letter + `[0-9]+\))*$`
	default:
		return false
	}
	return regexp.MustCompile(pattern).MatchString(compact)
}

func (a *app) totalCells(kind int) []string {
	switch kind {
	case dispTotalPending:
		return a.wb.TotalPendingCells
	case dispTotal:
		return a.wb.TotalCells
	}
	return nil
}

func trimTrailingEmpty(cells []string) []string {
	for len(cells) > 0 && cells[len(cells)-1] == "" {
		cells = cells[:len(cells)-1]
	}
	return cells
}

func (a *app) totalCellDisplayText(kind, col int) string {
	raw := a.totalCellEditText(kind, col)
	if strings.HasPrefix(strings.TrimSpace(raw), "=") {
		v, err := a.evalTableFormula(raw, -1, map[string]bool{})
		if err != nil {
			return "-"
		}
		if col == 5 {
			return fmtPct(v)
		}
		if col == 7 {
			return signed(commaf(v))
		}
		return commaf(v)
	}
	return raw
}

func (a *app) evalTableFormula(expr string, curRow int, seen map[string]bool) (float64, error) {
	s := stripExpr(expr)
	if s == "" {
		return 0, nil
	}
	upper := strings.ToUpper(strings.ReplaceAll(s, " ", ""))
	if strings.HasPrefix(upper, "SUM(") && strings.HasSuffix(upper, ")") && strings.Count(upper, "SUM(") == 1 {
		return a.evalSumRange(upper[4:len(upper)-1], seen)
	}
	var replaceErr error
	replaced := sumRangeRE.ReplaceAllStringFunc(s, func(call string) string {
		match := sumRangeRE.FindStringSubmatch(call)
		if len(match) != 5 {
			return "0"
		}
		spec := strings.ToUpper(match[1]) + match[2] + ":" + strings.ToUpper(match[3]) + match[4]
		v, err := a.evalSumRange(spec, seen)
		if err != nil && replaceErr == nil {
			replaceErr = err
			return "0"
		}
		return strconv.FormatFloat(v, 'f', -1, 64)
	})
	if replaceErr != nil {
		return 0, replaceErr
	}
	replaced = cellRefRE.ReplaceAllStringFunc(replaced, func(ref string) string {
		col := int(ref[0] - 'A')
		row, err := strconv.Atoi(ref[1:])
		if err != nil || row <= 0 {
			return "0"
		}
		v, ok := a.cellNumber(row-1, col, seen)
		if !ok {
			return "0"
		}
		return strconv.FormatFloat(v, 'f', -1, 64)
	})
	return evalUnits(replaced)
}

func (a *app) evalSumRange(spec string, seen map[string]bool) (float64, error) {
	parts := strings.Split(spec, ":")
	if len(parts) == 1 && len(parts[0]) >= 1 {
		col := int(parts[0][0] - 'A')
		if col < 0 || col > 7 {
			return 0, fmt.Errorf("bad range %s", spec)
		}
		var sum float64
		for row := range a.wb.Rows {
			v, _ := a.cellNumber(row, col, seen)
			sum += v
		}
		return sum, nil
	}
	if len(parts) != 2 || len(parts[0]) < 2 || len(parts[1]) < 2 {
		return 0, fmt.Errorf("bad range %s", spec)
	}
	startCol := int(parts[0][0] - 'A')
	endCol := int(parts[1][0] - 'A')
	startRow, err1 := strconv.Atoi(parts[0][1:])
	endRow, err2 := strconv.Atoi(parts[1][1:])
	if err1 != nil || err2 != nil || startCol != endCol || startCol < 0 || startCol > 7 {
		return 0, fmt.Errorf("bad range %s", spec)
	}
	if startRow > endRow {
		startRow, endRow = endRow, startRow
	}
	var sum float64
	for row := startRow - 1; row <= endRow-1 && row < len(a.wb.Rows); row++ {
		if row < 0 {
			continue
		}
		v, _ := a.cellNumber(row, startCol, seen)
		sum += v
	}
	return sum, nil
}

func (a *app) cellNumber(row, col int, seen map[string]bool) (float64, bool) {
	if row < 0 || row >= len(a.wb.Rows) {
		return 0, false
	}
	key := strconv.Itoa(row) + ":" + strconv.Itoa(col)
	if seen[key] {
		return 0, false
	}
	seen[key] = true
	defer delete(seen, key)
	as := a.wb.Rows[row]
	eval := func(expr string) (float64, bool) {
		v, err := a.evalTableFormula(expr, row, seen)
		return v, err == nil
	}
	switch col {
	case 3:
		if as.Expr != "" {
			return eval(as.Expr)
		}
		return as.Units, true
	case 4:
		if as.RateExpr != "" {
			return eval(as.RateExpr)
		}
		if as.Rate != nil {
			return *as.Rate, true
		}
		return 0, false
	case 5:
		if as.PctExpr != "" {
			return eval(as.PctExpr)
		}
		if as.Pct != nil {
			return *as.Pct, true
		}
		return 0, false
	case 6:
		if as.USDExpr != "" {
			return eval(as.USDExpr)
		}
		if as.USD != nil {
			return *as.USD, true
		}
		return 0, false
	case 7:
		if as.DUSDExpr != "" {
			return eval(as.DUSDExpr)
		}
		if as.DUSD != nil {
			return *as.DUSD, true
		}
		return 0, false
	}
	return 0, false
}

func optionalFloatText(ptr *float64) string {
	if ptr == nil {
		return ""
	}
	return strconv.FormatFloat(*ptr, 'f', -1, 64)
}

func (a *app) formulaRefGroups(expr string) []formulaRefGroup {
	s := stripExpr(expr)
	if s == "" {
		return nil
	}
	var groups []formulaRefGroup
	consumed := map[string]bool{}
	for _, match := range sumRangeRE.FindAllStringSubmatch(s, -1) {
		startCol := int(strings.ToUpper(match[1])[0] - 'A')
		endCol := int(strings.ToUpper(match[3])[0] - 'A')
		startRow, err1 := strconv.Atoi(match[2])
		endRow, err2 := strconv.Atoi(match[4])
		if err1 != nil || err2 != nil || startCol != endCol || startCol < 0 || startCol > 7 {
			continue
		}
		if startRow > endRow {
			startRow, endRow = endRow, startRow
		}
		var cells []formulaCellRef
		for row := startRow - 1; row <= endRow-1 && row < len(a.wb.Rows); row++ {
			if row < 0 {
				continue
			}
			cells = append(cells, formulaCellRef{Row: row, Col: startCol})
			consumed[strconv.Itoa(row)+":"+strconv.Itoa(startCol)] = true
		}
		if len(cells) > 0 {
			groups = append(groups, formulaRefGroup{Cells: cells})
		}
	}
	for _, match := range cellRefRE.FindAllStringSubmatch(s, -1) {
		col := int(match[1][0] - 'A')
		row, err := strconv.Atoi(match[2])
		if err != nil || row <= 0 || row > len(a.wb.Rows) || col < 0 || col > 7 {
			continue
		}
		key := strconv.Itoa(row-1) + ":" + strconv.Itoa(col)
		if consumed[key] {
			continue
		}
		groups = append(groups, formulaRefGroup{Cells: []formulaCellRef{{Row: row - 1, Col: col}}})
	}
	return groups
}
