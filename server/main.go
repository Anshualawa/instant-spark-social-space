package main

import (
	"chat-app/internal/config"
	"chat-app/internal/handlers"
	authmdw "chat-app/internal/middleware"

	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func main() {
	// Initialize database connection
	config.InitDB()
	defer config.DB.Close()

	r := chi.NewRouter()

	// Standard middlewares.
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	
	// CORS middleware
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Public routes
	r.Group(func(r chi.Router) {
		r.Post("/api/auth/register", handlers.Register)
		r.Post("/api/auth/login", handlers.Login)
	})

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(authmdw.Auth)

		// Chat routes
		r.Get("/api/chats", handlers.GetChats)
		r.Post("/api/chats", handlers.CreateChat)
		r.Route("/api/chats/{id}", func(r chi.Router) {
			r.Get("/messages", handlers.GetChats)
			r.Post("/messages", handlers.SendMessage)
		})

		// WebSocket
		r.Get("/ws", handlers.HandleWebSocket)

		// Add new users route
		r.Get("/api/users", handlers.GetUsers)
	})

	port := 8000
	fmt.Printf("Server running on http://localhost:%d\n", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), r))
}
