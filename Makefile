.PHONY: build test test-v fuzz check vet lint repl all build-wasm wasm-exec web serve check-wasm

# Build & Run
build :
	go build ./...

repl :
	go run ./cmd/repl

# WASM & Web
build-wasm :
	GOOS=js GOARCH=wasm go build -o web/nixy.wasm ./cmd/web

wasm-exec :
	cp "$$(go env GOROOT)/lib/wasm/wasm_exec.js" web/

web : wasm-exec build-wasm

serve : web
	cd web && python3 -m http.server 8080

check-wasm :
	GOOS=js GOARCH=wasm go build -o /dev/null ./cmd/web

# Testing
test :
	go test -p 1 ./...

test-v :
	go test -p 1 -v ./...

fuzz :
	go test -p 1 -count=10 ./pkg/game/quests/ -run TestFuzz
	go test -p 1 -count=10 ./pkg/session/ -run TestFuzzE2E

# Quality
check :
	staticcheck ./...

vet :
	go vet ./...

lint : check vet

# Convenience
all : build lint test
