client_dir := "client"
server_bin := "server"
server_entry := "./cmd/api/main.go"
compose := "docker compose"

default:
    @just --list

install:
    cd {{ client_dir }} && bun install

build: install
    go build -o {{ server_bin }} {{ server_entry }}
    cd {{ client_dir }} && bun run build

run: build
    ./{{ server_bin }}

dev:
    @echo "Starting Go backend with air and Vite dev server"
    #!/usr/bin/env bash
    trap 'kill 0' SIGINT
    air &
    cd {{ client_dir }} && bun run dev &
    wait

db:
    {{ compose }} up -d postgres

db-stop:
    {{ compose }} stop postgres

db-reset:
    {{ compose }} down -v postgres
    {{ compose }} up -d postgres

db-logs:
    {{ compose }} logs -f postgres

db-shell:
    docker exec -it courselite_db psql -U courselite -d courselite

app:
    {{ compose }} up -d

app-stop:
    {{ compose }} down

app-reset:
    {{ compose }} down -v
    {{ compose }} up -d

app-rebuild:
    {{ compose }} down
    {{ compose }} build --no-cache
    {{ compose }} up -d

app-rebuild-app:
    {{ compose }} stop app
    {{ compose }} build --no-cache app
    {{ compose }} up -d app

app-logs:
    {{ compose }} logs -f app

app-logs-all:
    {{ compose }} logs -f

profile-load:
    hey -z 30s -c 100 http://localhost:3000/api/auth/login

profile-cpu:
    @echo "Capturing 30s CPU profile — make sure the server is running"
    go tool pprof -http=:6060 -seconds 30 http://localhost:3000/debug/pprof/profile

profile-mem:
    go tool pprof -http=:6060 http://localhost:3000/debug/pprof/heap

help:
    @echo ""
    @echo "  install            Install frontend dependencies"
    @echo "  build              Build frontend + backend"
    @echo "  run                Build and run the server"
    @echo "  dev                Run air + Vite dev server"
    @echo ""
    @echo "  db                 Start postgres"
    @echo "  db-stop            Stop postgres"
    @echo "  db-reset           Wipe and restart postgres"
    @echo "  db-logs            Tail postgres logs"
    @echo "  db-shell           Open psql shell"
    @echo ""
    @echo "  app                Start all containers"
    @echo "  app-stop           Stop all containers"
    @echo "  app-reset          Wipe volumes and restart all"
    @echo "  app-rebuild        Full no-cache rebuild + restart"
    @echo "  app-rebuild-app    Rebuild only the app container"
    @echo "  app-logs           Tail app logs"
    @echo "  app-logs-all       Tail all container logs"
    @echo ""
    @echo "  profile-load       Run hey load test"
    @echo "  profile-cpu        Capture CPU profile"
    @echo "  profile-mem        Capture heap profile"
    @echo ""
