# workbook

A native spreadsheet editor written in Ziran with Kryon. It opens, edits and
saves `.gnumeric` and CSV documents; `geld` is the finance profile. README.md
documents usage, dependencies and the test tooling.

## Layout

| Path | Contents |
| --- | --- |
| `src/app.zi` | Desktop application |
| `src/engine.zi` | Spreadsheet engine for new workbooks and CSV |
| `src/cell.zi` | Headless engine driver (`cell eval FILE`) |
| `src/gnumeric.zi` | Imported-document support through the installed gnumeric library |
| `tests/` | Structure, conversion, desktop, menu and parity tests |
| `docs/GNUMERIC_PARITY.md` | Document-format notes and the function inventory |

## Commands

```sh
make            # build ./build/workbook-desktop and ./build/cell
make check      # type-check app, driver and tests
make test       # full suite; desktop test on a private display, parity
make test-ci    # CI suite without parity
make parity     # compare against gnumeric ssconvert
make fuzz REFERENCE=PATH  # compare against another cell build
make install    # install workbook, cell and geld under ~/.local
```

## Pins

- Ziran toolchain: `036cb7b5d2630302e0b73a2336ccbfb6d0250be5`
- Kryon: `c7286549f7741c48730cc983f14aed233cb47c37`
- mingcute: `9cc6872f75447c3a24c16764f9829d0f58751cb0`; css: `df8bbbbd665bd0632cd8687a145addc0b745eb87`

Commit as Waozi <waozi@proton.me>; ordinary changes stay in CHANGELOG's
`Unreleased` section.
