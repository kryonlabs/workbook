# Workbook ↔ Gnumeric parity notes

Imported documents use Gnumeric's installed native library, loaded through
`src/gnumeric.zi`. The native document remains alive for calculation, edits,
structural operations and saves. Undo clones it through an in-memory native
save and load, so document metadata is part of the undo state too.

`tests/gnumeric_test.zi` covers shared expressions, string time arithmetic,
named expressions, date and decimal formats, literal strings, booleans,
arrays and their edit boundaries, coordinates beyond 256 columns and 65,536
rows, UTF-16 XML, edits, row changes, independent undo snapshots and native
round trips. The source URI is retained for relative workbook links.
Its synthetic `tests/document.gnumeric` lives outside the CSV fixture folder
because its large coordinates would produce millions of empty CSV fields.

Compare imported files against `ssconvert` from the same installed Gnumeric
version. A newer reference can have functions or plugins missing from the
runtime library. For the distribution installation:

    GNUMERIC_SSCONVERT=/usr/bin/ssconvert make parity

The basic grid displays Gnumeric's formatted cell values. Native document
support does not imply complete UI parity: the grid retains its own layout
and styling. Cell fonts, fills, borders, sizes, merged regions, charts, drawing
objects and print layout are retained on save; drawing objects and print
layout are not rendered or edited by the grid.
Gnumeric canonicalizes function names to lowercase and exposes a leading
apostrophe in entered text for literal strings that resemble formulas.
Unknown features from versions newer than the installed Gnumeric are subject
to that Gnumeric version's own import limitations.

The notes below describe the file format and Workbook's independent Ziran
engine, which still serves new workbooks and CSV. They do not define the
native document library's feature inventory.

## The .gnumeric container

A `.gnumeric` file is XML in the `http://www.gnumeric.org/v10.dtd` namespace,
either plain or gzip-compressed. Gnumeric itself writes gzip; it accepts both
on read. Workbook reads both. Imported documents are saved by Gnumeric in its native
compressed format; new workbooks use the Ziran writer and plain XML.

Minimal document Gnumeric 1.12.57 accepts:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<gnm:Workbook xmlns:gnm="http://www.gnumeric.org/v10.dtd">
  <gnm:Version Epoch="1" Major="12" Minor="57" Full="1.12.57"/>
  <gnm:SheetNameIndex><gnm:SheetName gnm:Cols="256" gnm:Rows="65536">Data</gnm:SheetName></gnm:SheetNameIndex>
  <gnm:Sheets>
    <gnm:Sheet>
      <gnm:Name>Data</gnm:Name>
      <gnm:MaxCol>0</gnm:MaxCol><gnm:MaxRow>0</gnm:MaxRow>
      <gnm:Cells>
        <gnm:Cell Row="0" Col="0" ValueType="40">3.1415899999999999</gnm:Cell>
        <gnm:Cell Row="1" Col="0">=A1*2</gnm:Cell>
      </gnm:Cells>
      <gnm:SheetLayout TopLeft="A1"/>
    </gnm:Sheet>
  </gnm:Sheets>
  <gnm:UIData SelectedTab="0"/>
</gnm:Workbook>
```

Full document order observed from Gnumeric output:
`Version, Attributes, office:document-meta, Calculation, SheetNameIndex,
Sheets, UIData`. Sheet children order: `Name, MaxCol, MaxRow, Zoom, Names,
PrintInformation, Styles, Cols, Rows, Selections, Cells, SheetLayout, Solver`,
plus optional `MergedRegions`, `Filters`, `Objects`, `SheetObjectLayout`
blocks. Cell content must be XML-escaped (`&lt;` `&gt;` `&amp;` `&quot;`).

### ValueType attribute

| Code | Meaning | Serialized form |
|---|---|---|
| (none) | formula | text starting with `=` |
| 20 | boolean | `TRUE` / `FALSE` |
| 40 | float | `%.17g` (e.g. `3.1415899999999999`) |
| 50 | error | `#DIV/0!`, `#VALUE!`, `#REF!`, `#NAME?`, `#N/A`, `#NUM!`, `#NULL!` |
| 60 | string | UTF-8 text |

Integers also serialize as ValueType 40. Empty cells are simply absent.

### Styles

`gnm:Styles` → `gnm:StyleRegion startCol/startRow/endCol/endRow` →
`gnm:Style` attributes: `HAlign, VAlign, WrapText, ShrinkToFit, Rotation,
Shade, Indent, Locked, Hidden, Fore, Back, PatternColor, Format` and a
`gnm:Font` child (`Unit, Bold, Italic, Underline, StrikeThrough, Script`,
text = font name). Colors are 16-bit-per-channel `R:G:B` (e.g. `FFFF:0:0`).
`Format` is a number-format string (`General`, `0.00`, `#$,##0.00`, …).
Column widths: `gnm:Cols` → `gnm:ColInfo No="..." Unit="..." Count="..."` plus
`DefaultSizePts`; rows analogous with `gnm:RowInfo`.

## Expression language semantics (verified against 1.12.57)

- Operators by precedence: `:` range, ` ` intersect, unary `-`/`+`, `%` (÷100),
  `^`, `*` `/`, `+` `-`, `&` concat, comparisons `= <> < > <= >=`.
- `&` concatenates, coercing numbers to text (`"a"&"b"&1` → `ab1`).
- Comparisons produce booleans; booleans coerce to 1/0 in arithmetic
  (`=(1>2)+(3<>4)` → `1`).
- Text operand in arithmetic → `#VALUE!` (`="x"*2`).
- Range aggregators (SUM, AVERAGE, …) **skip** text and empty cells in ranges
  but count scalar arguments (`=AVERAGE(A1:A3,10)` counts the 10).
- References: `A1` relative, `$A$1` absolute, mixed `$A1`/`A$1`; sheets
  `Other!A1`; ranges `A1:B3`, `Other!A1:B3`, absolute `$A$1:$B$3`.
- Unknown function → `#NAME?`; wrong argument domain → `#NUM!`/`#VALUE!`;
  bad reference → `#REF!`; 0/0 → `#DIV/0!`; intersection of disjoint ranges
  → `#NULL!`.
- Booleans render `TRUE`/`FALSE` in CSV export; locale-sensitive rendering is
  avoided by running under `LC_ALL=C`.

## Function inventory (440 in this install, grouped by Gnumeric plugin)

math 63, stat 103, string 29, financial 40, complex 49, random 34, lookup 20,
info 21, date 13, database 13, eng 12, numtheory 9, logical 5, flt 4,
christian-date 6, tsa 4, hebrew-date 1, derivatives 1.

The independent Ziran engine implements a subset of the function inventory
for new workbooks and CSV. Imported Gnumeric documents use the installed
native library's functions and plugins. The current parity suite verifies
that document integration, evaluated exports and saved round trips; it is not
a separate numerical oracle for the independent evaluator.

## 1:1 test methodology

Ground truth is the installed Gnumeric itself, never hand-computed:

1. Fixture `.gnumeric` files live in `tests/fixtures/`.
2. `ssconvert fixture.gnumeric out.csv` (LC_ALL=C) → expected values.
3. `cell eval fixture.gnumeric` → workbook's own evaluation as CSV.
4. `tests/parity_test.zi` compares both outputs cell by cell.
5. Round-trip: `cell copy in out.gnumeric` writes a `.gnumeric` that Gnumeric
   re-reads with identical evaluated values.
