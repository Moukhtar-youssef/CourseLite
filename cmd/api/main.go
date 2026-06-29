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
	"github.com/Moukhtar-youssef/CourseLite/internal/utils"
	"github.com/joho/godotenv"
)

func gracefulShutdown(
	Server *server.Server,
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

	if err := Server.HTTPServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("server forced to shutdown: %v", err)
	}
	log.Println("HTTPserver shutdown")

	dbHandler.Stop()
	log.Println("Database shutdown")

	if Server.LoginRateLimiter != nil {
		Server.LoginRateLimiter.Close()
		log.Println("Login RateLimiter shutdown")
	}
	if Server.RateLimiter != nil {
		Server.RateLimiter.Close()
		log.Println("Ratelimiter shutdown")
	}

	log.Println("server exiting")
	done <- true
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using environment variables")
	}

	startCtx, startCancel := context.WithTimeout(context.Background(),
		10*time.Second)
	defer startCancel()

	dbHandler := handlers.NewDBHandler(os.Getenv("DATABASE_URL"))

	queries, err := dbHandler.Start(startCtx)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	Server := server.NewServer(queries)

	done := make(chan bool)

	go gracefulShutdown(Server, dbHandler, done)

	// #nosec G706
	log.Printf("server starting on %s", utils.SanitizeLog(Server.HTTPServer.Addr))

	if err := Server.HTTPServer.ListenAndServe(); err != nil &&
		err != http.ErrServerClosed {
		fmt.Sprintf("http server error: %s", err)
	}

	<-done
	log.Println("graceful shutdown complete")
}
