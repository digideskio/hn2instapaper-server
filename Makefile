default: server

server:
	@godep go run cmd/hn2instapaper/main.go

.PHONY: server
