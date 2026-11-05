# Changelog

## [Unreleased]

- Other Kryon apps embed Workbook's own grid with `workbook/grid` and
  `workbook/engine`: real cells and formulas, sheet tabs, column widths and
  count or money formats. The editor paints with the same grid squares. It
  replaces the short-lived `workbook/sheet_table`.
- Workbook now lives at github.com/kryonlabs/workbook.
- Hide selected rows from the row-number menu and click their thin lines to
  restore them, with undo and redo support.
- Use `SUBTOTAL(109,…)` to leave hidden rows out of totals while preserving
  them in saved Gnumeric documents.
