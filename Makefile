BINARY   := viy
CMD_DIR  := ./cmd/viy
VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT   ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE     ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS  := -s -w \
	-X github.com/oragazz0/viy/internal/version.Version=$(VERSION) \
	-X github.com/oragazz0/viy/internal/version.Commit=$(COMMIT) \
	-X github.com/oragazz0/viy/internal/version.Date=$(DATE)

.PHONY: build test lint run clean vuln

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) $(CMD_DIR)

test:
	go test -v -race -count=1 ./...

lint:
	golangci-lint run ./...

run:
	go run $(CMD_DIR)

clean:
	rm -f $(BINARY)

vuln:
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...
