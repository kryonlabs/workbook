#!/usr/bin/env sh
set -eu

if find . -path ./vendor -prune -o -path ./build -prune -o -type f -name '*.go' -print | grep -q .; then
  echo 'application Go source remains; Workbook must be authored in Kry' >&2
  exit 1
fi

test -f workbook.kry
test -f workbook.example.json
test -f profiles/geld.json

if grep -R -n --exclude-dir=.git --exclude-dir=vendor --exclude-dir=build \
  -E 'workbook\.kry holds private|workbook\.example\.kry|profiles/\*\.kry' .; then
  echo 'data files were incorrectly described as Kry source' >&2
  exit 1
fi

printf '%s\n' '{"workbook_kry_source_audit":"ok"}'
