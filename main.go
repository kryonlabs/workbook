// Command workbook is a native Kryon workbook.
//
// The geld profile adds finance formulas and rate fetching on top of the
// generic workbook.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"
)

// app holds the state shared between the fetch goroutine, the render loop and
// the headless -update path.
type app struct {
	dir     string
	profile string
	wb      *Workbook

	prev  map[string]float64 // rates before the startup fetch (for deltas)
	delta map[string]float64 // (new-old)/old per id
	stale map[string]bool    // id -> startup fetch did not confirm the rate

	fetching bool
	resCh    chan refreshResult
	last     time.Time // last completed startup fetch

	err  string
	info string
}

type refreshResult struct {
	rates map[string]float64
	errs  []string
}

func main() {
	profile := flag.String("profile", defaultProfile(os.Args[0]), "workbook profile: workbook or geld")
	dir := flag.String("dir", "", "data directory (default: profile env, cwd workbook.kry, then user data dir)")
	update := flag.Bool("update", false, "update profile data headlessly, print the table and exit")
	cli := flag.Bool("cli", false, "interactive terminal mode (no window)")
	flag.Parse()

	p := normalizedProfile(*profile)
	d := *dir
	if d == "" {
		var err error
		if d, err = defaultDataDir(p); err != nil {
			fmt.Fprintln(os.Stderr, p+":", err)
			os.Exit(1)
		}
	}
	wb, from, err := loadWorkbook(d)
	if err != nil {
		fmt.Fprintln(os.Stderr, p+":", err)
		os.Exit(1)
	}
	a := &app{
		dir:     d,
		profile: p,
		wb:      wb,
		prev:    map[string]float64{},
		delta:   map[string]float64{},
		stale:   map[string]bool{},
		resCh:   make(chan refreshResult, 1),
		last:    time.Now(),
	}
	a.ensureProfileCells()
	if from != "" {
		a.info = fmt.Sprintf("migrated %d rows from %s", len(wb.Rows), from)
	}
	switch {
	case *update:
		runHeadless(a)
	case *cli:
		runCLI(a)
	default:
		runGUI(a)
	}
}

// defaultDataDir keeps local development convenient while installed builds use
// a stable user-owned data directory instead of writing beside the executable.
func defaultDataDir(profile string) (string, error) {
	if profile == profileGeld {
		if d := os.Getenv("GELD_DIR"); d != "" {
			return d, nil
		}
	} else if profile != profileWorkbook {
		if d := os.Getenv(profileDirEnv(profile)); d != "" {
			return d, nil
		}
	} else {
		if d := os.Getenv("WORKBOOK_DIR"); d != "" {
			return d, nil
		}
		if d := os.Getenv("CELL_DIR"); d != "" {
			return d, nil
		}
	}
	if cwd, err := os.Getwd(); err == nil {
		if _, err := os.Stat(filepath.Join(cwd, workbookFile)); err == nil {
			return cwd, nil
		}
		if _, err := os.Stat(filepath.Join(cwd, legacyWorkbookFile)); err == nil {
			return cwd, nil
		}
	}
	return userDataDir(profile)
}

func profileDirEnv(profile string) string {
	name := strings.ToUpper(strings.Map(func(r rune) rune {
		if r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
			return r
		}
		if r >= 'a' && r <= 'z' {
			return r - 'a' + 'A'
		}
		return '_'
	}, profile))
	return name + "_DIR"
}

func userDataDir(profile string) (string, error) {
	name := profile
	if name == "" || name == profileCell {
		name = profileWorkbook
	}
	if d := os.Getenv("XDG_DATA_HOME"); d != "" {
		return filepath.Join(d, name), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share", name), nil
}

// exeDir is retained for older callers/tests that need the resolved binary dir.
func exeDir() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	return filepath.Dir(exe), nil
}

// runHeadless fetches rates without a window, prints the table with totals
// and exits.
func runHeadless(a *app) {
	if a.info != "" {
		fmt.Println(a.commandName()+":", a.info)
	}
	errs := []string(nil)
	if a.isGeldProfile() {
		rates, fetchErrs := fetchRates(a.wb.Rows)
		a.apply(refreshResult{rates, fetchErrs})
		errs = fetchErrs
	} else {
		a.save("")
	}
	printTable(a)
	for _, e := range errs {
		fmt.Fprintln(os.Stderr, a.commandName()+":", e)
	}
	fmt.Printf("updated %s · saved %s\n",
		a.wb.Updated.Format("2006-01-02 15:04"), filepath.Join(a.dir, workbookFile))
	if a.isGeldProfile() && len(a.wb.Rates) == 0 {
		os.Exit(1)
	}
}

