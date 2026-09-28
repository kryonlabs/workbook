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
CELL_C := build/cell-c

.PHONY: build cell check run test test-ci desktop-test parity audit install deb clean

build: $(APP) cell

# The desktop editor: Ziran generates C for Kryon's SDL and Cairo host.
$(APP): $(wildcard src/*.zi) ziran.toml ziran.lock
	@$(ZIRAN) tool Kryon build

# The headless engine driver. It takes its arguments from main(argc, argv),
# which C's pedantic mode wants spelled with char, so it builds without
# -pedantic-errors; signed char and char share a layout.
cell: $(CELL)

$(CELL): $(wildcard src/*.zi) ziran.toml ziran.lock
	@$(ZIRAN) build --project --target=c --entry cell:main -o $(CELL_C) src/cell.zi
	@$(CC) -std=c99 -O2 -ffunction-sections -fdata-sections -Wl,--gc-sections \
		-I$(shell $(ZIRAN) pkg path ziran)/include -I$(CELL_C) \
		$(CELL_C)/*.c -lm -o $(CELL)

check:
	@$(ZIRAN) tool Kryon check
	@$(ZIRAN) check --project src/cell.zi

run:
	@$(ZIRAN) tool Kryon run

audit:
	@scripts/source-audit.sh

# The desktop test runs the app on a private Xvfb display and drives it with
# xdotool. Parity compares every fixture with Gnumeric; the newest catalog
# functions need a Gnumeric master build (see scripts/parity-test.sh), which a
# stock CI runner lacks, so CI runs test-ci and the full gate is `make test`.
desktop-test: build
	@python3 tests/desktop_test.py

test-ci: build audit desktop-test

test: test-ci parity

parity: cell
	@scripts/parity-test.sh

# Installs the editor, its desktop entry, the geld finance profile, and cell
# under PREFIX (default ~/.local).
install: build
	@$(ZIRAN) install --prefix $(PREFIX)
	@mkdir -p $(PREFIX)/bin $(PREFIX)/share/applications
	@install -m 0755 $(CELL) $(PREFIX)/bin/cell
	@install -m 0755 scripts/geld $(PREFIX)/bin/geld
	@sed 's|^Exec=geld$$|Exec=$(PREFIX)/bin/geld|' packaging/geld.desktop \
		> $(PREFIX)/share/applications/geld.desktop

deb: build
	@packaging/deb/build-deb.sh "$(VERSION)" "$(DIST_DIR)"

clean:
	rm -rf build
