# Changelog

## [Unreleased]

- Add `workbook/sheet_table`, a read-only table other apps embed: counts,
  money, totals row, fitted columns, screen layout and CSV export.
- Hide selected rows from the row-number menu and click their thin lines to
  restore them, with undo and redo support.
- Use `SUBTOTAL(109,…)` to leave hidden rows out of totals while preserving
  them in saved Gnumeric documents.
