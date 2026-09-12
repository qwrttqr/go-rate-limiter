CLIENTS ?= 100
ITERATIONS ?= 50
MIN_REQS ?= 10
MAX_REQS ?= 100
MIN_COOLDOWN ?= 1
MAX_COOLDOWN ?= 3

.PHONY: help test_in_memory test_redis clean

help:
	@echo "Available test commands:"
	@echo "  make test_in_memory   - Runs entire stand using local memory architecture"
	@echo "  make test_redis       - Runs entire stand leveraging external Redis configuration"

test_in_memory:
	@echo "Configuring engine for 'in_memory' processing..."
	@sed -i 's/store: ".*"/store: "in_memory"/' config.yaml
	@docker build -t temp . || (echo "Build failed no container was run"; exit 1)
	@docker run -d --name temp-container -p 8080:8080 temp
	@echo "Waiting for app environment to stabilize..."
	@sleep 3
	@echo "Executing benchmark execution matrix..."
	go run test/main.go \
		-clients $(CLIENTS) \
		-iterations $(ITERATIONS) \
		-clients_min_reqs $(MIN_REQS) \
		-clients_max_reqs $(MAX_REQS) \
		-clients_min_cooldown $(MIN_COOLDOWN) \
		-clients_max_cooldown $(MAX_COOLDOWN)
	@docker stop temp-container || true
	@docker rm temp-container || true

test_redis:
	@echo "Configuring engine for 'redis' processing..."
	@sed -i 's/store: ".*"/store: "redis"/' config.yaml
	@docker compose up --build -d || (echo "Docker compose build up failed"; docker compose down -v > /dev/null 2>&1 || true || exit 1)
	@echo "Executing benchmark execution matrix..."
	go run test/main.go \
		-clients $(CLIENTS) \
		-iterations $(ITERATIONS) \
		-clients_min_reqs $(MIN_REQS) \
		-clients_max_reqs $(MAX_REQS) \
		-clients_min_cooldown $(MIN_COOLDOWN) \
		-clients_max_cooldown $(MAX_COOLDOWN)
	@docker compose down -v

clean:
	go clean
	@docker rm -f temp-container >/dev/null 2>&1 || true
	@docker compose down -v >/dev/null 2>&1 || true
	@docker image prune -f >/dev/null 2>&1 || true