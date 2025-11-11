#!/bin/bash

# 启动行情系统所有服务

echo "========================================"
echo "Starting Market System"
echo "========================================"

# 检查是否在项目根目录
if [ ! -f "go.mod" ]; then
    echo "Error: Please run this script from the project root directory"
    exit 1
fi

# 启动基础设施
echo ""
echo "Starting infrastructure..."
cd deploy
docker-compose up -d
cd ..

echo "Waiting for services to be ready..."
sleep 10

# 检查 Kafka 是否就绪
echo "Checking Kafka..."
until docker exec kafka kafka-topics --bootstrap-server localhost:9092 --list &> /dev/null
do
    echo "Waiting for Kafka to be ready..."
    sleep 2
done
echo "✓ Kafka is ready"

# 检查 Redis 是否就绪
echo "Checking Redis..."
until docker exec redis redis-cli ping &> /dev/null
do
    echo "Waiting for Redis to be ready..."
    sleep 2
done
echo "✓ Redis is ready"

# 创建日志目录
mkdir -p logs

# 启动采集服务
echo ""
echo "Starting Collector Service..."
nohup go run services/collector/cmd/main.go -config configs/collector.json > logs/collector.log 2>&1 &
COLLECTOR_PID=$!
echo "✓ Collector Service started (PID: $COLLECTOR_PID)"

# 等待一下
sleep 3

# 启动处理服务
echo ""
echo "Starting Processor Service..."
nohup go run services/processor/cmd/main.go -config configs/processor.json > logs/processor.log 2>&1 &
PROCESSOR_PID=$!
echo "✓ Processor Service started (PID: $PROCESSOR_PID)"

# 等待一下
sleep 3

# 启动 API 服务
echo ""
echo "Starting API Service..."
nohup go run services/api/cmd/main.go -f services/api/etc/market-api.yaml > logs/api.log 2>&1 &
API_PID=$!
echo "✓ API Service started (PID: $API_PID)"

# 保存 PID
echo $COLLECTOR_PID > logs/collector.pid
echo $PROCESSOR_PID > logs/processor.pid
echo $API_PID > logs/api.pid

echo ""
echo "========================================"
echo "All services started successfully!"
echo "========================================"
echo ""
echo "Service URLs:"
echo "  - API Service:        http://localhost:8080"
echo "  - Kafka UI:           http://localhost:8090"
echo "  - Redis Commander:    http://localhost:8091"
echo "  - InfluxDB:           http://localhost:8086"
echo ""
echo "Logs:"
echo "  - Collector:  logs/collector.log"
echo "  - Processor:  logs/processor.log"
echo "  - API:        logs/api.log"
echo ""
echo "To stop all services, run: ./deploy/scripts/stop-all.sh"
echo ""
