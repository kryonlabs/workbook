#!/bin/sh
set -eu

kryon_dir=${1:-vendor/kryon}
k2g=${2:-"$kryon_dir/build/linux-x86_64/bin/k2g"}
source_file=${3:-ui/workbook_native.kry}
work=$(mktemp -d "${TMPDIR:-/tmp}/workbook-native-ui-smoke.XXXXXX")

cleanup() { rm -rf "$work"; }
trap cleanup EXIT INT TERM

if [ ! -x "$k2g" ]; then
    echo "k2g not found or not executable: $k2g" >&2
    exit 1
fi
if [ ! -f "$source_file" ]; then
    echo "native smoke source not found: $source_file" >&2
    exit 1
fi

kryon_abs=$(cd "$kryon_dir" && pwd)
mkdir -p "$work/out"
matches_file="$work/blocked.matches"

"$k2g" --root "$kryon_dir" -o "$work/out" "$source_file"
sh "$kryon_dir/tests/check_clean_generated_output.sh" "$work/out"

out=$(find "$work/out" -name '*.go' | head -1)
[ -f "$out" ] || { echo "k2g produced no Go output" >&2; exit 1; }

grep -q 'import kryon "github.com/waozixyz/kryon/go/kryon"' "$out"
grep -q 'kryon.BeginFrame()' "$out"
grep -q 'kryon.TextField(kryon.TextFieldProps{' "$out"
grep -q 'kryon.TextArea(kryon.TextAreaProps{' "$out"
grep -q 'kryon.Button(kryon.ButtonProps{' "$out"
if rg -n 'kryon\.BeginUI|kryon\.EndUI' "$out" >"$matches_file"; then
    echo "native Workbook smoke generated retained UI lifecycle calls:" >&2
    cat "$matches_file" >&2
    exit 1
fi

if rg -n 'go/kryui|import "C"|DrawUI|UIText|TextInputControl|UIRender|BeginDrawing|EndDrawing|UIButtonStyle|UI_BUTTON_STYLE_[A-Z_]+|UI_TEXT(_BASE_SIZE|_[0-9]+)' "$work/out" >"$matches_file" 2>/dev/null; then
    echo "native Workbook smoke generated blocked runtime names:" >&2
    cat "$matches_file" >&2
    exit 1
fi

cat > "$work/out/go.mod" <<EOF
module workbook-native-ui-smoke

go 1.25.0

require (
    github.com/waozixyz/kryon/go/kryon v0.0.0
    golang.org/x/image v0.45.0 // indirect
    golang.org/x/sys v0.47.0 // indirect
    golang.org/x/text v0.41.0 // indirect
)
replace github.com/waozixyz/kryon/go/kryon => $kryon_abs/go/kryon
EOF
if [ -f "$kryon_abs/go/kryon/go.sum" ]; then
    cp "$kryon_abs/go/kryon/go.sum" "$work/out/go.sum"
fi

cat > "$work/out/workbook_native_test.go" <<'EOF'
package krygen

import (
	"bytes"
	"image"
	"image/color"
	"testing"

	"github.com/waozixyz/kryon/go/kryon"
)

func zeroString(buf []byte) string {
	if i := bytes.IndexByte(buf, 0); i >= 0 {
		return string(buf[:i])
	}
	return string(buf)
}

func workbookTestState() *WorkbookNativeState {
	st := &WorkbookNativeState{
		SectionCursor: 5,
		LabelCursor:   13,
		RowCursor:     3,
		UnitsCursor:   7,
		NotesCursor:   16,
		ScratchCursor: 0,
	}
	copy(st.Section[:], "banks")
	copy(st.Label[:], "demo checking")
	copy(st.RowId[:], "USD")
	copy(st.Units[:], "1250.00")
	copy(st.Notes[:], "startup snapshot")
	return st
}

