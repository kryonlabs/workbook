# workbook

Native Kryon spreadsheet-style workbook editor. `workbook` is the generic base
app; profile commands such as `geld` open specialized workbook versions with
their own scripts, data directories, and defaults.

## build & run

    make              # build ./workbook with the clean native Go runtime
    make test         # Go tests plus native k2g/go-kryon smoke
    make run          # open ./workbook
    ./workbook        # generic workbook
    ./workbook -profile geld
    geld              # installed finance profile wrapper
    geld -cli         # finance profile terminal mode
    geld -update      # fetch rates, print the finance table, exit
    make install      # install ~/bin/workbook, ~/bin/cell, and profile commands

## editing

The desktop path is the table editor. It uses `go/kryon` directly with
`CGO_ENABLED=0`; the legacy cgo UI bridge is blocked by audits.

    up/down wheel     select / scroll
    left/right Tab    move across cells
    Enter F2          edit the selected cell
    type              replace the selected cell
    Delete            clear the selected cell
    right click       row/cell actions
    Esc               quit

Formula cells show their value in the grid and their formula in the formula
bar. Drag the small handle at the bottom-right of a formula cell to fill it
across rows with row references shifted.

The top menu and toolbar provide save, row insertion/deletion, and per-cell
text/background color formatting. Cells accept text, numbers, or formulas.
Right-click a cell for formatting and positive/negative conditional coloring;
right-click a row or column header to insert or delete it. Structural row and
column edits rewrite formula references to follow the cells that moved.
Formatting and free-form values are saved in `workbook.kry`.

## profiles

`workbook` is the generic workbook profile. It does not fetch rates.

`geld` is the finance profile. It fills the rate/value/change cells as regular
spreadsheet cells:

    D = units
    E = rate
    G = value, e.g. =D1*E1
    H = diff, e.g. =(E1-OLD_RATE)*D1

Rates are fetched once when the Geld profile starts. They are never fetched
again while the window stays open, so editing cannot be interrupted by network
refreshes. Restart `geld`, run `geld -update`, or open `geld -cli` to fetch a
new snapshot.

Profile commands are installed from `profiles/*.kry`. To add a custom workbook
version, add a manifest with a unique `name` and `command`; `make install`
generates the wrapper command that invokes `workbook -profile <name>`.

## data

Data loads from `-dir`, then the profile environment variable, then a
`workbook.kry` in the current directory, then the profile data dir. Existing
`workbook.json` data is renamed to `workbook.kry` automatically on first use.

    WORKBOOK_DIR      generic workbook override
    CELL_DIR          legacy alias for WORKBOOK_DIR
    GELD_DIR          geld finance workbook override
    ~/.local/share/workbook
    ~/.local/share/geld

`workbook.kry` holds private workbook data and is intentionally ignored by
git. `workbook.example.kry` is safe sample data for demos.

## rates

Geld rows use coingecko ids (`bitcoin`, `ethereum`, ...) or fiat currency codes
(`EUR`, `USD`, `GBP`, `JPY`, `BRL`, `CHF`, `AUD`, `CAD`, `SGD`, `IDR`, `THB`,
`PYG`).

- crypto: `api.coingecko.com` (simple price, usd)
- fiat: `open.er-api.com`, `frankfurter.app` as fallback

If an api does not know an id, its last saved rate is kept and marked with `*`.
