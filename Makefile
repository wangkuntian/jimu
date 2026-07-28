.PHONY: run build migrate swagger cli docker

run:
	go run cmd/server/main.go

build:
	go build -o bin/server cmd/server/main.go

migrate:
	go run cmd/cli/main.go migrate

swagger:
	swag init -g cmd/server/main.go -o docs/openapi

cli:
	go build -o bin/jimu cmd/cli/main.go

docker:
	docker build -t jimu:latest .
