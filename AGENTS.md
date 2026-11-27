# workbook

A native spreadsheet editor written in Ziran with Kryon. It opens, edits and
saves `.gnumeric` and CSV documents; `geld` is the finance profile. README.md
documents usage, dependencies and the test tooling.

## Layout

| Path | Contents |
| --- | --- |
| `src/editor.zi` | Editor state, input and spreadsheet actions |
| `src/editor_api.zi` | Embeddable editor exported as `workbook/editor` (handle, frame, save) |
| `src/editor_view.tsx`, `src/grid_view.tsx` | Editor and cell composition |
| `src/app.zi`, `src/app_view.tsx` | Desktop and Werkzeug document hosts, menus and application layout |
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

- Ziran toolchain: `23dece1339d51cf51730be438325605e6ec29c87`
- Kryon: `3fc33496121382cc31a53a8c3178e0f2c90977fb`
- mingcute: `9cc6872f75447c3a24c16764f9829d0f58751cb0`; css: `452a760f7a5befc5d01f28a922f48e3ce1687e7f`

Commit as Waozi <waozi@proton.me>; ordinary changes stay in CHANGELOG's
`Unreleased` section.
