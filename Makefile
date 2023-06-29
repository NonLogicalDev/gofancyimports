export prefix?=$(HOME)/.local
export bindir?=$(prefix)/bin
export binary=gofancyimports

GORELEASER=./scripts/goreleaser.wrap

.PHONY: build
build:
	$(GORELEASER) build --rm-dist --snapshot --single-target --output "./dist/$(binary)"

.PHONY: release
release:
	$(GORELEASER) release --rm-dist

.PHONY: install
install: build
	cp "dist/$(binary)" "$(bindir)"

.PHONY: clean
clean:
	rm -rf dist

.PHONY: fmt
fmt:
	go run ./cmd/testanalyser -c 0 --group-local-prefixes github.com/NonLogicalDev/gofancyimports -debug fpstv -fix ./...

.PHONY: testdata
testdata:
	find testdata -iname '*.go' | grep -v 'result' | sed 's/.go//' | xargs -n1 sh -c 'go run ./cmd/gofancyimports fix -- $$1.go > $$1.result.go' --

.PHONY: lint
lint:
	staticcheck ./...
