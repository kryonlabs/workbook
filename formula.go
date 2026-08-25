package main

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"unicode"
)

// stripExpr trims a units input, dropping one leading "=" so spreadsheet-style
// formulas ("=0.1+0.25") work everywhere units are typed or stored.
func stripExpr(s string) string {
	s = strings.TrimSpace(s)
	return strings.TrimPrefix(s, "=")
}

// evalUnits parses a units value: a plain number or an arithmetic expression
// over numbers (+ - * / and parentheses, unary +/-), with an optional leading
// "=". Milestones can thus be accumulated as formulas, e.g. "=0.1+0.25".
func evalUnits(s string) (float64, error) {
	s = stripExpr(s)
	if s == "" {
		return 0, fmt.Errorf("empty value")
	}
	p := &exprParser{src: s}
	v, err := p.parse()
	if err != nil {
		return 0, err
	}
	p.ws()
	if p.pos != len(p.src) {
		return 0, fmt.Errorf("unexpected %q", p.src[p.pos:])
	}
	f, _ := v.Float64()
	return f, nil
}

// exprParser is a small recursive-descent parser: parse = term (± term)*,
// term = factor (*/ factor)*, factor = (± factor) | number | ( parse ).
type exprParser struct {
	src string
	pos int
}

func (p *exprParser) ws() {
	for p.pos < len(p.src) && unicode.IsSpace(rune(p.src[p.pos])) {
		p.pos++
	}
}

func (p *exprParser) parse() (*big.Rat, error) {
	v, err := p.term()
	if err != nil {
		return nil, err
	}
	for {
		p.ws()
		if p.pos >= len(p.src) || (p.src[p.pos] != '+' && p.src[p.pos] != '-') {
			return v, nil
		}
		op := p.src[p.pos]
		p.pos++
		w, err := p.term()
		if err != nil {
			return nil, err
		}
		if op == '+' {
			v.Add(v, w)
		} else {
			v.Sub(v, w)
		}
	}
}

func (p *exprParser) term() (*big.Rat, error) {
	v, err := p.factor()
	if err != nil {
		return nil, err
	}
	for {
		p.ws()
		if p.pos >= len(p.src) || (p.src[p.pos] != '*' && p.src[p.pos] != '/') {
			return v, nil
		}
		op := p.src[p.pos]
		p.pos++
		w, err := p.factor()
		if err != nil {
			return nil, err
		}
		if op == '*' {
			v.Mul(v, w)
		} else {
			if w.Sign() == 0 {
				return nil, fmt.Errorf("division by zero")
			}
			v.Quo(v, w)
		}
	}
}

func (p *exprParser) factor() (*big.Rat, error) {
	p.ws()
	if p.pos >= len(p.src) {
		return nil, fmt.Errorf("unexpected end of formula")
	}
	switch p.src[p.pos] {
	case '+':
		p.pos++
		return p.factor()
	case '-':
		p.pos++
		v, err := p.factor()
		if err != nil {
			return nil, err
		}
		return v.Neg(v), nil
	case '(':
		p.pos++
		v, err := p.parse()
		if err != nil {
			return nil, err
		}
		p.ws()
		if p.pos >= len(p.src) || p.src[p.pos] != ')' {
			return nil, fmt.Errorf("missing )")
		}
		p.pos++
		return v, nil
	}
	start := p.pos
	for p.pos < len(p.src) && strings.IndexByte("0123456789.", p.src[p.pos]) >= 0 {
		p.pos++
	}
	if p.pos < len(p.src) && (p.src[p.pos] == 'e' || p.src[p.pos] == 'E') { // 1e-3
		q := p.pos + 1
		if q < len(p.src) && (p.src[q] == '+' || p.src[q] == '-') {
			q++
		}
		if q < len(p.src) && strings.IndexByte("0123456789", p.src[q]) >= 0 {
			p.pos = q
			for p.pos < len(p.src) && strings.IndexByte("0123456789", p.src[p.pos]) >= 0 {
				p.pos++
			}
		}
	}
	if p.pos == start {
		return nil, fmt.Errorf("expected a number at %d", start)
	}
	number := p.src[start:p.pos]
	if _, err := strconv.ParseFloat(number, 64); err != nil {
		return nil, err
	}
	v, ok := new(big.Rat).SetString(number)
	if !ok {
		// big.Rat does not accept exponent notation directly.
		f, _, err := big.ParseFloat(number, 10, 256, big.ToNearestEven)
		if err != nil {
			return nil, err
		}
		v, _ = f.Rat(nil)
	}
	return v, nil
}
