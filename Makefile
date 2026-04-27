.PHONY: up down proto tidy run-task run-scheduler run-agent test

# ── 基础设施 ─────────────────────────────────────────────────
up:
	docker-compose up -d
	@echo "✅ MySQL:3306  Redis:6379  Kafka:9092  Grafana:3000  Jaeger:16686"

down:
	docker-compose down

logs:
	docker-compose logs -f

# ── Proto 代码生成 ────────────────────────────────────────────
proto:
	@which protoc > /dev/null || (echo "❌ install protoc first" && exit 1)
	protoc --go_out=. --go_opt=paths=source_relative \
		   --go-grpc_out=. --go-grpc_opt=paths=source_relative \
		   proto/scheduler.proto
	@echo "✅ proto generated"

# ── 依赖管理 ─────────────────────────────────────────────────
tidy:
	cd task-service && go mod tidy
	cd scheduler-service && go mod tidy
	cd agent-service && go mod tidy

# ── 启动服务 ─────────────────────────────────────────────────
run-task:
	cd task-service && go run main.go

run-scheduler:
	cd scheduler-service && go run main.go

run-agent:
	cd agent-service && go run main.go

# ── 测试 ─────────────────────────────────────────────────────
test:
	cd task-service && go test ./... -v -race
	cd scheduler-service && go test ./... -v -race
	cd agent-service && go test ./... -v -race

# ── 数据竞争检测（面试能说出来是加分项） ──────────────────────
race:
	cd scheduler-service && go test ./... -race -count=1

# ── 代码检查 ─────────────────────────────────────────────────
lint:
	@which golangci-lint > /dev/null || go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	golangci-lint run ./...
