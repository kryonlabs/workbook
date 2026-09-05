CC ?= cc
BIN ?= workbook
BIN_DIR ?= $(HOME)/bin
VERSION ?= 0.1.0
DIST_DIR ?= dist
BUILD_DIR ?= build
CODEGEN_DIR := $(BUILD_DIR)/generated
KRYON_DIR := vendor/kryon

UNAME_S := $(shell uname -s)
UNAME_M := $(shell uname -m)
KRYON_ARCH := $(if $(filter amd64,$(UNAME_M)),x86_64,$(UNAME_M))
KRYON_PLATFORM := $(if $(filter Linux,$(UNAME_S)),linux,$(shell printf '%s' '$(UNAME_S)' | tr A-Z a-z))
KRYON_BUILD_DIR := $(KRYON_DIR)/build/$(KRYON_PLATFORM)-$(KRYON_ARCH)
K2C := $(KRYON_BUILD_DIR)/bin/k2c
KRYON_LIB := $(KRYON_BUILD_DIR)/libkryon.a
RAYLIB_A := $(KRYON_BUILD_DIR)/raylib/libraylib.a

PKG_CFLAGS := $(shell pkg-config --cflags sdl2 gl x11 gtk+-3.0 libdrm gbm egl glesv2 2>/dev/null)
PKG_LIBS := $(shell pkg-config --libs sdl2 gl x11 gtk+-3.0 libdrm gbm egl glesv2 2>/dev/null)
CFLAGS ?= -O2 -Wall -Wextra -std=gnu99
CFLAGS += -I$(KRYON_DIR)/include -I$(CODEGEN_DIR) $(PKG_CFLAGS)
LDLIBS := $(KRYON_LIB) $(RAYLIB_A) $(PKG_LIBS) -lm -lpthread -ldl -lrt

.PHONY: all build run test install deb clean kryon-tools kryon-libs generate smoke

all: build

kryon-tools: $(K2C)

kryon-libs: $(KRYON_LIB) $(RAYLIB_A)

$(K2C):
	$(MAKE) -C $(KRYON_DIR) $(patsubst $(KRYON_DIR)/%,%,$@)

$(KRYON_LIB) $(RAYLIB_A):
	$(MAKE) -C $(KRYON_DIR) $(patsubst $(KRYON_DIR)/%,%,$@)

$(CODEGEN_DIR)/.generated: workbook.kry $(K2C)
	mkdir -p $(CODEGEN_DIR)
	$(K2C) --root . -o $(CODEGEN_DIR) workbook.kry
	touch $@

generate: $(CODEGEN_DIR)/.generated

build: $(BIN)

$(BIN): $(CODEGEN_DIR)/.generated $(KRYON_LIB) $(RAYLIB_A)
	$(CC) $(CFLAGS) -o $@ \
		$(CODEGEN_DIR)/workbook.c $(CODEGEN_DIR)/kryon_project.c \
		$(LDLIBS)

run: build
	./$(BIN)

test: build
	scripts/kry-source-audit.sh
	scripts/native-ui-smoke.sh ./$(BIN)

install: build
	mkdir -p $(BIN_DIR)
	install -m 0755 $(BIN) $(BIN_DIR)/$(BIN)
	install -m 0755 scripts/cell $(BIN_DIR)/cell
	install -m 0755 scripts/geld $(BIN_DIR)/geld

deb: build
	packaging/deb/build-deb.sh "$(VERSION)" "$(DIST_DIR)"

clean:
	rm -rf $(BUILD_DIR) $(BIN)
