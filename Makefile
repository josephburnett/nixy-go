.PHONY: check
check :
	staticcheck ./...

repl :
	go run ./cmd/repl
