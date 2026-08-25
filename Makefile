GO ?= go
BIN ?= workbook
BIN_DIR ?= $(HOME)/bin
DATA_HOME ?= $(if $(XDG_DATA_HOME),$(XDG_DATA_HOME),$(HOME)/.local/share)
DATA_DIR ?= $(DATA_HOME)/workbook
GELD_DATA_DIR ?= $(DATA_HOME)/geld
VERSION ?= 0.1.0
DIST_DIR ?= dist
KRYON_DIR := vendor/kryon
KRYON_LIB_TARGET := build/linux-x86_64/libkryon.a
RAYLIB_LIB_TARGET := build/linux-x86_64/raylib/libraylib.a
K2G_TARGET := build/linux-x86_64/bin/k2g

.PHONY: all build run test install deb clean kryon-libs kryon-tools native-ui-smoke native-runtime-audit native-binary-audit

all: build

kryon-libs:
	$(MAKE) -C $(KRYON_DIR) $(KRYON_LIB_TARGET) $(RAYLIB_LIB_TARGET)

kryon-tools:
	$(MAKE) -C $(KRYON_DIR) $(K2G_TARGET)

build:
	CGO_ENABLED=0 $(GO) build -mod=mod -buildvcs=false -o $(BIN) .

run: build
	./$(BIN)

test: native-runtime-audit native-ui-smoke native-binary-audit
	CGO_ENABLED=0 $(GO) test -mod=mod -buildvcs=false ./...

native-ui-smoke: kryon-tools
	scripts/native-ui-smoke.sh "$(KRYON_DIR)" "$(KRYON_DIR)/$(K2G_TARGET)" ui/workbook_native.kry

native-runtime-audit:
	scripts/native-runtime-audit.sh

native-binary-audit: build
	scripts/native-binary-audit.sh "$(BIN)"

install: build
	mkdir -p $(BIN_DIR)
	install -m 0755 $(BIN) $(BIN_DIR)/$(BIN)
	install -m 0755 scripts/cell $(BIN_DIR)/cell
	$(GO) run ./cmd/profile-install -profiles profiles -bin-dir "$(BIN_DIR)" -binary "$(BIN)"
	mkdir -p $(DATA_DIR)
	mkdir -p $(GELD_DATA_DIR)

deb: build
	packaging/deb/build-deb.sh "$(VERSION)" "$(DIST_DIR)"

clean:
	rm -f workbook cell geld
