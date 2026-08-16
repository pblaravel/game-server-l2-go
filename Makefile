.PHONY: tidy test test-short load build login game

tidy:
	go mod tidy

test-short:
	go test ./... -short -count=1

test:
	go test ./... -count=1

load:
	go test ./internal/loginserver ./internal/gameserver -count=1 -run Load -v

build:
	mkdir -p bin
	go build -o bin/loginserver ./cmd/loginserver
	go build -o bin/gameserver ./cmd/gameserver

login:
	go run ./cmd/loginserver

game:
	go run ./cmd/gameserver
