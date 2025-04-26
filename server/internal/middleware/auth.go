
package middleware

import (
	"chat-app/internal/config"
	"chat-app/internal/models"
	"chat-app/internal/store"
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"

	"github.com/dgrijalva/jwt-go"
)

func Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "OPTIONS" {
			next.ServeHTTP(w, r)
			return
		}

		// Support token from query parameter for WebSocket connections
		tokenString := ""
		if r.URL.Path == "/ws" {
			tokenString = r.URL.Query().Get("token")
		}

		// If not in query params, check Authorization header
		if tokenString == "" {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				http.Error(w, "Missing or invalid authorization header", http.StatusUnauthorized)
				return
			}
			tokenString = strings.TrimPrefix(authHeader, "Bearer ")
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return config.JWTSecret, nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			http.Error(w, "Invalid token claims", http.StatusUnauthorized)
			return
		}

		userID, ok := claims["user_id"].(string)
		if !ok {
			http.Error(w, "Invalid user ID in token", http.StatusUnauthorized)
			return
		}

		// First check the in-memory store for the user
		user, found := store.GetUser(userID)
		if !found {
			// If not found in memory, try to fetch from database
			var dbUser models.User
			err := config.DB.QueryRow(`
				SELECT id, username, email, avatar, is_online, last_seen
				FROM users WHERE id = ?
			`, userID).Scan(
				&dbUser.ID, &dbUser.Username, &dbUser.Email, 
				&dbUser.Avatar, &dbUser.IsOnline, &dbUser.LastSeen,
			)
			
			if err != nil {
				if err == sql.ErrNoRows {
					http.Error(w, "User not found", http.StatusUnauthorized)
				} else {
					http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
				}
				return
			}
			
			// Add user to the store for future requests
			store.AddUser(dbUser)
			user = dbUser
		}

		ctx := context.WithValue(r.Context(), "user", user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
