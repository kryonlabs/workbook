# Changelog

## [Unreleased]

- Export the full editor as `workbook/editor` for other Kryon apps:
  `EditorOpen`/`EditorNew` return a handle, `EditorFrame` draws and handles one
  frame in a host region, `EditorSave`/`EditorSaveAs` save it, and
  `EditorClose` releases it. Undo, autosave, focus and the per-document change
  and save counters belong to the handle; `EditorSample` passes input without
  taking the keyboard from an unfocused host. `make host-test` embeds it in a
  minimal host on a private display.
- The standalone app and the Werkzeug bundle use that same editor; `src/app.zi`
  keeps profiles, documents, the window title, menus and the theme.
- Build the Werkzeug bundle from the project's exports with
  `ziran bundle --project`.
- Pin Kryon `facaace3`, CSS `7dd9fff2` and Ziran `9432a110` to share the
  Autotrade, plot and data_views build graph.
- Build the editor and embedded grid against Kryon's public widget modules.
- Other Kryon apps embed Workbook's own grid with `workbook/grid` and
  `workbook/engine`: real cells and formulas, sheet tabs, column widths and
  count or money formats. The editor paints with the same grid squares. It
  replaces the short-lived `workbook/sheet_table`.
- The embedded grid scrolls three rows a wheel notch (Shift+wheel for
  columns), can keep title rows on top and style total rows, and can draw
  only the used cells like a report.
- Workbook now lives at github.com/kryonlabs/workbook.
- Hide selected rows from the row-number menu and click their thin lines to
  restore them, with undo and redo support.
- Use `SUBTOTAL(109,…)` to leave hidden rows out of totals while preserving
  them in saved Gnumeric documents.
