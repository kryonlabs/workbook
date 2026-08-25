#!/bin/sh
set -eu

: "${GOCACHE:=${TMPDIR:-/tmp}/workbook-native-runtime-audit-go-cache}"
export GOCACHE

go_files=$(find . -maxdepth 1 -type f -name '*.go' -printf '%p\n')
mod_files=$(find . -maxdepth 1 -type f \( -name 'go.mod' -o -name 'go.*.mod' \) -printf '%p\n')
scan_files="README.md $mod_files $go_files"
if [ -d .github ]; then
    github_files=$(find .github -type f)
    scan_files="$scan_files $github_files"
fi
if [ -d ui ]; then
    ui_files=$(find ui -type f)
    scan_files="$scan_files $ui_files"
fi

# shellcheck disable=SC2086
matches="$(rg -n 'github.com/waozixyz/kryon/go/kryui|go/kryui|import "C"|DrawUI|UIText|TextInputControl|UIRender|UIButtonStyle|UI_BUTTON_STYLE_[A-Z_]+|UI_TEXT(_BASE_SIZE|_[0-9]+)' $scan_files || true)"

if [ -n "$matches" ]; then
    echo "Workbook still depends on blocked legacy/cgo runtime surfaces:" >&2
    echo "$matches" >&2
    exit 1
fi

printf '{"workbook_native_runtime_audit":"ok"}\n'
