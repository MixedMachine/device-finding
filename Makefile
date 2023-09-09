build: check
	@echo "Building with Docker..."
	@docker build -t device-image .
	@docker image prune -f
	@echo "Build complete."

build.local:
	@echo "Building binaries..."
	@go build -o device.exe .
	@echo "Build complete."

run:
	@echo "Running with Docker..."
	@docker compose up -d
	@echo "Run complete."

run.local:
	@echo "Running locally..."
	@./device.exe
	@echo "Run complete."

full: build run

stop:
	@echo "Stopping containers..."
	@docker compose down
	@echo "Stop complete."

restart: stop full

check:
	@echo "Checking..."
	@go mod tidy
	@go fmt ./...
	@echo "Check complete."

clean:
	@echo "Cleaning..."
	@rm device.exe
	@docker rmi device-image
	@echo "Clean complete."

.PHONY: build build.local run run.local full stop restart check clean

