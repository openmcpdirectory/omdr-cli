.PHONY: help build test install clean release

help:
	@echo "OMDR CLI Build Commands"
	@echo ""
	@echo "  make build       - Build the CLI binary"
	@echo "  make test        - Run tests"
	@echo "  make install     - Install locally"
	@echo "  make clean       - Clean build artifacts"
	@echo "  make release     - Create a release (requires tag)"
	@echo ""

build:
	go build -o bin/omdr ./cmd/omdr

test:
	go test -v ./...

install:
	go install ./cmd/omdr

clean:
	rm -rf bin/ dist/

release:
	@if [ -z "$(VERSION)" ]; then \
		echo "Error: VERSION is required. Usage: make release VERSION=v0.1.0"; \
		exit 1; \
	fi
	git tag -a $(VERSION) -m "Release $(VERSION)"
	git push origin $(VERSION)
	@echo "Release $(VERSION) tagged and pushed. GitHub Actions will handle the rest."
