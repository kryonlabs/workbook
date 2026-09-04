package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// runCLI is the terminal mode: a tiny shell over the same workbook.kry the gui
// edits — list/add/set/rm — for when a window is not at hand.

const cliHelp = `commands:
  list                     show the table
  add SECTION LABEL COIN UNITS   new row (UNITS may be a formula like 1.2+0.5)
  set LABEL field value    field: section label coin units
  rm LABEL                 delete the row (first match)
  quit                     save already happened per command; just exit`

func runCLI(a *app) {
	if a.info != "" {
		fmt.Println(a.commandName()+":", a.info)
	}
	if a.isGeldProfile() {
		rates, errs := fetchRates(a.wb.Rows)
		a.apply(refreshResult{rates, errs})
		for _, e := range errs {
			fmt.Println(a.commandName()+":", e)
		}
	}
	fmt.Println(a.commandName(), "cli —", strconv.Itoa(len(a.wb.Rows)), "rows · 'help' for commands")
	printTable(a)

	sc := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print(a.commandName() + "> ")
		if !sc.Scan() {
			fmt.Println()
			return
		}
		fields := strings.Fields(sc.Text())
		if len(fields) == 0 {
			continue
		}
		switch fields[0] {
		case "help", "h":
			fmt.Print(cliHelp)
		case "list", "ls":
			printTable(a)
		case "add":
			if len(fields) < 5 {
				fmt.Println("usage: add SECTION LABEL COIN UNITS")
				continue
			}
			units, expr := parseUnits(strings.Join(fields[4:], " "))
			if units == 0 && expr == "" {
				fmt.Println(a.commandName() + ": bad units")
				continue
			}
			a.wb.Rows = append(a.wb.Rows, Row{
				Section: strings.ToLower(fields[1]),
				Label:   fields[2],
				ID:      normID(fields[3]),
				Units:   units,
				Expr:    expr,
			})
			a.save("row added")
			printTable(a)
		case "set":
			if len(fields) < 4 {
				fmt.Println("usage: set LABEL field value")
				continue
			}
			i := findByLabel(a, fields[1])
			if i < 0 {
				continue
			}
			value := strings.Join(fields[3:], " ")
			as := a.wb.Rows[i]
			switch fields[2] {
			case "section":
				as.Section = strings.ToLower(value)
			case "label":
				as.Label = value
			case "coin":
				if value == "" {
					fmt.Println(a.commandName() + ": coin id cannot be empty — use rm")
					continue
				}
				as.ID = normID(value)
			case "units":
				units, expr := parseUnits(value)
				if units == 0 && expr == "" {
					fmt.Println(a.commandName() + ": bad units")
					continue
				}
				as.Units, as.Expr = units, expr
			default:
				fmt.Println(a.commandName() + ": unknown field (section label coin units)")
				continue
			}
			a.wb.Rows[i] = as
			a.save(fields[2] + " saved")
			printTable(a)
		case "rm", "del":
			if len(fields) < 2 {
				fmt.Println("usage: rm LABEL")
				continue
			}
			if i := findByLabel(a, fields[1]); i >= 0 {
				id := a.wb.Rows[i].ID
				a.rewriteWorkbookFormulaRows(i, false)
				a.shiftCellFormatsForDelete(i)
				a.shiftCellValuesForRowDelete(i)
				a.wb.Rows = append(a.wb.Rows[:i], a.wb.Rows[i+1:]...)
				a.save("deleted " + id)
				printTable(a)
			}
		case "quit", "q", "exit":
			return
		default:
			fmt.Println(a.commandName() + ": unknown command — 'help'")
		}
	}
}

// findByLabel matches a row by (case-insensitive) label, first hit wins.
func findByLabel(a *app, label string) int {
	for i := range a.wb.Rows {
		if strings.EqualFold(a.wb.Rows[i].Label, label) {
			return i
		}
	}
	fmt.Println(a.commandName()+": no row labeled", label)
	return -1
}

// parseUnits evaluates a units value (number or formula), mirroring what the
// gui's cell editor stores: formula inputs keep their expression.
func parseUnits(s string) (units float64, expr string) {
	formula := strings.HasPrefix(strings.TrimSpace(s), "=")
	s = stripExpr(s)
	v, err := evalUnits(s)
	if err != nil {
		return 0, ""
	}
	if _, e := strconv.ParseFloat(s, 64); e != nil || formula {
		expr = s
	}
	return v, expr
}
