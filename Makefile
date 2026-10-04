# drbrain-motor-programming - development tooling

.PHONY: help build build-frontend test run dev validate fmt vet lint generate clean verify

PORT ?= 8000

help:
	@echo "drbrain-motor-programming - available targets:"
	@echo "  build           - Build the server binary (./drbrain-motor-programming)"
	@echo "  build-frontend  - Build the SvelteKit UI and copy it into static/"
	@echo "  test            - Run all tests with the race detector"
	@echo "  run             - Build and run the server on PORT (default 8000)"
	@echo "  dev             - Run the server from source with debug logging"
	@echo "  validate        - Validate every map and its reference solution"
	@echo "  fmt             - gofmt all Go files"
	@echo "  vet             - go vet ./..."
	@echo "  lint            - golangci-lint run (requires golangci-lint)"
	@echo "  generate        - Regenerate gqlgen code from graph/schema.graphqls"
	@echo "  clean           - Remove build artifacts"
	@echo "  verify          - fmt check, vet, lint, test"

build:
	go build -o drbrain-motor-programming .

build-frontend:
	@test -d frontend/node_modules || (cd frontend && npm ci)
	cd frontend && npm run build:static

test:
	go test -race ./...

run: build
	./drbrain-motor-programming -port $(PORT)

dev:
	go run . -port $(PORT) -debug

validate:
	go run ./cmd/validate -maps-dir maps -solutions-dir solutions

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './frontend/*' -not -path './graph/generated/*')

vet:
	go vet ./...

lint:
	golangci-lint run

generate:
	go run github.com/99designs/gqlgen generate

clean:
	rm -f drbrain-motor-programming coverage.out

verify:
	@out=$$(gofmt -l $$(find . -name '*.go' -not -path './frontend/*' -not -path './graph/generated/*')); \
	if [ -n "$$out" ]; then echo "gofmt needed:"; echo "$$out"; exit 1; fi
	$(MAKE) vet
	$(MAKE) lint
	$(MAKE) test