func TestGeneratedWorkbookNativeInputSemantics(t *testing.T) {
	host := kryon.NewHost(kryon.AppConfig{Title: "test", Width: 1180, Height: 720, FPS: 30})
	defer host.Close()
	st := workbookTestState()
	draw := func() {
		host.Draw(func() {
			kryon.BeginFrame()
			WorkbookNative_WorkbookFrame(st)
			kryon.EndFrame()
		})
	}
	assertVisible := func(step string) {
		t.Helper()
		if got, max := len(host.FrameOps()), 24; got > max {
			t.Fatalf("%s frame op count = %d, want <= %d: %#v", step, got, max, host.FrameOps())
		}
		img := host.Render()
		if got := img.Bounds().Dx(); got != 1180 {
			t.Fatalf("%s render width = %d, want 1180", step, got)
		}
		if got := img.Bounds().Dy(); got != 720 {
			t.Fatalf("%s render height = %d, want 720", step, got)
		}
		if got := countRenderedPixels(img, rgba(kryon.RAYWHITE)); got < 4000 {
			t.Fatalf("%s render changed only %d pixels, want visible UI", step, got)
		}
	}

	draw()
	assertVisible("initial")

	host.QueueTap(150, 90)
	host.QueueText(" vault")
	draw()
	if got, want := zeroString(st.Section[:]), "banks vault"; got != want {
		t.Fatalf("section after tap/type = %q, want %q", got, want)
	}

	host.QueueKey(kryon.KeyTab)
	draw()
	host.QueueText(" updated")
	draw()
	if got, want := zeroString(st.Label[:]), "demo checking updated"; got != want {
		t.Fatalf("label after tab/type = %q, want %q", got, want)
	}

	host.SetSelection(1002, 0, 4)
	host.QueueShortcut(kryon.KeyC)
	draw()
	if got, want := host.ClipboardText(), "demo"; got != want {
		t.Fatalf("clipboard = %q, want %q", got, want)
	}

	host.SetSelection(1002, 5, 13)
	host.QueueText("savings")
	draw()
	if got, want := zeroString(st.Label[:]), "demo savings updated"; got != want {
		t.Fatalf("label selection replace = %q, want %q", got, want)
	}

	host.SetSelection(1002, 0, 4)
	host.QueueShortcut(kryon.KeyX)
	draw()
	if got, want := zeroString(st.Label[:]), " savings updated"; got != want {
		t.Fatalf("label cut = %q, want %q", got, want)
	}
	if got, want := host.ClipboardText(), "demo"; got != want {
		t.Fatalf("clipboard after cut = %q, want %q", got, want)
	}

	host.SetSelection(1002, 0, 0)
	host.QueueShortcut(kryon.KeyV)
	draw()
	if got, want := zeroString(st.Label[:]), "demo savings updated"; got != want {
		t.Fatalf("label paste = %q, want %q", got, want)
	}

	host.SetFocus(1003)
	host.QueueKey(kryon.KeyLeft)
	draw()
	host.QueueKey(kryon.KeyBackspace)
	draw()
	if got, want := zeroString(st.RowId[:]), "UD"; got != want {
		t.Fatalf("row id backspace = %q, want %q", got, want)
	}
	host.QueueText("S")
	draw()
	if got, want := zeroString(st.RowId[:]), "USD"; got != want {
		t.Fatalf("row id restore = %q, want %q", got, want)
	}

	host.SetFocus(1004)
	host.QueueKey(kryon.KeyLeft)
	draw()
	host.QueueKey(kryon.KeyDelete)
	draw()
	if got, want := zeroString(st.Units[:]), "1250.0"; got != want {
		t.Fatalf("units delete = %q, want %q", got, want)
	}
	host.QueueText("0")
	draw()
	if got, want := zeroString(st.Units[:]), "1250.00"; got != want {
		t.Fatalf("units restore = %q, want %q", got, want)
	}

	host.SetFocus(1005)
	host.SetSelection(1005, 8, 16)
	host.QueueText("notes")
	draw()
	if got, want := zeroString(st.Notes[:]), "startup notes"; got != want {
		t.Fatalf("notes selection replace = %q, want %q", got, want)
	}

	host.SetFocus(1006)
	for i := 0; i < 1200; i++ {
		host.QueueText("x")
		draw()
		if i%200 == 0 {
			assertVisible("scratch stress")
		}
	}
	if got, want := len(zeroString(st.Scratch[:])), 1200; got != want {
		t.Fatalf("scratch typed length = %d, want %d", got, want)
	}
	if got, want := st.ScratchCursor, int32(1200); got != want {
		t.Fatalf("scratch cursor = %d, want %d", got, want)
	}

	host.QueueTap(40, 496)
	draw()
	if got, want := st.Action, int32(1); got != want {
		t.Fatalf("action after save = %d, want %d", got, want)
	}

	host.QueueTap(144, 496)
	draw()
	if got, want := st.Action, int32(11); got != want {
		t.Fatalf("action after add row = %d, want %d", got, want)
	}

	assertVisible("final")
}

func countRenderedPixels(img interface {
	Bounds() image.Rectangle
	RGBAAt(int, int) color.RGBA
}, bg color.RGBA) int {
	count := 0
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if img.RGBAAt(x, y) != bg {
				count++
			}
		}
	}
	return count
}

func rgba(c kryon.Color) color.RGBA {
	return color.RGBA{R: c.R, G: c.G, B: c.B, A: c.A}
}
EOF

(cd "$work/out" && GOCACHE="${GOCACHE:-$work/go-cache}" CGO_ENABLED=0 go test ./...)
printf '{"workbook_native_ui_smoke":"ok","source":"%s"}\n' "$source_file"
