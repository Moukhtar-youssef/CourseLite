// Package handlers is a package that contain the handlers for several services
// e.g. database
package handlers

import (
	"context"
	"fmt"

	DB "github.com/Moukhtar-youssef/CourseLite/internal/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DBHandler struct {
	pool  *pgxpool.Pool
	DBurl string
}

func NewDBHandler(dbURL string) *DBHandler {
	return &DBHandler{
		DBurl: dbURL,
	}
}

func (h *DBHandler) Start(ctx context.Context) (*DB.Queries, error) {
	pool, err := pgxpool.New(ctx, h.DBurl)
	if err != nil {
		return nil, fmt.Errorf("error creating postgres pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("error pinging postgres: %w", err)
	}

	h.pool = pool

	return DB.New(h.pool), nil
}

func (h *DBHandler) Stop() {
	if h.pool != nil {
		h.pool.Close()
	}
}
