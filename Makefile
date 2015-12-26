default: server

server:
	@go run cmd/hn2instapaper/main.go

.PHONY: server
