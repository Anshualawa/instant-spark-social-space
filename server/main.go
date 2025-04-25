
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"golang.org/x/crypto/bcrypt"
)

// JWT secret key (in production, this should be an environment variable)
var jwtSecret = []byte("your_jwt_secret_key")

// WebSocket upgrade configuration
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins in dev (in production, restrict this)
		return true
	},
}

// In-memory data store (replace with MySQL in production)
var (
	users       = make(map[string]User)
	chats       = make(map[string]Chat)
	messages    = make(map[string][]Message)
	usersByAuth = make(map[string]User) // Maps tokens to users
	connections = make(map[string]*websocket.Conn)
	mutex       = &sync.RWMutex{}
)

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
	ID           string   `json:"id"`
	Name         string   `json:"name"` // Only for group chats
	IsGroup      bool     `json:"isGroup"`
	ParticipantIDs []string `json:"participantIds"`
	CreatedAt    time.Time `json:"createdAt"`
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

// WebSocket message type
type WSMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// Login request
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Register request
type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Create chat request
type CreateChatRequest struct {
	ParticipantIDs []string `json:"participantIds"`
}

// Create group chat request
type CreateGroupRequest struct {
	Name          string   `json:"name"`
	ParticipantIDs []string `json:"participantIds"`
}

// Send message request
type SendMessageRequest struct {
	Content string `json:"content"`
}

// Login response with token
type LoginResponse struct {
	User
	Token string `json:"token"`
}

// Main function
func main() {
	// Create router
	r := mux.NewRouter()
	
	// Auth routes
	r.HandleFunc("/api/auth/register", registerHandler).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/auth/login", loginHandler).Methods("POST", "OPTIONS")
	
	// User routes
	r.HandleFunc("/api/users", authMiddleware(getUsersHandler)).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/users/me", authMiddleware(getCurrentUserHandler)).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/users/{id}", authMiddleware(getUserHandler)).Methods("GET", "OPTIONS")
	
	// Chat routes
	r.HandleFunc("/api/chats", authMiddleware(getChatsHandler)).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/chats", authMiddleware(createChatHandler)).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/chats/group", authMiddleware(createGroupChatHandler)).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/chats/{id}", authMiddleware(getChatHandler)).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/chats/{id}/messages", authMiddleware(getChatMessagesHandler)).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/chats/{id}/messages", authMiddleware(sendMessageHandler)).Methods("POST", "OPTIONS")
	
	// WebSocket route
	r.HandleFunc("/ws", authMiddleware(wsHandler))
	
	// Use CORS middleware
	r.Use(corsMiddleware)
	
	// Create some test data
	createTestData()
	
	// Start server
	port := 8000
	fmt.Printf("Server running on http://localhost:%d\n", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), r))
}

// CORS middleware
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers for all responses
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		
		// Handle preflight OPTIONS requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		// Process the request
		next.ServeHTTP(w, r)
	})
}

// Auth middleware
func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Always allow OPTIONS requests
		if r.Method == "OPTIONS" {
			next.ServeHTTP(w, r)
			return
		}
		
		// WebSocket handling
		if r.URL.Path == "/ws" {
			token := r.URL.Query().Get("token")
			if token == "" {
				http.Error(w, "Missing authentication token", http.StatusUnauthorized)
				return
			}
			
			user, found := usersByAuth[token]
			if !found {
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				return
			}
			
			// Add user to request context
			ctx := r.Context()
			ctx = context.WithValue(ctx, "user", user)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}
		
		// Regular API auth
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || len(authHeader) <= 7 || authHeader[:7] != "Bearer " {
			http.Error(w, "Missing or invalid authorization header", http.StatusUnauthorized)
			return
		}
		
		tokenString := authHeader[7:]
		
		// Parse token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return jwtSecret, nil
		})
		
		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}
		
		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			userID, ok := claims["user_id"].(string)
			if !ok {
				http.Error(w, "Invalid token claims", http.StatusUnauthorized)
				return
			}
			
			mutex.RLock()
			user, found := users[userID]
			mutex.RUnlock()
			
			if !found {
				http.Error(w, "User not found", http.StatusUnauthorized)
				return
			}
			
			// Add user to request context
			ctx := r.Context()
			ctx = context.WithValue(ctx, "user", user)
			next.ServeHTTP(w, r.WithContext(ctx))
		} else {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
		}
	}
}

// Get current user from context
func getUserFromContext(r *http.Request) (User, bool) {
	user, ok := r.Context().Value("user").(User)
	return user, ok
}

