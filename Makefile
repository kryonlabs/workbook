.DEFAULT_GOAL := build

# Workbook is a Ziran package. The Ziran toolchain and Kryon come from
# ziran.lock; for local development an ignored ziran.local.toml can map them
# to sibling checkouts.
ZIRAN ?= ziran
CC ?= cc
PREFIX ?= $(HOME)/.local
VERSION ?= 0.1.0
DIST_DIR ?= dist

APP := build/workbook-desktop
CELL := build/cell
TESTS := source_audit_test structure_test conversion_test desktop_test parity_test
TEST_PROGRAMS := $(TESTS:%=build/%)
SOURCES := $(wildcard src/*.zi) ziran.toml ziran.lock

# Programs that read their arguments from main(argc, argv) build without
# -pedantic-errors: C wants that signature spelled with char, and signed char
# has the same layout.
NATIVE_FLAGS := -std=c99 -O2 -ffunction-sections -fdata-sections -Wl,--gc-sections

.PHONY: build cell check run test test-ci structure-test conversion-test desktop-test parity audit install deb clean

build: $(APP) $(CELL)

# The desktop editor: Ziran generates C for Kryon's SDL and Cairo host.
$(APP): $(SOURCES)
	@$(ZIRAN) tool Kryon build

# A headless program: Ziran generates C for one entry, then cc links it.
# $(call program,entry,source,output)
define program
	@$(ZIRAN) build --project --target=c --entry $(1):main -o $(3)-c $(2)
	@$(CC) $(NATIVE_FLAGS) -I$(shell $(ZIRAN) pkg path ziran)/include -I$(3)-c \
		$(3)-c/*.c -lm -o $(3)
endef

# The engine driver, and the test programs, which are Ziran too.
cell: $(CELL)

$(CELL): $(SOURCES)
	$(call program,cell,src/cell.zi,$(CELL))

$(TEST_PROGRAMS): build/%: tests/%.zi $(SOURCES) $(wildcard tests/*.zi)
	$(call program,$*,tests/$*.zi,$@)

check:
	@$(ZIRAN) tool Kryon check
	@$(ZIRAN) check --project src/cell.zi
	@for test in $(TESTS); do $(ZIRAN) check --project tests/$$test.zi || exit 1; done

run:
	@$(ZIRAN) tool Kryon run

audit: build/source_audit_test
	@build/source_audit_test

# The desktop test runs the app on a private Xvfb display and drives it with
# xdotool. Parity compares every fixture with Gnumeric; the newest catalog
# functions need a Gnumeric master build (see tests/parity_test.zi), which a
# stock CI runner lacks, so CI runs test-ci and the full gate is `make test`.
desktop-test: build build/desktop_test
	@env -u DISPLAY -u WAYLAND_DISPLAY -u XAUTHORITY build/desktop_test

test-ci: build audit structure-test conversion-test desktop-test

test: test-ci parity

parity: cell build/parity_test
	@build/parity_test

# Installs the editor, its desktop entry, the geld finance profile, and cell
# under PREFIX (default ~/.local). geld is a link to workbook that opens the
# geld profile.
install: build
	@$(ZIRAN) install --prefix $(PREFIX)
	@mkdir -p $(PREFIX)/bin $(PREFIX)/share/applications
	@install -m 0755 $(CELL) $(PREFIX)/bin/cell
	@ln -sf workbook $(PREFIX)/bin/geld
	@sed 's|^Exec=geld$$|Exec=$(PREFIX)/bin/geld|' packaging/geld.desktop \
		> $(PREFIX)/share/applications/geld.desktop

# A Debian package of the same files.
DEB_NAME := workbook_$(VERSION)_amd64
DEB_ROOT := $(DIST_DIR)/$(DEB_NAME)
define DEB_CONTROL
Package: workbook
Version: $(VERSION)
Section: utils
Priority: optional
Architecture: amd64
Maintainer: Waozi <waozi@proton.me>
Installed-Size: INSTALLED_SIZE
Depends: libc6, libsdl2-2.0-0, libcairo2
Homepage: https://github.com/kryonlabs/workbook
Description: Spreadsheet editor compatible with Gnumeric
 Workbook is a native spreadsheet editor written in Ziran with Kryon. It reads
 and writes Gnumeric and CSV files and evaluates Gnumeric formulas; the cell
 command does the same without a window. Installed profile commands such as
 geld start specialized workbook profiles with their own data directories.
endef
export DEB_CONTROL

deb: build
	rm -rf $(DEB_ROOT) $(DIST_DIR)/$(DEB_NAME).deb
	install -D -m 0755 $(APP) $(DEB_ROOT)/usr/bin/workbook
	install -D -m 0755 $(CELL) $(DEB_ROOT)/usr/bin/cell
	ln -s workbook $(DEB_ROOT)/usr/bin/geld
	install -D -m 0644 packaging/workbook.desktop $(DEB_ROOT)/usr/share/applications/workbook.desktop
	install -D -m 0644 packaging/geld.desktop $(DEB_ROOT)/usr/share/applications/geld.desktop
	install -D -m 0644 README.md $(DEB_ROOT)/usr/share/doc/workbook/README.md
	install -D -m 0644 workbook.example.json $(DEB_ROOT)/usr/share/doc/workbook/workbook.example.json
	mkdir -p $(DEB_ROOT)/DEBIAN
	printf '%s\n' "$$DEB_CONTROL" | sed "s/INSTALLED_SIZE/$$(du -sk $(DEB_ROOT)/usr | cut -f1)/" \
		> $(DEB_ROOT)/DEBIAN/control
	dpkg-deb --build --root-owner-group $(DEB_ROOT) $(DIST_DIR)/$(DEB_NAME).deb

clean:
	rm -rf build

structure-test: build/structure_test
	@build/structure_test

conversion-test: build/conversion_test
	@build/conversion_test
