.PHONY: build test check fmt bench release

VERSION ?= 0.1.0-dev

build:
	go build -trimpath -o bin/turnsmith ./cmd/turnsmith

test:
	go test ./...

check:
	go run ./scripts/check-format.go
	go test ./...
	go vet ./...
	go test -race ./...

fmt:
	gofmt -w cmd internal scripts

bench:
	go test ./internal/app -run '^$$' -bench . -benchmem

release:
	go run ./scripts/release.go --version $(VERSION) --changelog
