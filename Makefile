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
	@docker build -t temp . || (echo "Build failed no container was run" && exit 1)
	@docker run -d --name temp-container -e STORE_BACKEND=in_memory -p 8080:8080 temp
	@echo "Executing benchmark execution matrix..."
	-go run ./test \
	   -clients $(CLIENTS) \
	   -iterations $(ITERATIONS) \
	   -clients_min_reqs $(MIN_REQS) \
	   -clients_max_reqs $(MAX_REQS) \
	   -clients_min_cooldown $(MIN_COOLDOWN) \
	   -clients_max_cooldown $(MAX_COOLDOWN)
	$(MAKE) clean

test_redis:
	@echo "Configuring engine for 'redis' processing..."
	@docker compose up --build -d || (echo "Docker compose build up failed" && docker compose down -v && exit 1)
	@echo "Executing benchmark execution matrix..."
	-go run ./test \
	   -clients $(CLIENTS) \
	   -iterations $(ITERATIONS) \
	   -clients_min_reqs $(MIN_REQS) \
	   -clients_max_reqs $(MAX_REQS) \
	   -clients_min_cooldown $(MIN_COOLDOWN) \
	   -clients_max_cooldown $(MAX_COOLDOWN)
	$(MAKE) clean

clean:
	go clean
	-docker rm -f temp-container
	-docker compose down -v
	-docker image prune -f