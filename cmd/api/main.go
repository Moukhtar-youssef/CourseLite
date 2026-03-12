package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Moukhtar-youssef/CourseLite/internal/handlers"
	"github.com/Moukhtar-youssef/CourseLite/internal/server"
	"github.com/joho/godotenv"
)

func gracefulShutdown(
	apiServer *http.Server,
	dbHandler *handlers.DBHandler,
	done chan bool,
) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	<-sigCh

	log.Println("shutting down gracefully, press Ctrl+C again to force")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(),
		30*time.Second)
	defer shutdownCancel()

	if err := apiServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("server forced to shutdown: %v", err)
	}
	log.Println("Server shutdown")

	dbHandler.Stop()
	log.Println("Database shutdown")

	log.Println("server exiting")
	done <- true
}

func main() {
	// .env is optional — in production, env vars come from the environment
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using environment variables")
	}

	// Startup gets its own short-lived context — separate from app lifecycle
	startCtx, startCancel := context.WithTimeout(context.Background(),
		10*time.Second)
	defer startCancel()

	dbHandler := handlers.NewDBHandler(os.Getenv("DATABASE_URL"))

	queries, err := dbHandler.Start(startCtx)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	httpServer := server.NewServer(queries)

	done := make(chan bool, 1)

	go gracefulShutdown(httpServer, dbHandler, done)

	log.Printf("server starting on %s", httpServer.Addr)

	if err := httpServer.ListenAndServe(); err != nil &&
		err != http.ErrServerClosed {
		panic(fmt.Sprintf("http server error: %s", err))
	}

	<-done
	log.Println("graceful shutdown complete")
}
