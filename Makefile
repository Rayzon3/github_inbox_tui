.PHONY: run build fmt tidy

run:
	go run ./cmd/github_inbox_tui

build:
	go build ./...

fmt:
	gofmt -w *.go

tidy:
	go mod tidy
