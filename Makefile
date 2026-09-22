CLIENTS ?= 50
DURATION ?= 600
INTERVAL ?= 200
.PHONY: help test_in_memory test_redis clean

help:
	@echo "Available test commands:"
	@echo "  make test_in_memory   - Runs entire stand using local memory architecture"
	@echo "  make test_redis       - Runs entire stand leveraging external Redis configuration"

test_in_memory:
	@echo "Configuring engine for 'in_memory' processing..."
	@docker build -t go-rate-limiter-stress-test . || (echo "Build failed no container was run" && exit 1)
	@docker run -d --name go-rate-limiter-stress-test -e STORE_BACKEND=in_memory -p 8080:8080 go-rate-limiter-stress-test
	@echo "Executing benchmark execution matrix..."
	-go run ./test \
	   -clients $(CLIENTS) \
	   -duration $(DURATION) \
	   -interval $(INTERVAL)
	$(MAKE) clean

test_redis:
	@echo "Configuring engine for 'redis' processing..."
	@docker compose up --build -d || (echo "Docker compose build up failed" && docker compose down -v && exit 1)
	@echo "Executing benchmark execution matrix..."
	-go run ./test \
	   -clients $(CLIENTS) \
	   -duration $(DURATION) \
	   -interval $(INTERVAL)
	$(MAKE) clean

clean:
	go clean
	-docker rm -f go-rate-limiter-stress-test
	-docker compose down -v
	-docker image prune -f