BINARY := ccm
PKG    := ./cmd/ccm
GOBIN  := $(shell go env GOPATH)/bin
PREFIX ?= /usr/local

.PHONY: all build install symlink uninstall clean test tidy fmt vet run

all: build

build:
	go build -o $(BINARY) $(PKG)

install:
	go install $(PKG)
	@echo "Installed to $(GOBIN)/$(BINARY)"
	@echo "Ensure $(GOBIN) is on your PATH, or run 'make symlink' to expose it under $(PREFIX)/bin (requires sudo)."

symlink: install
	sudo ln -sf $(GOBIN)/$(BINARY) $(PREFIX)/bin/$(BINARY)
	@echo "Symlinked $(PREFIX)/bin/$(BINARY) -> $(GOBIN)/$(BINARY)"

uninstall:
	rm -f $(GOBIN)/$(BINARY)
	sudo rm -f $(PREFIX)/bin/$(BINARY)

clean:
	rm -f $(BINARY)

test:
	go test ./...

tidy:
	go mod tidy

fmt:
	go fmt ./...

vet:
	go vet ./...

run:
	go run $(PKG)
