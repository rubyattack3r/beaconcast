.PHONY: build test vet lint vulncheck fmt-check licenses licenses-check check examples

build:
	mkdir -p bin
	go build -trimpath -o bin/beaconcast ./cmd/beaconcast

test:
	go test ./...

vet:
	go vet ./...

# golangci-lint's default linters: errcheck, govet, ineffassign, staticcheck,
# and unused.
lint:
	golangci-lint run ./...
	actionlint

# Needs network access to the Go vulnerability database.
vulncheck:
	govulncheck ./...

fmt-check:
	@test -z "$$(gofmt -l .)" || { gofmt -l .; exit 1; }

# Copy the license texts of linked dependencies into licenses/.
licenses:
	rm -rf licenses
	sh scripts/licenses.sh

licenses-check:
	sh scripts/licenses.sh --check

check: fmt-check vet lint licenses-check test build

# Regenerate the example renders in examples/.
examples: build
	for mode in dark light; do \
		bin/beaconcast render --mode $$mode \
			--svg examples/situational-awareness-$$mode.svg \
			--gif examples/situational-awareness-$$mode.gif \
			examples/situational-awareness.toml; \
	done
