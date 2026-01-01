APP_NAME=go-impl-postgres-ha
CONFIG=config.yaml
IMAGE=ghcr.io/daffahilmyf/go-impl-postgres-ha

.PHONY: help fmt vet test build run-server run-consumer run-outbox migrate-up migrate-down seed docker-build docker-push kustomize-dev kustomize-prod

help:
	@echo "Targets:"
	@echo "  fmt            Run gofmt"
	@echo "  vet            Run go vet"
	@echo "  test           Run go test"
	@echo "  build          Build the binary"
	@echo "  run-server     Run API server"
	@echo "  run-consumer   Run JetStream consumer"
	@echo "  run-outbox     Run outbox worker"
	@echo "  migrate-up     Apply migrations"
	@echo "  migrate-down   Rollback last migration"
	@echo "  seed           Seed database"
	@echo "  docker-build   Build Docker image"
	@echo "  docker-push    Push Docker image"
	@echo "  kustomize-dev  Build dev manifests"
	@echo "  kustomize-prod Build prod manifests"

fmt:
	gofmt -w .

vet:
	go vet ./...

test:
	go test ./...

build:
	go build -o bin/$(APP_NAME) main.go

run-server:
	go run main.go server --config $(CONFIG)

run-consumer:
	go run main.go consumer --config $(CONFIG)

run-outbox:
	go run main.go outbox-worker --config $(CONFIG)

migrate-up:
	go run main.go migration up --config $(CONFIG)

migrate-down:
	go run main.go migration down --config $(CONFIG)

seed:
	go run main.go seed --config $(CONFIG)

docker-build:
	docker build -t $(IMAGE):latest .

docker-push:
	docker push $(IMAGE):latest

kustomize-dev:
	kustomize build kustomize/overlays/dev

kustomize-prod:
	kustomize build kustomize/overlays/prod
