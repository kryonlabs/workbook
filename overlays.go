package main

import (
	"fmt"

	rl "github.com/waozixyz/kryon/go/kryon"
)

func statusLine(a *app) (string, rl.Color) {
	switch {
	case a.fetching:
		return "fetching rates…", colAccent
	case a.err != "":
		return a.err, colRed
	default:
		return fmt.Sprintf("updated %s", a.wb.Updated.Format("15:04:05")), colDim
	}
}
