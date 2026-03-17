// Package server is a package that contain the main routes and the main struct
// for server
package server

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	DB "github.com/Moukhtar-youssef/CourseLite/internal/db"
)

type Server struct {
	port          int
	Db            *DB.Queries
	AccessSecret  string
	RefreshSecret string
}

func NewServer(db *DB.Queries) *http.Server {
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

	return &http.Server{
		Addr:         fmt.Sprintf(":%d", s.port),
		Handler:      s.RegisterRoutes(os.Getenv("STATICDIR")),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
}
