.PHONY: help install infra-up infra-down collector processor api start-all stop-all clean test

help:
	@echo "Market System - Makefile Commands"
	@echo ""
	@echo "Installation:"
	@echo "  make install      - Install dependencies"
	@echo ""
	@echo "Infrastructure:"
	@echo "  make infra-up     - Start infrastructure (Kafka, Redis, InfluxDB)"
	@echo "  make infra-down   - Stop infrastructure"
	@echo ""
	@echo "Services:"
	@echo "  make collector    - Start collector service"
	@echo "  make processor    - Start processor service"
	@echo "  make api          - Start API service"
	@echo ""
	@echo "All Services:"
	@echo "  make start-all    - Start all services"
	@echo "  make stop-all     - Stop all services"
	@echo ""
	@echo "Development:"
	@echo "  make test         - Run tests"
	@echo "  make clean        - Clean build artifacts and logs"

install:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy
	@echo "✓ Dependencies installed"

infra-up:
	@echo "Starting infrastructure..."
	cd deploy && docker-compose up -d
	@echo "Waiting for services to be ready..."
	@sleep 10
	@echo "✓ Infrastructure started"
	@echo ""
	@echo "Service URLs:"
	@echo "  - Kafka UI:        http://localhost:8090"
	@echo "  - Redis Commander: http://localhost:8091"
	@echo "  - InfluxDB:        http://localhost:8086"

infra-down:
	@echo "Stopping infrastructure..."
	cd deploy && docker-compose down
	@echo "✓ Infrastructure stopped"

collector:
	@echo "Starting Collector Service..."
	go run services/collector/cmd/main.go -config configs/collector.json

processor:
	@echo "Starting Processor Service..."
	go run services/processor/cmd/main.go -config configs/processor.json

api:
	@echo "Starting API Service..."
	go run services/api/cmd/main.go -f services/api/etc/market-api.yaml

start-all:
	@echo "Starting all services..."
	@bash deploy/scripts/start-all.sh

stop-all:
	@echo "Stopping all services..."
	@bash deploy/scripts/stop-all.sh

test:
	@echo "Running tests..."
	go test -v ./...

clean:
	@echo "Cleaning..."
	rm -rf logs/*.log
	rm -rf logs/*.pid
	@echo "✓ Cleaned"

# 构建所有服务
build:
	@echo "Building all services..."
	@mkdir -p bin
	go build -o bin/collector services/collector/cmd/main.go
	go build -o bin/processor services/processor/cmd/main.go
	go build -o bin/api services/api/cmd/main.go
	@echo "✓ Build complete"

# 生成 API 文档
api-doc:
	@echo "Generating API documentation..."
	goctl api go -api services/api/market.api -dir services/api

# Docker 构建
docker-build:
	@echo "Building Docker images..."
	docker build -t market-collector:latest -f deploy/Dockerfile.collector .
	docker build -t market-processor:latest -f deploy/Dockerfile.processor .
	docker build -t market-api:latest -f deploy/Dockerfile.api .
	@echo "✓ Docker images built"
