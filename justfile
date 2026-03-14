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

db:
    docker compose up -d postgres

db-stop:
    docker compose down

db-reset:
    docker compose down -v
    docker compose up -d postgres

db-logs:
    docker compose logs -f postgres

db-shell:
    docker exec -it courselite_db psql -U courselite -d courselite

profile-load:
    hey -z 30s -c 50 http://localhost:3000/api/health

profile-cpu:
    @echo "Capturing 30s CPU profile — make sure the server is running"
    go tool pprof -http=:6060 -seconds 30 http://localhost:3000/debug/pprof/profile

profile-mem:
    go tool pprof -http=:6060 http://localhost:3000/debug/pprof/heap
