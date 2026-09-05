#!/usr/bin/env sh
set -eu

version="${1:-0.1.0}"
dist_dir="${2:-dist}"
arch="amd64"
pkg="workbook"
root="${dist_dir}/${pkg}_${version}_${arch}"
deb="${dist_dir}/${pkg}_${version}_${arch}.deb"

rm -rf "$root" "$deb"
mkdir -p \
  "$root/DEBIAN" \
  "$root/usr/bin" \
  "$root/usr/share/applications" \
  "$root/usr/share/doc/workbook"

install -m 0755 workbook "$root/usr/bin/workbook"
install -m 0755 scripts/cell "$root/usr/bin/cell"
install -m 0755 scripts/geld "$root/usr/bin/geld"
install -m 0644 packaging/workbook.desktop "$root/usr/share/applications/workbook.desktop"
install -m 0644 packaging/geld.desktop "$root/usr/share/applications/geld.desktop"
install -m 0644 README.md "$root/usr/share/doc/workbook/README.md"
install -m 0644 workbook.example.json "$root/usr/share/doc/workbook/workbook.example.json"

installed_size="$(du -sk "$root/usr" | awk '{print $1}')"
cat > "$root/DEBIAN/control" <<EOF
Package: workbook
Version: ${version}
Section: utils
Priority: optional
Architecture: ${arch}
Maintainer: Waozi <waozi@proton.me>
Installed-Size: ${installed_size}
Depends: libc6, libsdl2-2.0-0, libgl1, libgtk-3-0, libssl3, zlib1g, libbrotli1, libzstd1, libasound2, libpulse0, libsamplerate0, libx11-6
Homepage: https://github.com/kryonlabs/workbook
Description: Native workbook tracker
 Workbook is a standalone Kry spreadsheet-style workbook. Installed profile
 commands such as geld start specialized workbook profiles with their own data
 directories and automation scripts.
EOF

dpkg-deb --build --root-owner-group "$root" "$deb"
printf '%s\n' "$deb"
