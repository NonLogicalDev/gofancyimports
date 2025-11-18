export prefix?=$(HOME)/.local
export bindir?=$(prefix)/bin
export binary=gofancyimports

GORELEASER=./scripts/goreleaser.wrap

.PHONY: build
build:
	$(GORELEASER) build --clean --snapshot --single-target --output "./dist/$(binary)"

.PHONY: build-all
build-all:
	$(GORELEASER) build --clean --snapshot --output "./dist/$(binary)"

.PHONY: install
install: build
	install -m 755 "dist/$(binary)" "$(bindir)"
