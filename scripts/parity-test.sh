#!/bin/sh
# 1:1 parity test: evaluate every fixture with Gnumeric (ssconvert) and with
# the workbook engine (cell), compare every output cell. Ground truth is the
# installed Gnumeric itself.
set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=${TMPDIR:-/tmp}/workbook-parity.$$
cell_bin=${CELL_BIN:-$root/cell}

cleanup() { rm -rf "$work"; }
trap cleanup EXIT INT TERM

if ! command -v ssconvert >/dev/null 2>&1; then
    echo "parity: ssconvert not found; install gnumeric to run 1:1 tests" >&2
    exit 1
fi
if [ ! -x "$cell_bin" ]; then
    echo "parity: cell binary not found at $cell_bin (run make cell)" >&2
    exit 1
fi

mkdir -p "$work"

failures=0
checked=0

norm_csv() {
    python3 - "$1" "$2" <<'EOF'
import csv, sys
rows = list(csv.reader(open(sys.argv[1], newline='')))
while rows and all(c == '' for c in rows[-1]):
    rows.pop()
width = 0
for r in rows:
    w = len(r)
    while w > 0 and r[w-1] == '':
        w -= 1
    width = max(width, w)
with open(sys.argv[2], 'w') as out:
    for r in rows:
        out.write(','.join(r[:width]) + '\n')
EOF
}

for fixture in "$root"/tests/fixtures/*.gnumeric; do
    name=$(basename "$fixture" .gnumeric)
    LC_ALL=C ssconvert "$fixture" "$work/$name.gnm.csv" >/dev/null 2>&1
    "$cell_bin" eval "$fixture" -o "$work/$name.cell.csv" >/dev/null 2>&1
    norm_csv "$work/$name.gnm.csv" "$work/$name.gnm.norm"
    norm_csv "$work/$name.cell.csv" "$work/$name.cell.norm"
    if ! diff -u "$work/$name.gnm.norm" "$work/$name.cell.norm" > "$work/$name.diff" 2>&1; then
        echo "FAIL $name"
        sed -n '1,20p' "$work/$name.diff"
        failures=$((failures + 1))
    else
        checked=$((checked + 1))
        echo "ok   $name"
    fi
done

# round-trip: a .gnumeric written by our engine must be readable by Gnumeric
# with the same evaluated values
for fixture in "$root"/tests/fixtures/*.gnumeric; do
    case "$(basename "$fixture")" in *_gzip.gnumeric) continue ;; esac
    name=$(basename "$fixture" .gnumeric)
    "$cell_bin" eval "$fixture" -o "$work/$name.cells.csv" >/dev/null 2>&1
    "$cell_bin" copy "$fixture" "$work/$name.ours.gnumeric" >/dev/null 2>&1
    LC_ALL=C ssconvert "$work/$name.ours.gnumeric" "$work/$name.back.csv" >/dev/null 2>&1
    norm_csv "$work/$name.cells.csv" "$work/$name.cells.norm"
    norm_csv "$work/$name.back.csv" "$work/$name.back.norm"
    if ! diff -u "$work/$name.cells.norm" "$work/$name.back.norm" > "$work/$name.rt.diff" 2>&1; then
        echo "FAIL $name (round-trip)"
        sed -n '1,20p' "$work/$name.rt.diff"
        failures=$((failures + 1))
    else
        echo "ok   $name (round-trip)"
    fi
done

if [ "$failures" != 0 ]; then
    echo "parity: $failures failing case(s)"
    exit 1
fi
echo "parity: $checked fixture(s) 1:1 with gnumeric"
