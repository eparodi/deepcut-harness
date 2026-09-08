# Harness — local-first AI engineering team

BINARY := harness

.PHONY: build test vet run clean htmx

# htmx fetches the pinned htmx release (sha256-verified) into the
# dashboard embed directory — required by //go:embed htmx/* (go-htmx
# delivery contract: no CDN at runtime, no committed library copy).
htmx:
	go run ./tools/fetchhtmx

build: htmx
	go build -o $(BINARY) .

test: htmx
	go test ./... -count=1

vet: htmx
	go vet ./...

run:
	go run .

clean:
	rm -f $(BINARY)
