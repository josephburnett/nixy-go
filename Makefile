.PHONY: build test test-v fuzz check vet lint repl all

# Build & Run
build :
	go build ./...

repl :
	go run ./cmd/repl

# Testing
test :
	go test -p 1 ./...

test-v :
	go test -p 1 -v ./...

fuzz :
	go test -p 1 -count=10 ./pkg/game/quests/ -run TestFuzz

# Quality
check :
	staticcheck ./...

vet :
	go vet ./...

lint : check vet

# Convenience
all : build lint test
