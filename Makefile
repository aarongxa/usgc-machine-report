MODULE := github.com/aarongxa/usgc-machine-report
BIN    := machine-report

.PHONY: build test run clean tidy

build:
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o $(BIN) .

test:
	CGO_ENABLED=0 go test ./...

tidy:
	go mod tidy

run: build
	./$(BIN) --no-color

clean:
	rm -f $(BIN) $(BIN).exe

# Cross-compile static binaries
dist:
	mkdir -p dist
	CGO_ENABLED=0 GOOS=linux  GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o dist/$(BIN)-linux-amd64 .
	CGO_ENABLED=0 GOOS=linux  GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o dist/$(BIN)-linux-arm64 .
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o dist/$(BIN)-darwin-amd64 .
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o dist/$(BIN)-darwin-arm64 .
