FROM oven/bun:1 AS frontend

WORKDIR /client

COPY client/package.json client/bun.lock ./
RUN bun ci

COPY client/ .
RUN bun run build

FROM golang:1.26-alpine AS backend

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o server ./cmd/api/main.go


FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=backend /app/server .
COPY --from=frontend /client/dist ./client/dist

EXPOSE 3000

CMD ["./server"]
