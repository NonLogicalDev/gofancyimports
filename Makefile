.PHONY: gen
gen:
	go generate ./...

.PHONY: fmt
fmt:
	find . \( -iname _example -prune \) -or \( -iname '*.go' -exec gofancyimports -w {} \; \)
