.PHONY:
.SILENT:
.DEFAULT_GOAL := info

info:
	echo "Blockchain"

run/client:
	go run cmd/client/main.go

run/node:
	go run cmd/node/main.go

build:
	go build -o bin/client cmd/client/main.go
	go build -o bin/node cmd/node/main.go
	go build -o bin/mainbc mainbc.go
