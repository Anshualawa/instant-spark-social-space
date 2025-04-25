
package handlers

import (
	"chat-app/internal/config"
	"chat-app/internal/models"
	"chat-app/internal/store"
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func Register(w http.ResponseWriter, r *http.Request) {
	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Email == "" || req.Password == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	var count int
	err := config.DB.QueryRow("SELECT COUNT(*) FROM users WHERE email = ?", req.Email).Scan(&count)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	if count > 0 {
		http.Error(w, "Email is already in use", http.StatusBadRequest)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Error hashing password", http.StatusInternalServerError)
		return
	}

	userID := uuid.New().String()

	_, err = config.DB.Exec(`
		INSERT INTO users (id, username, email, password, is_online, last_seen, created_at)
		VALUES (?, ?, ?, ?, true, NOW(), NOW())
	`, userID, req.Username, req.Email, string(hashedPassword))
	if err != nil {
		http.Error(w, "Error creating user", http.StatusInternalServerError)
		return
	}

	var user models.User
	var avatar sql.NullString
	
	err = config.DB.QueryRow(`
		SELECT id, username, email, avatar, is_online, last_seen, created_at 
		FROM users WHERE id = ?
	`, userID).Scan(
		&user.ID, &user.Username, &user.Email, &avatar,
		&user.IsOnline, &user.LastSeen, &user.CreatedAt,
	)
	if err != nil {
		http.Error(w, "Error retrieving user", http.StatusInternalServerError)
		return
	}

	if avatar.Valid {
		user.Avatar = avatar.String
	}

	token := generateToken(user.ID)

	response := models.LoginResponse{
		User:  user,
		Token: token,
	}
	
	// Add user to the in-memory store for WebSocket
	store.AddUser(user)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	var user models.User
	var hashedPassword string
	var avatar sql.NullString
	
	err := config.DB.QueryRow(`
		SELECT id, username, email, password, avatar, is_online, last_seen, created_at 
		FROM users WHERE email = ?
	`, req.Email).Scan(
		&user.ID, &user.Username, &user.Email, &hashedPassword, &avatar,
		&user.IsOnline, &user.LastSeen, &user.CreatedAt,
	)
	if err != nil {
		http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}

	if avatar.Valid {
		user.Avatar = avatar.String
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(req.Password)); err != nil {
		http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}

	_, err = config.DB.Exec("UPDATE users SET is_online = true, last_seen = NOW() WHERE id = ?", user.ID)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	user.IsOnline = true
	user.LastSeen = time.Now()

	token := generateToken(user.ID)

	response := models.LoginResponse{
		User:  user,
		Token: token,
	}
	
	// Add user to the in-memory store for WebSocket
	store.AddUser(user)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func generateToken(userID string) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(time.Hour * 24 * 7).Unix(), // 1 week
	})

	tokenString, err := token.SignedString(config.JWTSecret)
	if err != nil {
		return ""
	}

	return tokenString
}
