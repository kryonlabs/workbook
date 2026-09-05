# workbook

Native spreadsheet editor authored in Kry, compatible with Gnumeric file
formats and expressions. `workbook.kry` is the desktop application;
`src/engine.kry` is the spreadsheet engine (grid model, formula evaluator,
function library, .gnumeric/CSV I/O); `k2c` lowers both through KIR to
generated C for the native Kryon runtime.

The engine reads and writes `.gnumeric` (Gnumeric XML, gzipped or plain),
evaluates the Gnumeric expression language (references, ranges, sheets,
operators, 120+ functions with Gnumeric semantics including errors and
date serials), and is verified 1:1 against the installed Gnumeric by
`scripts/parity-test.sh` — every fixture is evaluated by both engines and
compared cell by cell, including round-trips through our own
`.gnumeric` writer. Format notes: `docs/GNUMERIC_PARITY.md`.

## build & run

    make              # compile workbook.kry and build ./workbook
    make cell         # build the headless engine driver ./cell
    make test         # source audit, native UI smoke, 1:1 gnumeric parity
    make parity       # 1:1 evaluation tests against the installed gnumeric
    make run          # open ./workbook
    ./workbook        # generic workbook
    cell eval FILE    # evaluate a .gnumeric/.csv file and print CSV
    geld              # installed finance profile wrapper
    make install      # install workbook, cell, and geld

## editing

The desktop editor uses Kryon's native UI, layout, storage, and JSON APIs
directly from `workbook.kry`.

    up/down wheel     select / scroll
    left/right Tab    move across cells
    double click      focus the selected cell in the formula bar
    Enter             commit the formula bar value
    Delete            clear the selected cell
    Ctrl+S            save
    Esc               quit

Formula cells show their calculated value in the grid and their source in the
formula bar. Arithmetic, cell references such as `=D1*E1`, and rectangular
`SUM` ranges such as `=SUM(G1:G8)` are supported.

The toolbar provides save and row/column insertion and deletion. Cells accept
text, numbers, or formulas. Free-form values are saved in `workbook.json`.

## profiles

`workbook` is the generic profile. `geld` uses an independent data directory
for finance rows. Both execute the same Kry application source:

    D = units
    E = rate
    G = value, e.g. =D1*E1
    H = diff, e.g. =(E1-OLD_RATE)*D1

The `profiles/*.json` files are profile metadata, not Kry source.

## data

Data loads from the profile environment variable, then a `workbook.json` in
the current directory, then the profile data directory.

    WORKBOOK_DIR      generic workbook override
    GELD_DIR          geld finance workbook override
    ~/.local/share/workbook
    ~/.local/share/geld

`workbook.json` holds private workbook data and is intentionally ignored by
git. `workbook.example.json` is safe sample data for demos.

JSON remains the persisted data format. `.kry` is used only for executable Kry
source code.
