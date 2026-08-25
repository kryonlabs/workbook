package main

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// commaf formats v with two decimals and thousands commas.
func commaf(v float64) string { return commafDec(v, 2) }

func commafDec(v float64, dec int) string {
	s := strconv.FormatFloat(math.Abs(v), 'f', dec, 64)
	intPart, frac := s, ""
	if i := strings.IndexByte(s, '.'); i >= 0 {
		intPart, frac = s[:i], s[i:]
	}
	var b strings.Builder
	if pre := len(intPart) % 3; pre > 0 {
		b.WriteString(intPart[:pre])
	}
	for i := len(intPart) % 3; i+3 <= len(intPart); i += 3 {
		if b.Len() > 0 {
			b.WriteByte(',')
		}
		b.WriteString(intPart[i : i+3])
	}
	out := b.String() + frac
	if v < 0 {
		out = "-" + out
	}
	return out
}

// signed adds an explicit + to delta values that are not negative.
func signed(v string) string {
	if !strings.HasPrefix(v, "-") {
		return "+" + v
	}
	return v
}

// fmtRate renders a usd rate; tiny ones (most shitcoins) go scientific.
func fmtRate(v float64) string {
	switch {
	case v == 0:
		return "-"
	case math.Abs(v) >= 1000:
		return commafDec(v, 0)
	case math.Abs(v) >= 1:
		return strconv.FormatFloat(v, 'f', 2, 64)
	case math.Abs(v) >= 0.0001:
		return strconv.FormatFloat(v, 'f', 4, 64)
	default:
		return strconv.FormatFloat(v, 'e', 2, 64)
	}
}

// fmtUnits renders a unit count, trimming trailing zeros (0.35, 3.7, 1,234).
func fmtUnits(u float64) string {
	if u == 0 {
		return "0"
	}
	if u == math.Trunc(u) && math.Abs(u) < 1e15 {
		return commafDec(u, 0)
	}
	dec := 4
	if math.Abs(u) < 1 {
		dec = 8
	}
	s := strconv.FormatFloat(u, 'f', dec, 64)
	if strings.IndexByte(s, '.') >= 0 {
		s = strings.TrimRight(s, "0")
		s = strings.TrimSuffix(s, ".")
	}
	return s
}

func fmtPct(d float64) string {
	if d == 0 {
		return "-"
	}
	return fmt.Sprintf("%+.2f%%", d*100)
}

// coinSymbols maps coingecko ids to ticker symbols for display; anything not
// listed (fiat codes, unknown ids) shows as-is.
var coinSymbols = map[string]string{
	"bitcoin":       "btc",
	"ethereum":      "eth",
	"litecoin":      "ltc",
	"monero":        "xmr",
	"cardano":       "ada",
	"cosmos":        "atom",
	"dogecoin":      "doge",
	"binancecoin":   "bnb",
	"tether":        "usdt",
	"usd-coin":      "usdc",
	"uniswap":       "uni",
	"chainlink":     "link",
	"solana":        "sol",
	"polkadot":      "dot",
	"matic-network": "matic",
	"shiba-inu":     "shib",
	"terra-luna":    "luna",
	"secret":        "scrt",
	"illuvium":      "ilv",
	"status":        "snt",
}

func coinSymbol(id string) string {
	if s, ok := coinSymbols[id]; ok {
		return s
	}
	return id
}
