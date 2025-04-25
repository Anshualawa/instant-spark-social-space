
package models

import "time"

// User model
type User struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Password  string    `json:"-"` // Hashed password, not exposed in JSON
	Avatar    string    `json:"avatar,omitempty"`
	IsOnline  bool      `json:"isOnline"`
	LastSeen  time.Time `json:"lastSeen,omitempty"`
	CreatedAt time.Time `json:"createdAt,omitempty"`
}

// Chat model
type Chat struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"` // Only for group chats
	IsGroup        bool      `json:"isGroup"`
	ParticipantIDs []string  `json:"participantIds"`
	CreatedAt      time.Time `json:"createdAt"`
}

// Message model
type Message struct {
	ID        string    `json:"id"`
	ChatID    string    `json:"chatId"`
	SenderID  string    `json:"senderId"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
	IsRead    bool      `json:"isRead"`
}

// Request/Response types
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type CreateChatRequest struct {
	ParticipantIDs []string `json:"participantIds"`
}

type CreateGroupRequest struct {
	Name           string   `json:"name"`
	ParticipantIDs []string `json:"participantIds"`
}

type SendMessageRequest struct {
	Content string `json:"content"`
}

type LoginResponse struct {
	User
	Token string `json:"token"`
}
