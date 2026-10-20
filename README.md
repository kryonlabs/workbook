# workbook

Native spreadsheet editor written in [Ziran](https://github.com/ziranlang/ziran)
with [Kryon](https://github.com/kryonlabs/kryon). Open a document from your file
manager, drop it onto Workbook, or pass its path directly:

    workbook /path/to/file.gnumeric

The window title shows the document's filename. Edits save back to that file;
when you drop another document, Workbook saves the current edit before opening
it. Launching without a file uses the profile described below.

Everything in the repository is Ziran (`.zi`) apart from the Makefile and
configuration. `src/app.zi` is the desktop application, `src/engine.zi` is the
spreadsheet engine, and `src/cell.zi` is the headless driver. Imported documents
use the installed `gnumeric` library for loading, calculation and saving.
New workbooks and CSV use the independent Ziran engine. Document format,
dependencies and rendering limitations are covered in
[format notes](docs/GNUMERIC_PARITY.md).

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
    make test         # source audit, structure and conversion tests, desktop test on a private display, gnumeric parity
    make test-ci      # the same without parity, which is what CI runs
    make gnumeric-test # native document, edit, undo and round-trip regressions
    make parity       # evaluation and round-trip tests against Gnumeric
    make fuzz REFERENCE=PATH   # compare this cell with another on generated inputs
    make install      # install workbook, cell, and geld under ~/.local
    make deb          # build a Debian package

    cell eval FILE    # evaluate a .gnumeric/.csv file and print CSV
    geld              # the finance profile: a link to workbook

Update the pinned dependencies with `ziran update kryon` or `ziran update ziran`
and commit the new `ziran.lock`. To build against sibling checkouts while
developing Kryon or Ziran, put their paths in an ignored `ziran.local.toml`:

    [overrides]
    kryon = "../kryon"
    ziran = "../../ziranlang/ziran"

`make test` needs `xvfb-run`, `xdotool`, `xfwm4`, `dbus-run-session`, `xmessage`,
ImageMagick's `import`, and `gnumeric` (for `ssconvert`). The
desktop test (`tests/desktop_test.zi`) runs the editor only on a private Xvfb
display; it never opens a window on your desktop. Parity compares against the
`ssconvert` on your path, or
against `GNUMERIC_SSCONVERT`, or a local master build at
`~/Tools/gnumeric-1.12.62`; the newest catalog functions exist only in a
Gnumeric master build, so an older distribution Gnumeric reports those fixtures
as different. For imported documents, use a reference from the same installed
version, e.g. `GNUMERIC_SSCONVERT=/usr/bin/ssconvert make parity`.
CI runs `make test-ci` with Gnumeric installed.

The menu test also runs a private Xfce window manager and D-Bus session. It
checks that the first right-click after another window has focus shows the
menu, that commands act on the clicked cell, and that menus work after resizing.
To verify the installed app, run:

    WORKBOOK_TEST_APP="$HOME/.local/bin/workbook" build/desktop_test --menu-only

`tests/differential_fuzz.zi` checks one build of `cell` against another. Point
`REFERENCE` at a build whose answers are right, for example `cell` built from an
earlier commit, and `make fuzz` runs the current build and the reference on the
same generated inputs, then reports each one they answer differently: random
function calls on a fixed sheet, random CSV files, the fixtures with random
damage, and workbooks whose sheets refer to each other. `SEED` and `COUNT`
choose the inputs, and the program can also be run by hand:

    build/differential_fuzz REFERENCE build/cell formulas 1 100
    build/differential_fuzz REFERENCE build/cell check 'DAY(-2147483648.5)'

A run that hangs is killed after `FUZZ_TIMEOUT` seconds (20 by default).

## editing

    arrows            move the selection
    Tab               move right
    type              start editing the selected cell (replacing its value)
    Enter             edit the selected cell; in the formula bar, commit and move down
    Esc               leave the formula bar without changing the cell
    Delete            clear the selected cell
    double click      edit the clicked cell in the formula bar
    right click       open the cell menu, including when activating the window
    Ctrl+C/X/V        copy, cut, or paste cells through the system clipboard
    Ctrl+Z            undo; while a cell is edited, undo its typing word by word
    Ctrl+Y            redo (also Ctrl+Shift+Z)
    wheel             scroll rows
    Ctrl+S            save

Formula cells show their calculated value in the grid and their source in the
formula bar. The toolbar provides save and row/column insertion and deletion,
and the sheet tabs at the bottom switch sheets. Cells accept text, numbers, or
formulas in the Gnumeric expression language.

Changes save automatically, including the cell you are typing, pasted values,
and inserted or deleted rows and columns. Clicking another cell or closing the
app keeps the latest edit without requiring Enter or Ctrl+S. Escape restores
the cell's value from before the edit. A failed save leaves the previous file
intact, reports the failure in the status bar, and retries automatically.

Undo and redo cover every change: a finished cell edit, paste, cut, clear,
fill, and row or column insertion and deletion, each with the formula
references it moved. Undoing a change selects the cells it touched again and
shows their sheet. Up to the last 100 changes can be undone.

## profiles

`workbook` is the generic profile. `geld` is a link to the same program; it
opens the finance profile, with an independent data directory, because of the
name it is started under (`WORKBOOK_PROFILE=geld workbook` does the same).
`WORKBOOK_PROFILE=NAME workbook` opens any other profile the same way, with its
data in `$XDG_DATA_HOME/NAME`; a name uses lowercase letters, digits, `-` and
`_`. Harmony starts its Workbook profiles like this. All of them run the same
application:

    D = units
    E = rate
    G = value, e.g. =D1*E1
    H = diff, e.g. =(E1-OLD_RATE)*D1

The `profiles/*.json` files are profile metadata, not source code.

## data

A workbook is saved as `workbook.gnumeric`. The generic profile loads from the
first of:

    WORKBOOK_DIR      generic workbook directory
    workbook.gnumeric in the current directory
    ~/.local/share/workbook/workbook.gnumeric

Geld always uses `GELD_DIR` when set, or `$XDG_DATA_HOME/geld` (default
`~/.local/share/geld`). A workbook in the directory Geld was launched from
does not replace its finance profile. Missing data directories are created
automatically. Named profiles and explicit data-directory overrides import
legacy `workbook.json` files when no `workbook.gnumeric` exists beside them; subsequent changes go to
`workbook.gnumeric`, keeping the original JSON intact. Ordinary Workbook launches leave the former
finance JSON untouched and start with an empty sheet when no saved document exists.

`workbook.json` holds private workbook data and is intentionally ignored by
git. `workbook.example.json` is safe sample data for demos.


## Werkzeug app bundle

`make bundle` builds `build/workbook.zib`, the spreadsheet editor for a host
with the `spreadsheet.v1` capability. Werkzeug on Linux includes the native
calculation service and keeps Workbook documents in its own private folder.
The bundle has the same compact grid, bottom sheet tabs and editing commands
as the desktop editor. Its File menu imports and exports documents with the
system chooser; imported source files stay unchanged. File → Download
Workbook.zib saves the actual app bundle. Open that file with Werkzeug, or
run `werk --bundle /path/to/workbook.zib`.

The native menu bar contains File, Edit, View, Insert and Sheet. Ctrl+O opens
a document, Ctrl+Shift+S saves a copy, and F10 or Alt+F focuses the menus.
