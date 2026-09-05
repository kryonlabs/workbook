# workbook

Native spreadsheet-style workbook editor authored in Kry. `workbook.kry` is
the application source; `k2c` lowers it through KIR to generated C for the
native Kryon runtime.

## build & run

    make              # compile workbook.kry and build ./workbook
    make test         # source audit plus native UI smoke test
    make run          # open ./workbook
    ./workbook        # generic workbook
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
