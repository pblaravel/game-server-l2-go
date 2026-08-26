.PHONY: tidy test test-short load build login game lint docker-up docker-down docker-smoke docker-java-up docker-java-down docker-java-logs

tidy:
	go mod tidy

test-short:
	go test ./... -short -count=1

test:
	go test ./... -count=1

load:
	go test ./internal/loginserver ./internal/gameserver -count=1 -run Load -v

api-test:
	go test ./internal/apitest ./internal/crypt -count=1

lint:
	golangci-lint run ./...

build:
	mkdir -p bin
	go build -o bin/loginserver ./cmd/loginserver
	go build -o bin/gameserver ./cmd/gameserver
	go build -o bin/smoketest ./cmd/smoketest
	go build -o bin/apirest ./cmd/apirest
	go build -o bin/apicompare ./cmd/apicompare

login:
	go run ./cmd/loginserver

game:
	go run ./cmd/gameserver

# Nested Docker / some VMs drop bridged ICC when bridge-nf-call-iptables=1.
docker-net:
	-@sysctl -w net.bridge.bridge-nf-call-iptables=0 net.bridge.bridge-nf-call-ip6tables=0 >/dev/null 2>&1 || true

docker-up: docker-net
	docker compose up -d --build postgres loginserver gameserver

docker-down:
	docker compose down

docker-smoke: docker-up
	docker compose --profile test run --rm smoketest

# Java reference servers (MariaDB + login + gameserver). Ports 2107/9015/7778.
docker-java-up: docker-net
	docker compose -f docker-compose.java.yml up -d --build

docker-java-down:
	docker compose -f docker-compose.java.yml down

docker-java-logs:
	docker compose -f docker-compose.java.yml logs -f
