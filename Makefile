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

.PHONY: fmt
fmt:
	go run ./cmd/testanalyser -c 0 -localImportPrefix github.com/NonLogicalDev/go.fancyimports -fix ./...

.PHONY: example
example:
	find _example -iname '*.go' | grep -v 'result' | sed 's/.go//' | xargs -n1 sh -c 'go run ./cmd/gofancyimports -- $$1.go > $$1.result.go' --