// printTable renders the workbook with section headers and totals, the same
// table the gui shows but as tab-separated columns for the terminal.
func printTable(a *app) {
	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "label\tcoin\tunits\trate\tusd\tΔusd")
	for i, as := range a.wb.Rows {
		rate, rateOK := a.cellNumber(i, 4, map[string]bool{})
		units := fmtUnits(as.Units)
		if as.Expr != "" {
			units += " (=" + as.Expr + ")"
		}
		usd, dusd := "-", "-"
		if v, ok := a.cellNumber(i, 6, map[string]bool{}); ok {
			usd = commaf(v)
		}
		if v, ok := a.cellNumber(i, 7, map[string]bool{}); ok {
			dusd = signed(commaf(v))
		}
		rateText := "-"
		if rateOK {
			rateText = fmtRate(rate)
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			as.Label, coinSymbol(as.ID), units, rateText,
			usd, dusd)
	}
	tusd, tdusd, hasPrev := a.totals()
	pend, dpend, phas := a.pendingTotals()
	dusd := "-"
	if hasPrev || phas {
		dusd = signed(commaf(tdusd + dpend))
	}
	fmt.Fprintf(w, "total + pending\t\t\t\t%s\t%s\n", commaf(tusd+pend), dusd)
	dusd = "-"
	if hasPrev {
		dusd = signed(commaf(tdusd))
	}
	fmt.Fprintf(w, "total\t\t\t\t%s\t%s\n", commaf(tusd), dusd)
	w.Flush()
}

// startRefresh kicks off the one startup background fetch.
func (a *app) startRefresh() {
	if a.fetching {
		return
	}
	a.fetching = true
	a.err = ""
	rows := append([]Row(nil), a.wb.Rows...)
	go func() {
		rates, errs := fetchRates(rows)
		a.resCh <- refreshResult{rates, errs}
	}()
}

// pollRefresh applies the startup background fetch, if finished.
func (a *app) pollRefresh() {
	select {
	case res := <-a.resCh:
		a.apply(res)
	default:
	}
}

// apply merges fresh startup rates into the workbook: ids an api did not return keep
// their last known rate (marked stale), deltas are computed against the
// previous ones and everything is saved.
func (a *app) apply(res refreshResult) {
	a.fetching = false
	a.last = time.Now()
	a.err = strings.Join(res.errs, "; ")
	if strings.HasPrefix(a.info, "migrated") {
		a.info = ""
	}
	prev := map[string]float64{}
	for id := range a.wb.Rates {
		prev[id] = a.wb.Rates[id]
		a.stale[id] = true
	}
	for id, r := range res.rates {
		a.wb.Rates[id] = r
		a.stale[id] = false
	}
	a.prev = prev
	for id, cur := range a.wb.Rates {
		a.delta[id] = 0
		if p, ok := prev[id]; ok && p != 0 {
			a.delta[id] = (cur - p) / p
		}
	}
	for i := range a.wb.Rows {
		as := a.wb.Rows[i]
		if r, ok := res.rates[as.ID]; ok && as.RateExpr == "" {
			as.Rate = &r
		} else if as.Rate == nil && as.RateExpr == "" {
			if r := a.wb.Rates[as.ID]; r > 0 {
				as.Rate = &r
			}
		}
		if d, ok := a.delta[as.ID]; ok && as.PctExpr == "" {
			as.Pct = &d
		}
		a.wb.Rows[i] = as
	}
	a.ensureProfileCells()
	a.wb.Updated = time.Now()
	a.save("")
}

// save persists the workbook, showing msg (or the error) in the status line.
func (a *app) save(msg string) {
	a.ensureProfileCells()
	if err := saveWorkbook(a.dir, a.wb); err != nil {
		a.err = err.Error()
		return
	}
	if msg != "" {
		a.info = msg
	}
}

// totals sums the usd value and the usd change since the startup fetch,
// excluding the pending section (milestones not received yet) — totals are
// always usd; per-row eur is a convenience column only.
func (a *app) totals() (usd, dusd float64, hasPrev bool) {
	currentSection := ""
	for row, as := range a.wb.Rows {
		if section := strings.ToLower(strings.TrimSpace(as.Section)); section != "" {
			currentSection = section
		}
		if currentSection == "pending" {
			continue
		}
		if v, ok := a.cellNumber(row, 6, map[string]bool{}); ok {
			usd += v
		}
		if v, ok := a.cellNumber(row, 7, map[string]bool{}); ok {
			dusd += v
			hasPrev = true
		}
	}
	return
}

// pendingTotals is the same sums restricted to the pending section, for the
// "total + pending" line above the grand total.
func (a *app) pendingTotals() (usd, dusd float64, hasPrev bool) {
	for row, as := range a.wb.Rows {
		if as.Section != "pending" {
			continue
		}
		if v, ok := a.cellNumber(row, 6, map[string]bool{}); ok {
			usd += v
		}
		if v, ok := a.cellNumber(row, 7, map[string]bool{}); ok {
			dusd += v
			hasPrev = true
		}
	}
	return
}
