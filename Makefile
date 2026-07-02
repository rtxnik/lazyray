VERSION := $(shell git describe --tags --always 2>/dev/null || echo "dev")
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: build build-all install clean test install-script-check fmt check-fmt lint test-e2e smoke-real completions man reference docs vulncheck test-race

build:
	go build -ldflags "$(LDFLAGS)" -o lzr .

build-all:
	GOOS=darwin  GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/lzr-darwin-arm64 .
	GOOS=darwin  GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/lzr-darwin-amd64 .
	GOOS=linux   GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/lzr-linux-amd64 .
	GOOS=linux   GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/lzr-linux-arm64 .
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/lzr-windows-amd64.exe .
	GOOS=windows GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/lzr-windows-arm64.exe .

install:
	go install -ldflags "$(LDFLAGS)" .

test:
	go test ./...

install-script-check:
	shellcheck scripts/install.sh scripts/install-dryrun.sh
	sh scripts/install-dryrun.sh

fmt:
	gofmt -w .

check-fmt:
	@test -z "$$(gofmt -l .)" || (echo "Files not formatted:"; gofmt -l .; exit 1)

lint: check-fmt
	go vet ./...
	golangci-lint run ./...

clean:
	rm -rf dist/ lzr lzr.exe completions/ man/

## test-e2e: run the hysteria2 docker e2e suite (needs docker + XRAY_BIN whose dir has geoip.dat/geosite.dat)
test-e2e:
	cd test/e2e/hysteria2 && ./gen-cert.sh && docker compose up -d && sleep 4
	XRAY_BIN=$${XRAY_BIN:-$$(command -v xray)} go test -tags e2e ./test/e2e/hysteria2/ \
		-run TestE2E_Hysteria2_Pinned -v; \
		status=$$?; cd test/e2e/hysteria2 && docker compose down; exit $$status

## smoke-real: one-time local smoke against YOUR hysteria2 server (no secrets committed)
smoke-real:
	@test/e2e/hysteria2/smoke-real.sh

## completions: generate static shell completions into completions/
completions:
	mkdir -p completions
	go run . completion bash > completions/lzr.bash
	go run . completion zsh  > completions/_lzr
	go run . completion fish > completions/lzr.fish

## man: generate troff man pages into man/man1/
man:
	TZ=UTC LC_ALL=C go run ./tools/gen-docs --format man --out man/man1

## reference: generate the Markdown command + keybindings reference into docs/reference/
reference:
	TZ=UTC LC_ALL=C go run ./tools/gen-docs --format markdown \
		--md-out docs/reference/cli --keys-out docs/reference/keybindings.md

## docs: generate completions, man pages, and the Markdown reference
docs: completions man reference

## vulncheck: report known vulnerabilities reachable from the code (mirrors CI)
vulncheck:
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

## test-race: run the test suite with the race detector (mirrors CI)
test-race:
	go test ./... -race
