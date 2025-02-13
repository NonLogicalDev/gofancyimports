export prefix?=$(HOME)/.local
export bindir?=$(prefix)/bin
export binary=gofancyimports

GORELEASER=./scripts/goreleaser.wrap

.PHONY: update_stdlib
update_stdlib:
	-git remote remove go_x_tools
	-rm -rf internal/stdlib/go_x_stdlib
	git remote add go_x_tools https://github.com/golang/tools
	git fetch go_x_tools
	git read-tree -u --prefix=internal/stdlib/go_x_stdlib go_x_tools/master:internal/stdlib

.PHONY: build
build:
	$(GORELEASER) build --clean --snapshot --single-target --output "./dist/$(binary)"

.PHONY: release
release:
	$(GORELEASER) release --clean

.PHONY: install
install: build
	cp "dist/$(binary)" "$(bindir)"

.PHONY: test
test:
	go test ./...

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
