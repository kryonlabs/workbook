# workbook

Native spreadsheet editor written in [Ziran](https://github.com/ziranlang/ziran)
with [Kryon](https://github.com/kryonlabs/kryon), compatible with Gnumeric file
formats and expressions. Everything in the repository is Ziran (`.zi`) apart
from the Makefile and configuration. `src/app.zi` is the desktop application and
`src/engine.zi` is the spreadsheet engine (grid model, formula evaluator,
function library, .gnumeric/CSV I/O). `src/cell.zi` is the headless driver
built on the same engine, and `tests/*.zi` are the test programs.

The engine reads and writes `.gnumeric` (Gnumeric XML, gzipped or plain),
evaluates the Gnumeric expression language (references, ranges, sheets,
operators, 120+ functions with Gnumeric semantics including errors and
date serials), and is verified 1:1 against the installed Gnumeric by
`tests/parity_test.zi` — every fixture is evaluated by both engines and
compared cell by cell, including round-trips through our own
`.gnumeric` writer. Format notes: `docs/GNUMERIC_PARITY.md`.

The engine's helpers live beside it: `src/quad.zi` (double-double arithmetic),
`src/xml.zi` and `src/gzip.zi` (the Gnumeric file format), `src/formula_text.zi`
(cell names and rounding used by formulas), and `src/c_runtime.zi` (the C
library calls the engine makes).

## build & run

Workbook is a Ziran package. `ziran.toml` names the dependencies and
`ziran.lock` pins the Ziran toolchain and Kryon to exact commits, so a fresh
clone builds the same code. Install the `ziran` command once with
`make install-user` in a Ziran checkout, and have the SDL2 and Cairo
development packages installed.

    make              # build ./build/workbook-desktop and ./build/cell
    make cell         # build only the headless engine driver
    make check        # type-check the application, the driver, and the tests
    make run          # open the editor
    make test         # source audit, desktop test on a private display, gnumeric parity
    make test-ci      # the same without parity, which is what CI runs
    make parity       # 1:1 evaluation tests against the installed gnumeric
    make install      # install workbook, cell, and geld under ~/.local
    make deb          # build a Debian package

    cell eval FILE    # evaluate a .gnumeric/.csv file and print CSV
    geld              # the finance profile: a link to workbook

Update the pinned dependencies with `ziran update Kryon` or `ziran update ziran`
and commit the new `ziran.lock`. To build against sibling checkouts while
developing Kryon or Ziran, put their paths in an ignored `ziran.local.toml`:

    [overrides]
    Kryon = "../kryon"
    ziran = "../ziran"

`make test` needs `xvfb-run`, `xdotool`, and `gnumeric` (for `ssconvert`). The
desktop test (`tests/desktop_test.zi`) runs the editor only on a private Xvfb
display; it never opens a window on your desktop. Parity compares against the
`ssconvert` on your path, or
against `GNUMERIC_SSCONVERT`, or a local master build at
`~/Tools/gnumeric-1.12.62`; the newest catalog functions exist only in a
Gnumeric master build, so an older distribution Gnumeric reports those fixtures
as different. CI therefore runs `make test-ci`.

## editing

    arrows            move the selection
    Tab               move right
    type              start editing the selected cell (replacing its value)
    Enter             edit the selected cell; in the formula bar, commit and move down
    Esc               leave the formula bar without changing the cell
    Delete            clear the selected cell
    double click      edit the clicked cell in the formula bar
    wheel             scroll rows
    Ctrl+S            save

Formula cells show their calculated value in the grid and their source in the
formula bar. The toolbar provides save and row/column insertion and deletion,
and the sheet tabs at the bottom switch sheets. Cells accept text, numbers, or
formulas in the Gnumeric expression language.

## profiles

`workbook` is the generic profile. `geld` is a link to the same program; it
opens the finance profile, with an independent data directory, because of the
name it is started under (`WORKBOOK_PROFILE=geld workbook` does the same). Both
run the same application:

    D = units
    E = rate
    G = value, e.g. =D1*E1
    H = diff, e.g. =(E1-OLD_RATE)*D1

The `profiles/*.json` files are profile metadata, not source code.

## data

A workbook is saved as `workbook.gnumeric`. It loads from the first of:

    WORKBOOK_DIR      generic workbook directory (GELD_DIR for the geld profile)
    workbook.gnumeric in the current directory
    workbook.json     in the current directory (the legacy profile format)
    ~/.local/share/workbook/workbook.gnumeric   (~/.local/share/geld for geld)

`workbook.json` holds private workbook data and is intentionally ignored by
git. `workbook.example.json` is safe sample data for demos.
