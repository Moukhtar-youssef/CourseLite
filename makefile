install:
	cd client && bun install

build: install
	go build -o server ./cmd/api/main.go
	cd client && bun run build

run: build
	./server

dev:
	@echo "Starting Go backend on :8080 and Vite dev server on :5173"
	@trap 'kill 0' SIGINT; \
	  ( air) & \
	  (cd client && bun run build -- --watch) & \
	  wait

