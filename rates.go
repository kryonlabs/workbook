package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Rates are fetched without any api key: coingecko for crypto,
// exchangerate-api (frankfurter as fallback) for fiat.
var httpClient = &http.Client{Timeout: 8 * time.Second}

func getJSON(url string, out any) error {
	resp, err := httpClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// fetchRates returns fresh usd-per-unit rates for every row it can. Ids an
// api does not know simply stay absent — callers keep the last known rate.
func fetchRates(rows []Row) (map[string]float64, []string) {
	var errs []string
	out := map[string]float64{}

	var ids []string
	seen := map[string]bool{}
	fiats := map[string]bool{"EUR": true} // always needed for the eur column
	for _, as := range rows {
		if isFiat(as.ID) {
			fiats[strings.ToUpper(as.ID)] = true
			continue
		}
		if as.ID != "" && !seen[as.ID] {
			seen[as.ID] = true
			ids = append(ids, as.ID)
		}
	}

	if len(ids) > 0 {
		var resp map[string]struct {
			USD float64 `json:"usd"`
		}
		url := "https://api.coingecko.com/api/v3/simple/price?ids=" +
			strings.Join(ids, ",") + "&vs_currencies=usd"
		if err := getJSON(url, &resp); err != nil {
			errs = append(errs, "coingecko: "+err.Error())
		} else {
			for id, p := range resp {
				if p.USD > 0 {
					out[id] = p.USD
				}
			}
		}
	}

	m, err := forexTable()
	if err != nil {
		errs = append(errs, "fiat: "+err.Error())
	} else {
		for code := range fiats {
			if code == "USD" {
				out["USD"] = 1
				continue
			}
			if r, ok := m[code]; ok && r > 0 {
				out[code] = 1 / r // per-USD table -> usd per unit
			}
		}
	}
	return out, errs
}

// forexTable returns per-1-USD fiat rates from a free, keyless source.
func forexTable() (map[string]float64, error) {
	var erapi struct {
		Result string             `json:"result"`
		Rates  map[string]float64 `json:"rates"`
	}
	if err := getJSON("https://open.er-api.com/v6/latest/USD", &erapi); err == nil &&
		erapi.Result == "success" && len(erapi.Rates) > 0 {
		return erapi.Rates, nil
	}
	var frank struct {
		Rates map[string]float64 `json:"rates"`
	}
	if err := getJSON("https://api.frankfurter.app/latest?from=USD", &frank); err == nil &&
		len(frank.Rates) > 0 {
		return frank.Rates, nil
	}
	return nil, fmt.Errorf("no fiat source reachable")
}
