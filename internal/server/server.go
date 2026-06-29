// Package server provides the HTTP server setup and route registration
// for the CourseLite application.
package server

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	DB "github.com/Moukhtar-youssef/CourseLite/internal/db"
	ratelimiter "github.com/Moukhtar-youssef/CourseLite/pkg/rateLimiter"
)

// Server holds the configuration for the HTTP server.
type Server struct {
	port          int
	AccessSecret  string
	RefreshSecret string

	Db               *DB.Queries
	HTTPServer       *http.Server
	RateLimiter      ratelimiter.RateLimiter
	LoginRateLimiter ratelimiter.RateLimiter
}

// NewServer creates and configures a new HTTP server instance.
func NewServer(db *DB.Queries) *Server {
	port, _ := strconv.Atoi(os.Getenv("PORT"))
	if port == 0 {
		port = 8080
	}

	s := &Server{
		port:          port,
		Db:            db,
		AccessSecret:  os.Getenv("ACCESS_SECRET"),
		RefreshSecret: os.Getenv("REFRESH_SECRET"),
	}
	s.HTTPServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.port),
		Handler:      s.RegisterRoutes(os.Getenv("STATICDIR")),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	return s
}
