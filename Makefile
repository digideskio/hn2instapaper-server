default: server

server:
	@godep run cmd/hn2instapaper/main.go

.PHONY: server
