#!/bin/sh
# Workbook is authored in Ziran: no Kry source, handwritten C or Go, vendored
# runtime, or SSH package URL may come back.
set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$root"

# Everything outside build outputs and the git directory.
sources() {
    find . \( -path ./.git -o -path ./build -o -path ./dist \) -prune -o -type f "$@" -print
}

if sources \( -name '*.kry' -o -name '*.krb' -o -name '*.kir' \) | grep -q .; then
    echo 'legacy Kry source remains; Workbook is written in Ziran (.zi)' >&2
    sources \( -name '*.kry' -o -name '*.krb' -o -name '*.kir' \) >&2
    exit 1
fi

if sources \( -name '*.c' -o -name '*.h' -o -name '*.go' \) | grep -q .; then
    echo 'handwritten C or Go source remains; application behavior belongs in Ziran' >&2
    sources \( -name '*.c' -o -name '*.h' -o -name '*.go' \) >&2
    exit 1
fi

if [ -e .gitmodules ] || [ -e vendor ]; then
    echo 'dependencies are Ziran packages in ziran.toml, not submodules or vendor/' >&2
    exit 1
fi

for required in ziran.toml ziran.lock src/app.zi src/engine.zi src/cell.zi \
    workbook.example.json profiles/geld.json; do
    if [ ! -f "$required" ]; then
        echo "missing $required" >&2
        exit 1
    fi
done

bad=$(grep -nE '^[[:space:]]*git[[:space:]]*=' ziran.toml |
    grep -vE '^[0-9]+:[[:space:]]*git[[:space:]]*=[[:space:]]*"https://' || true)
if [ -n "$bad" ]; then
    echo "package URLs must be public HTTPS:" >&2
    echo "$bad" >&2
    exit 1
fi

printf '%s\n' '{"workbook_source_audit":"ok"}'
