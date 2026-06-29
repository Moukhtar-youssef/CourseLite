// Package handlers contains HTTP handlers for the CourseLite service layer.
package handlers

import (
	"context"
	"fmt"

	DB "github.com/Moukhtar-youssef/CourseLite/internal/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DBHandler manages the database connection pool.
type DBHandler struct {
	pool  *pgxpool.Pool
	DBurl string
}

// NewDBHandler creates a new DBHandler with the given database URL.
func NewDBHandler(dbURL string) *DBHandler {
	return &DBHandler{
		DBurl: dbURL,
	}
}

// Start initializes the database connection pool and returns a Queries instance.
// It pings the database to verify the connection and returns an error if unsuccessful.
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

// Stop closes the database connection pool.
func (h *DBHandler) Stop() {
	if h.pool != nil {
		h.pool.Close()
	}
}
