#!/bin/bash

# 停止行情系统所有服务

echo "========================================"
echo "Stopping Market System"
echo "========================================"

# 停止 Go 服务
if [ -f "logs/collector.pid" ]; then
    PID=$(cat logs/collector.pid)
    echo "Stopping Collector Service (PID: $PID)..."
    kill $PID 2>/dev/null || echo "Process not found"
    rm logs/collector.pid
fi

if [ -f "logs/processor.pid" ]; then
    PID=$(cat logs/processor.pid)
    echo "Stopping Processor Service (PID: $PID)..."
    kill $PID 2>/dev/null || echo "Process not found"
    rm logs/processor.pid
fi

if [ -f "logs/api.pid" ]; then
    PID=$(cat logs/api.pid)
    echo "Stopping API Service (PID: $PID)..."
    kill $PID 2>/dev/null || echo "Process not found"
    rm logs/api.pid
fi

# 停止基础设施
echo ""
echo "Stopping infrastructure..."
cd deploy
docker-compose down
cd ..

echo ""
echo "========================================"
echo "All services stopped"
echo "========================================"
