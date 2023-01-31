export prefix?=$(HOME)/.local
export bindir?=$(prefix)/bin
export binary=gofancyimports

GORELEASER=./scripts/goreleaser.wrap

.PHONY: build
build:
	$(GORELEASER) build --rm-dist --snapshot --single-target --output "./dist/$(binary)"

.PHONY: release
release:
	$(GORELEASER) release --rm-dist --skip-publish

.PHONY: install
install: build
	cp "dist/$(binary)" "$(bindir)"

.PHONY: clean
clean:
	rm -rf dist