// Register handler
func registerHandler(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	
	// Validate request
	if req.Username == "" || req.Email == "" || req.Password == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}
	
	// Check if email is already taken
	mutex.RLock()
	for _, u := range users {
		if u.Email == req.Email {
			mutex.RUnlock()
			http.Error(w, "Email is already in use", http.StatusBadRequest)
			return
		}
	}
	mutex.RUnlock()
	
	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Error hashing password", http.StatusInternalServerError)
		return
	}
	
	// Create new user
	newUser := User{
		ID:        uuid.New().String(),
		Username:  req.Username,
		Email:     req.Email,
		Password:  string(hashedPassword),
		IsOnline:  true,
		LastSeen:  time.Now(),
		CreatedAt: time.Now(),
	}
	
	// Generate token
	token := generateToken(newUser.ID)
	
	// Save user
	mutex.Lock()
	users[newUser.ID] = newUser
	usersByAuth[token] = newUser
	mutex.Unlock()
	
	// Return user with token
	response := LoginResponse{
		User:  newUser,
		Token: token,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Login handler
func loginHandler(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	
	// Find user by email
	var foundUser User
	var found bool
	
	mutex.RLock()
	for _, u := range users {
		if u.Email == req.Email {
			foundUser = u
			found = true
			break
		}
	}
	mutex.RUnlock()
	
	if !found {
		http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}
	
	// Check password
	if err := bcrypt.CompareHashAndPassword([]byte(foundUser.Password), []byte(req.Password)); err != nil {
		http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}
	
	// Update user status
	mutex.Lock()
	foundUser.IsOnline = true
	foundUser.LastSeen = time.Now()
	users[foundUser.ID] = foundUser
	mutex.Unlock()
	
	// Generate token
	token := generateToken(foundUser.ID)
	
	// Save user auth
	mutex.Lock()
	usersByAuth[token] = foundUser
	mutex.Unlock()
	
	// Broadcast status change
	broadcastUserStatus(foundUser.ID, true)
	
	// Return user with token
	response := LoginResponse{
		User:  foundUser,
		Token: token,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Get current user handler
func getCurrentUserHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := getUserFromContext(r)
	if !ok {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// Get all users handler
func getUsersHandler(w http.ResponseWriter, r *http.Request) {
	currentUser, _ := getUserFromContext(r)
	
	userList := []User{}
	
	mutex.RLock()
	for _, u := range users {
		if u.ID != currentUser.ID {
			userList = append(userList, u)
		}
	}
	mutex.RUnlock()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(userList)
}

// Get user by ID handler
func getUserHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	userID := params["id"]
	
	mutex.RLock()
	user, found := users[userID]
	mutex.RUnlock()
	
	if !found {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// Get all chats for current user
func getChatsHandler(w http.ResponseWriter, r *http.Request) {
	currentUser, _ := getUserFromContext(r)
	
	userChats := []map[string]interface{}{}
	
	mutex.RLock()
	for _, c := range chats {
		// Check if user is a participant
		isParticipant := false
		for _, pid := range c.ParticipantIDs {
			if pid == currentUser.ID {
				isParticipant = true
				break
			}
		}
		
		if !isParticipant {
			continue
		}
		
		// Get participants
		participants := []User{}
		for _, pid := range c.ParticipantIDs {
			if user, ok := users[pid]; ok {
				participants = append(participants, user)
			}
		}
		
		// Get last message
		var lastMessage *Message
		var unreadCount int
		
		chatMessages := messages[c.ID]
		if len(chatMessages) > 0 {
			lastMessage = &chatMessages[len(chatMessages)-1]
			
			// Count unread messages
			for _, m := range chatMessages {
				if !m.IsRead && m.SenderID != currentUser.ID {
					unreadCount++
				}
			}
		}
		
		chat := map[string]interface{}{
			"id":           c.ID,
			"name":         c.Name,
			"isGroup":      c.IsGroup,
			"participants": participants,
			"unreadCount":  unreadCount,
		}
		
		if lastMessage != nil {
			chat["lastMessage"] = lastMessage
		}
		
		userChats = append(userChats, chat)
	}
	mutex.RUnlock()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(userChats)
}

// Get chat by ID
func getChatHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	chatID := params["id"]
	currentUser, _ := getUserFromContext(r)
	
	mutex.RLock()
	chat, found := chats[chatID]
	mutex.RUnlock()
	
	if !found {
		http.Error(w, "Chat not found", http.StatusNotFound)
		return
	}
	
	// Check if user is a participant
	isParticipant := false
	for _, pid := range chat.ParticipantIDs {
		if pid == currentUser.ID {
			isParticipant = true
			break
		}
	}
	
	if !isParticipant {
		http.Error(w, "Unauthorized access to chat", http.StatusForbidden)
		return
	}
	
	// Get participants
	participants := []User{}
	mutex.RLock()
	for _, pid := range chat.ParticipantIDs {
		if user, ok := users[pid]; ok {
			participants = append(participants, user)
		}
	}
	mutex.RUnlock()
	
	response := map[string]interface{}{
		"id":           chat.ID,
		"name":         chat.Name,
		"isGroup":      chat.IsGroup,
		"participants": participants,
		"createdAt":    chat.CreatedAt,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Create new chat
func createChatHandler(w http.ResponseWriter, r *http.Request) {
	var req CreateChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	
	currentUser, _ := getUserFromContext(r)
	
	// Validate request
	if len(req.ParticipantIDs) == 0 {
		http.Error(w, "No participants specified", http.StatusBadRequest)
		return
	}
	
	// Check if a chat already exists between these participants
	chatExists := false
	var existingChat Chat
	
	mutex.RLock()
	for _, c := range chats {
		if !c.IsGroup && len(c.ParticipantIDs) == 2 {
			hasCurrentUser := false
			hasOtherUser := false
			
			for _, pid := range c.ParticipantIDs {
				if pid == currentUser.ID {
					hasCurrentUser = true
				}
				if pid == req.ParticipantIDs[0] {
					hasOtherUser = true
				}
			}
			
			if hasCurrentUser && hasOtherUser {
				chatExists = true
				existingChat = c
				break
			}
		}
	}
	mutex.RUnlock()
	
	if chatExists {
		// Return existing chat
		participants := []User{}
		mutex.RLock()
		for _, pid := range existingChat.ParticipantIDs {
			if user, ok := users[pid]; ok {
				participants = append(participants, user)
			}
		}
		mutex.RUnlock()
		
		response := map[string]interface{}{
			"id":           existingChat.ID,
			"name":         existingChat.Name,
			"isGroup":      existingChat.IsGroup,
			"participants": participants,
			"createdAt":    existingChat.CreatedAt,
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}
	
	// Create new chat
	newChat := Chat{
		ID:           uuid.New().String(),
		IsGroup:      false,
		ParticipantIDs: append(req.ParticipantIDs, currentUser.ID),
		CreatedAt:    time.Now(),
	}
	
	mutex.Lock()
	chats[newChat.ID] = newChat
	messages[newChat.ID] = []Message{}
	mutex.Unlock()
	
	// Get participants
	participants := []User{}
	mutex.RLock()
	for _, pid := range newChat.ParticipantIDs {
		if user, ok := users[pid]; ok {
			participants = append(participants, user)
		}
	}
	mutex.RUnlock()
	
	response := map[string]interface{}{
		"id":           newChat.ID,
		"name":         newChat.Name,
		"isGroup":      newChat.IsGroup,
		"participants": participants,
		"createdAt":    newChat.CreatedAt,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Create group chat
func createGroupChatHandler(w http.ResponseWriter, r *http.Request) {
	var req CreateGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	
	currentUser, _ := getUserFromContext(r)
	
	// Validate request
	if req.Name == "" {
		http.Error(w, "Group name is required", http.StatusBadRequest)
		return
	}
	
	if len(req.ParticipantIDs) == 0 {
		http.Error(w, "No participants specified", http.StatusBadRequest)
		return
	}
	
	// Create new group chat
	newChat := Chat{
		ID:           uuid.New().String(),
		Name:         req.Name,
		IsGroup:      true,
		ParticipantIDs: append(req.ParticipantIDs, currentUser.ID),
		CreatedAt:    time.Now(),
	}
	
	mutex.Lock()
	chats[newChat.ID] = newChat
	messages[newChat.ID] = []Message{}
	mutex.Unlock()
	
	// Get participants
	participants := []User{}
	mutex.RLock()
	for _, pid := range newChat.ParticipantIDs {
		if user, ok := users[pid]; ok {
			participants = append(participants, user)
		}
	}
	mutex.RUnlock()
	
	response := map[string]interface{}{
		"id":           newChat.ID,
		"name":         newChat.Name,
		"isGroup":      newChat.IsGroup,
		"participants": participants,
		"createdAt":    newChat.CreatedAt,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Get messages for a chat
func getChatMessagesHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	chatID := params["id"]
	currentUser, _ := getUserFromContext(r)
	
	mutex.RLock()
	chat, found := chats[chatID]
	mutex.RUnlock()
	
	if !found {
		http.Error(w, "Chat not found", http.StatusNotFound)
		return
	}
	
	// Check if user is a participant
	isParticipant := false
	for _, pid := range chat.ParticipantIDs {
		if pid == currentUser.ID {
			isParticipant = true
			break
		}
	}
	
	if !isParticipant {
		http.Error(w, "Unauthorized access to chat", http.StatusForbidden)
		return
	}
	
	// Get messages
	mutex.RLock()
	chatMessages := messages[chatID]
	mutex.RUnlock()
	
	// Mark messages as read
	mutex.Lock()
	for i := range chatMessages {
		if chatMessages[i].SenderID != currentUser.ID {
			chatMessages[i].IsRead = true
		}
	}
	messages[chatID] = chatMessages
	mutex.Unlock()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chatMessages)
}

// Send message
func sendMessageHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	chatID := params["id"]
	currentUser, _ := getUserFromContext(r)
	
	var req SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	
	// Validate request
	if req.Content == "" {
		http.Error(w, "Message content is required", http.StatusBadRequest)
		return
	}
	
	mutex.RLock()
	chat, found := chats[chatID]
	mutex.RUnlock()
	
	if !found {
		http.Error(w, "Chat not found", http.StatusNotFound)
		return
	}
	
	// Check if user is a participant
	isParticipant := false
	for _, pid := range chat.ParticipantIDs {
		if pid == currentUser.ID {
			isParticipant = true
			break
		}
	}
	
	if !isParticipant {
		http.Error(w, "Unauthorized access to chat", http.StatusForbidden)
		return
	}
	
	// Create new message
	newMessage := Message{
		ID:        uuid.New().String(),
		ChatID:    chatID,
		SenderID:  currentUser.ID,
		Content:   req.Content,
		Timestamp: time.Now(),
		IsRead:    false,
	}
	
	// Save message
	mutex.Lock()
	messages[chatID] = append(messages[chatID], newMessage)
	mutex.Unlock()
	
	// Broadcast message to all participants
	broadcastMessage(newMessage)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(newMessage)
}

// WebSocket handler
func wsHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := getUserFromContext(r)
	if !ok {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}
	
	// Upgrade to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Error upgrading to WebSocket: %v", err)
		return
	}
	
	// Store connection
	mutex.Lock()
	connections[user.ID] = conn
	mutex.Unlock()
	
	// Update user status
	mutex.Lock()
	user.IsOnline = true
	user.LastSeen = time.Now()
	users[user.ID] = user
	mutex.Unlock()
	
	// Broadcast user status
	broadcastUserStatus(user.ID, true)
	
	// Handle WebSocket messages
	go handleWebSocketMessages(user.ID, conn)
}

// Handle WebSocket messages
func handleWebSocketMessages(userID string, conn *websocket.Conn) {
	defer func() {
		conn.Close()
		
		mutex.Lock()
		delete(connections, userID)
		
		if user, ok := users[userID]; ok {
			user.IsOnline = false
			user.LastSeen = time.Now()
			users[userID] = user
		}
		mutex.Unlock()
		
		broadcastUserStatus(userID, false)
	}()
	
	for {
		var wsMsg WSMessage
		err := conn.ReadJSON(&wsMsg)
		if err != nil {
			break
		}
		
		switch wsMsg.Type {
		case "typing":
			if data, ok := wsMsg.Payload.(map[string]interface{}); ok {
				chatID, _ := data["chatId"].(string)
				isTyping, _ := data["isTyping"].(bool)
				
				mutex.RLock()
				chat, found := chats[chatID]
				mutex.RUnlock()
				
				if found {
					// Broadcast typing status to other participants
					for _, pid := range chat.ParticipantIDs {
						if pid != userID {
							sendWebSocketMessage(pid, WSMessage{
								Type: "typing",
								Payload: map[string]interface{}{
									"chatId":   chatID,
									"userId":   userID,
									"isTyping": isTyping,
								},
							})
						}
					}
				}
			}
		}
	}
}

// Broadcast message to all participants
func broadcastMessage(msg Message) {
	mutex.RLock()
	chat, found := chats[msg.ChatID]
	mutex.RUnlock()
	
	if !found {
		return
	}
	
	wsMsg := WSMessage{
		Type:    "message",
		Payload: msg,
	}
	
	for _, pid := range chat.ParticipantIDs {
		if pid != msg.SenderID {
			sendWebSocketMessage(pid, wsMsg)
		}
	}
}

// Broadcast user status change to all relevant users
func broadcastUserStatus(userID string, isOnline bool) {
	wsMsg := WSMessage{
		Type: "status",
		Payload: map[string]interface{}{
			"userId":   userID,
			"isOnline": isOnline,
		},
	}
	
	// Find all chats this user participates in
	userChatIDs := []string{}
	mutex.RLock()
	for id, chat := range chats {
		for _, pid := range chat.ParticipantIDs {
			if pid == userID {
				userChatIDs = append(userChatIDs, id)
				break
			}
		}
	}
	mutex.RUnlock()
	
	// Find all users that share a chat with this user
	notifiedUsers := make(map[string]bool)
	mutex.RLock()
	for _, chatID := range userChatIDs {
		if chat, ok := chats[chatID]; ok {
			for _, pid := range chat.ParticipantIDs {
				if pid != userID {
					notifiedUsers[pid] = true
				}
			}
		}
	}
	mutex.RUnlock()
	
	// Send WebSocket message to all relevant users
	for pid := range notifiedUsers {
		sendWebSocketMessage(pid, wsMsg)
	}
}

// Send WebSocket message to a user
func sendWebSocketMessage(userID string, msg WSMessage) {
	mutex.RLock()
	conn, ok := connections[userID]
	mutex.RUnlock()
	
	if ok {
		if err := conn.WriteJSON(msg); err != nil {
			log.Printf("Error sending WebSocket message to user %s: %v", userID, err)
		}
	}
}

// Generate JWT token
func generateToken(userID string) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(time.Hour * 24 * 7).Unix(), // 1 week
	})
	
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		log.Printf("Error generating token: %v", err)
		return ""
	}
	
	return tokenString
}

// Create test data for development
func createTestData() {
	// Create test users
	testUsers := []struct {
		username string
		email    string
		password string
	}{
		{"Alice", "alice@example.com", "password123"},
		{"Bob", "bob@example.com", "password123"},
		{"Charlie", "charlie@example.com", "password123"},
		{"Diana", "diana@example.com", "password123"},
	}
	
	for _, u := range testUsers {
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(u.password), bcrypt.DefaultCost)
		
		user := User{
			ID:        uuid.New().String(),
			Username:  u.username,
			Email:     u.email,
			Password:  string(hashedPassword),
			IsOnline:  false,
			LastSeen:  time.Now(),
			CreatedAt: time.Now(),
		}
		
		users[user.ID] = user
	}
	
	// Get user IDs
	var userIDs []string
	for id := range users {
		userIDs = append(userIDs, id)
	}
	
	// Create test chats
	if len(userIDs) >= 2 {
		// Create private chat between first two users
		privateChat := Chat{
			ID:           uuid.New().String(),
			IsGroup:      false,
			ParticipantIDs: []string{userIDs[0], userIDs[1]},
			CreatedAt:    time.Now(),
		}
		chats[privateChat.ID] = privateChat
		
		// Add some messages
		messages[privateChat.ID] = []Message{
			{
				ID:        uuid.New().String(),
				ChatID:    privateChat.ID,
				SenderID:  userIDs[0],
				Content:   "Hi there!",
				Timestamp: time.Now().Add(-time.Hour * 2),
				IsRead:    true,
			},
			{
				ID:        uuid.New().String(),
				ChatID:    privateChat.ID,
				SenderID:  userIDs[1],
				Content:   "Hello! How are you?",
				Timestamp: time.Now().Add(-time.Hour),
				IsRead:    true,
			},
		}
	}
	
	if len(userIDs) >= 3 {
		// Create group chat with first three users
		groupChat := Chat{
			ID:           uuid.New().String(),
			Name:         "Test Group",
			IsGroup:      true,
			ParticipantIDs: []string{userIDs[0], userIDs[1], userIDs[2]},
			CreatedAt:    time.Now(),
		}
		chats[groupChat.ID] = groupChat
		
		// Add some messages
		messages[groupChat.ID] = []Message{
			{
				ID:        uuid.New().String(),
				ChatID:    groupChat.ID,
				SenderID:  userIDs[0],
				Content:   "Welcome to the group!",
				Timestamp: time.Now().Add(-time.Hour * 3),
				IsRead:    true,
			},
			{
				ID:        uuid.New().String(),
				ChatID:    groupChat.ID,
				SenderID:  userIDs[1],
				Content:   "Thanks for adding me",
				Timestamp: time.Now().Add(-time.Hour * 2),
				IsRead:    true,
			},
			{
				ID:        uuid.New().String(),
				ChatID:    groupChat.ID,
				SenderID:  userIDs[2],
				Content:   "Great to be here",
				Timestamp: time.Now().Add(-time.Hour),
				IsRead:    true,
			},
		}
	}
}
