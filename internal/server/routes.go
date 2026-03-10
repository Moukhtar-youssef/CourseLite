package server

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"github.com/Moukhtar-youssef/CourseLite/internal/auth"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func (s *Server) RegisterRoutes(STATICDIR string) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Use(middleware.Recoverer)

	fs := http.FileServer(http.Dir(STATICDIR))
	r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		path := filepath.Join(STATICDIR, r.URL.Path)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			http.ServeFile(w, r, filepath.Join(STATICDIR, "index.html"))
			return
		}
		fs.ServeHTTP(w, r)
	})

	authHandler := &auth.Handler{}

	apiRouter := chi.NewRouter()
	apiRouter.Get("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"message": "hello"})
	})
	apiRouter.Post("/auth/register", authHandler.Register)
	apiRouter.Post("/api/auth/register", authHandler.Register)
	apiRouter.Post("/api/auth/login", authHandler.Login)
	apiRouter.Post("/api/auth/refresh", authHandler.Refresh)
	apiRouter.Post("/api/auth/logout", authHandler.Logout)
	apiRouter.Post("/api/auth/forgot-password", authHandler.ForgotPassword)
	apiRouter.Post("/api/auth/reset-password", authHandler.ResetPassword)

	r.Mount("/api", apiRouter)

	return r
}
