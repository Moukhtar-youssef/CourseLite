// Package server is a package that contain the main routes and the main struct
// for server
package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/Moukhtar-youssef/CourseLite/internal/handlers"
	"github.com/Moukhtar-youssef/CourseLite/internal/middleware"
	ratelimiter "github.com/Moukhtar-youssef/CourseLite/pkg/rateLimiter"
	"github.com/Moukhtar-youssef/CourseLite/pkg/rateLimiter/local"
	redisrl "github.com/Moukhtar-youssef/CourseLite/pkg/rateLimiter/redis"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/redis/go-redis/v9"
)

func skipHealthChecks(r *http.Request) bool {
	return r.URL.Path == "/api/health"
}

func buildRateLimiter() ratelimiter.RateLimiter {
	cfg := ratelimiter.Config{
		Limit:     100,
		Window:    1 * time.Minute,
		KeyPrefix: "rl:global:",
	}

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		log.Println("REDIS_URL not set using local (in-memory) rate limiter")
		return local.New(cfg)
	}

	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Fatalf("invalid REDIS_URL: %v", err)
	}

	client := redis.NewClient(opt)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		log.Fatalf("cannot connect to Redis: %v", err)
	}

	log.Printf("Using Redis rate limiter (%s)", redisURL)
	return redisrl.New(cfg, client)
}

func (s *Server) RegisterRoutes(staticDir string) http.Handler {
	rl := buildRateLimiter()
	defer rl.Close()

	loginLimiter := local.New(ratelimiter.Config{
		Limit:     5,
		Window:    15 * time.Minute,
		KeyPrefix: "login:",
	})
	defer loginLimiter.Close()

	r := chi.NewRouter()

	r.Use(cors.AllowAll().Handler)
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(middleware.RateLimit(rl, middleware.Options{
		KeyFunc:  middleware.KeyByIP,
		SkipFunc: skipHealthChecks,
	}))

	authHandler := &handlers.AuthHandler{
		DB:            s.Db,
		AccessSecret:  s.AccessSecret,
		RefreshSecret: s.RefreshSecret,
	}
	courseHandler := &handlers.CourseHandler{
		DB:           s.Db,
		AccessSecret: s.AccessSecret,
	}
	if os.Getenv("MODE") == "dev" {
		r.Mount("/debug", chimiddleware.Profiler())
	}

	r.Route("/api", func(r chi.Router) {
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		})
		r.Get("/hello", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"message": "hello"})
		})

		r.Route("/auth", func(r chi.Router) {
			r.Get("/sessions", authHandler.Session)
			r.Group(func(r chi.Router) {
				r.Use(middleware.RateLimit(loginLimiter, middleware.Options{
					KeyFunc: middleware.KeyByIP,
					OnLimitReached: func(w http.ResponseWriter, r *http.Request) {
						handlers.JsonMessage(w,
							"Too many login attempts. Try again in 15 minutes",
							http.StatusTooManyRequests)
					},
				}))
				r.Post("/register", authHandler.Register)
				r.Post("/login", authHandler.Login)
			})
			r.Post("/refresh", authHandler.Refresh)
			r.Post("/logout", authHandler.Logout)
			r.Post("/forgot-password", authHandler.ForgotPassword)
			r.Post("/reset-password", authHandler.ResetPassword)
		})
		r.Route("/courses", func(r chi.Router) {
			r.Get("/", courseHandler.GetAll)
			// r.Post("/")
			// r.Route("/{id}", func(r chi.Router) {
			// r.Get("/")
			// r.Put("/")
			// r.Delete("/")
			// r.Post("/publish")
			// r.Post("/sections")
			// r.Get("/progress")
		})
	})
	// r.Route("/lessons/{id}", func(r chi.Router) {
	// 	r.Post("/upload-url")
	// 	r.Post("/stream")
	// 	r.Put("/")
	// 	r.Post("/complete")
	// })
	// r.Route("/sections/{id}", func(r chi.Router) {
	// 	r.Put("/")
	// 	r.Post("/lessons")
	// })
	// })

	if staticDir != "" {
		fs := http.FileServer(http.Dir(staticDir))
		r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
			path := filepath.Join(staticDir, filepath.Clean(r.URL.Path))
			if _, err := os.Stat(path); os.IsNotExist(err) {
				http.ServeFile(w, r, filepath.Join(staticDir, "index.html"))
				return
			}
			fs.ServeHTTP(w, r)
		})
	}

	return r
}
