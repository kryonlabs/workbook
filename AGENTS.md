# workbook

A native spreadsheet editor written in Ziran with Kryon. It opens, edits and
saves `.gnumeric` and CSV documents; `geld` is the finance profile. README.md
documents usage, dependencies and the test tooling.

## Layout

| Path | Contents |
| --- | --- |
| `src/editor.zi` | Embeddable editor exported as `workbook/editor` (handle, frame, save) |
| `src/app.zi` | Desktop host: profiles, documents, title, menus and theme |
| `src/engine.zi` | Spreadsheet engine for new workbooks and CSV |
| `src/cell.zi` | Headless engine driver (`cell eval FILE`) |
| `src/gnumeric.zi` | Imported-document support through the installed gnumeric library |
| `previews/host.zi` | Minimal host with a sidebar around the embedded editor |
| `tests/` | Structure, conversion, desktop, host, menu and parity tests |
| `docs/GNUMERIC_PARITY.md` | Document-format notes and the function inventory |

## Commands

```sh
make            # build ./build/workbook-desktop and ./build/cell
make check      # type-check app, driver and tests
make test       # full suite; desktop and host tests on private displays, parity
make test-ci    # CI suite without parity
make host-test  # the exported editor inside a minimal host, private display
make parity     # compare against gnumeric ssconvert
make fuzz REFERENCE=PATH  # compare against another cell build
make install    # install workbook, cell and geld under ~/.local
```

## Pins

- Ziran toolchain: `c041afb6de9b5f5f8be13c367563c267841e1cb6`
- Kryon: `09664f076d3e8ea8db2013fb46fbcd9eda45c54b`
- mingcute: `9cc6872f75447c3a24c16764f9829d0f58751cb0`; css: `7dd9fff21bd1721deea91071a2317d106e901208`

Commit as Waozi <waozi@proton.me>; ordinary changes stay in CHANGELOG's
`Unreleased` section.
