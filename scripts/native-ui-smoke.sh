#!/usr/bin/env sh
set -eu

binary=${1:-./workbook}
data_dir=$(mktemp -d)
cleanup() {
  rm -rf "$data_dir"
}
trap cleanup EXIT INT TERM

set +e
XDG_DATA_HOME="$data_dir" timeout 3s xvfb-run -a "$binary" >"$data_dir/run.log" 2>&1
status=$?
set -e

if [ "$status" -ne 124 ]; then
  cat "$data_dir/run.log" >&2
  exit "$status"
fi

printf '%s\n' '{"workbook_native_ui_smoke":"ok","source":"workbook.kry"}'
