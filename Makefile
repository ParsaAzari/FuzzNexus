# ==============================================================================
# FuzzNexus Makefile
# Developed by Parsa Azari (https://parsaazari.com)
# ==============================================================================

.PHONY: all release dmg mac win windows linux test testlab fmt vet build lint vuln clean check install

all: release

release:
	@./scripts/build-release.sh all

dmg: mac
mac:
	@./scripts/build-release.sh mac

install:
	@./scripts/build-release.sh mac --install

win: windows
windows:
	@./scripts/build-release.sh win

linux:
	@./scripts/build-release.sh linux

test:
	@echo "==> Running frontend tests..."
	@cd fuzznexus-desktop/frontend && npm test
	@echo "==> Running root Go tests..."
	@go test -race -count=1 ./...
	@echo "==> Running desktop Go tests..."
	@cd fuzznexus-desktop && go test -race -count=1 ./...

testlab:
	@go test -race -tags=testlab -count=1 ./testlab/...

fmt:
	@files="$$(gofmt -l .)"; test -z "$$files" || { echo "Unformatted Go files:"; echo "$$files"; exit 1; }

vet:
	@go vet ./...
	@cd fuzznexus-desktop && go vet ./...

build:
	@go build ./...
	@cd fuzznexus-desktop && go build ./...

lint:
	@golangci-lint run
	@cd fuzznexus-desktop && golangci-lint run --config ../.golangci.yml

vuln:
	@govulncheck ./...
	@cd fuzznexus-desktop && govulncheck ./...

check:
	@$(MAKE) fmt
	@$(MAKE) build
	@$(MAKE) vet
	@$(MAKE) test

clean:
	@rm -rf dist/
	@rm -rf fuzznexus-desktop/build/bin/
	@rm -rf fuzznexus-desktop/frontend/dist/
	@rm -f testlab.test
	@echo "Cleaned all old build artifacts."
