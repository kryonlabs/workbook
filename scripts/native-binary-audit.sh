#!/bin/sh
set -eu

bin=${1:-./workbook}

if [ ! -x "$bin" ]; then
    echo "native binary audit target not found or not executable: $bin" >&2
    exit 1
fi

info=$(go version -m "$bin" 2>/dev/null || true)
if ! printf '%s\n' "$info" | rg -q 'github\.com/waozixyz/kryon/go/kryon'; then
    echo "Workbook binary is not linked against clean go/kryon runtime: $bin" >&2
    printf '%s\n' "$info" >&2
    exit 1
fi
if ! printf '%s\n' "$info" | rg -q 'build\s+CGO_ENABLED=0'; then
    echo "Workbook binary was not built with CGO_ENABLED=0: $bin" >&2
    printf '%s\n' "$info" >&2
    exit 1
fi

matches_file=$(mktemp "${TMPDIR:-/tmp}/workbook-native-binary-audit.XXXXXX")
cleanup() { rm -f "$matches_file"; }
trap cleanup EXIT INT TERM

if LC_ALL=C strings "$bin" | rg -n 'github\.com/waozixyz/kryon/go/kryui|go/kryui|import "C"|DrawUI|UIText|TextInputControl|UIRender|UIButtonStyle|UI_BUTTON_STYLE_[A-Z_]+|UI_TEXT(_BASE_SIZE|_[0-9]+)|_Cfunc_(BeginDrawing|EndDrawing)' >"$matches_file"; then
    echo "Workbook binary still contains blocked legacy/cgo rendering symbols: $bin" >&2
    cat "$matches_file" >&2
    exit 1
fi

printf '{"workbook_native_binary_audit":"ok","binary":"%s"}\n' "$bin"
