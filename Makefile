.PHONY: clean test lint

SOURCES = $(shell find . -name \*.go)

UTILS_BIN_DIR = cmd/protonutils
UTILS_BIN_FILE = protonutils
UTILS_BIN = $(UTILS_BIN_DIR)/$(UTILS_BIN_FILE)

VERSION := $(shell ./build/version.sh)
BUILDTIME := $(shell date -u +"%Y%m%d%H%M%S")

GOLDFLAGS += -X main.Version=$(VERSION)
GOLDFLAGS += -X main.Buildtime=$(BUILDTIME)
GOFLAGS = -ldflags "$(GOLDFLAGS)"

build: $(UTILS_BIN)

$(UTILS_BIN): $(SOURCES)
	cd $(UTILS_BIN_DIR); go build $(GOFLAGS)

$(UTILS_BIN_FILE): $(UTILS_BIN)

clean:
	rm -f $(UTILS_BIN)

run: run_utils

run_utils: $(UTILS_BIN)
	./$(UTILS_BIN) $(args)

test:
	go test ./...

lint:
	golint ./...

release: GOLDFLAGS += -s -w
release: build
	upx -qq9 $(UTILS_BIN)
