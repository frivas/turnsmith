.PHONY: build test check fmt bench release

build:
	go build -trimpath -o bin/scramble ./cmd/scramble

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
	go run ./scripts/release.go --version 0.1.0
