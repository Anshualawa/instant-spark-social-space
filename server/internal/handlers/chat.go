
package handlers

import (
	"chat-app/internal/config"
	"chat-app/internal/models"
	"chat-app/internal/store"
	
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func GetChats(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(models.User)
	
	// Get all chats for the user including participants
	rows, err := config.DB.Query(`
		SELECT DISTINCT 
			c.id, c.name, c.is_group, c.created_at,
			u.id, u.username, u.email, u.avatar, u.is_online, u.last_seen,
			(
				SELECT COUNT(*) 
				FROM messages m 
				WHERE m.chat_id = c.id 
				AND m.sender_id != ? 
				AND m.is_read = false
			) as unread_count,
			(
				SELECT JSON_OBJECT(
					'id', m.id,
					'chat_id', m.chat_id,
					'sender_id', m.sender_id,
					'content', m.content,
					'is_read', m.is_read,
					'created_at', m.created_at
				)
				FROM messages m
				WHERE m.chat_id = c.id
				ORDER BY m.created_at DESC
				LIMIT 1
			) as last_message
		FROM chats c
		JOIN chat_participants cp ON c.id = cp.chat_id
		JOIN users u ON u.id IN (
			SELECT user_id 
			FROM chat_participants 
			WHERE chat_id = c.id AND user_id != ?
		)
		WHERE c.id IN (
			SELECT chat_id 
			FROM chat_participants 
			WHERE user_id = ?
		)
	`, user.ID, user.ID, user.ID)
	
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type ChatResponse struct {
		ID           string          `json:"id"`
		Name         string          `json:"name"`
		IsGroup      bool            `json:"isGroup"`
		Participants []models.User   `json:"participants"`
		UnreadCount  int            `json:"unreadCount"`
		LastMessage  *models.Message `json:"lastMessage,omitempty"`
		CreatedAt    time.Time       `json:"createdAt"`
	}

	chatMap := make(map[string]*ChatResponse)
	
	for rows.Next() {
		var chat ChatResponse
		var participant models.User
		var unreadCount int
		var lastMessageJSON *string
		
		err := rows.Scan(
			&chat.ID, &chat.Name, &chat.IsGroup, &chat.CreatedAt,
			&participant.ID, &participant.Username, &participant.Email, 
			&participant.Avatar, &participant.IsOnline, &participant.LastSeen,
			&unreadCount, &lastMessageJSON,
		)
		if err != nil {
			http.Error(w, "Error scanning rows", http.StatusInternalServerError)
			return
		}

		if existingChat, ok := chatMap[chat.ID]; ok {
			existingChat.Participants = append(existingChat.Participants, participant)
		} else {
			chat.Participants = []models.User{participant}
			chat.UnreadCount = unreadCount
			if lastMessageJSON != nil {
				var lastMessage models.Message
				if err := json.Unmarshal([]byte(*lastMessageJSON), &lastMessage); err == nil {
					chat.LastMessage = &lastMessage
				}
			}
			chatMap[chat.ID] = &chat
		}
	}

	chats := make([]ChatResponse, 0, len(chatMap))
	for _, chat := range chatMap {
		chats = append(chats, *chat)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chats)
}

func CreateChat(w http.ResponseWriter, r *http.Request) {
	var req models.CreateChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	user := r.Context().Value("user").(models.User)

	if len(req.ParticipantIDs) == 0 {
		http.Error(w, "No participants specified", http.StatusBadRequest)
		return
	}

	// Validate participant IDs
	for _, participantID := range req.ParticipantIDs {
		var count int
		err := config.DB.QueryRow("SELECT COUNT(*) FROM users WHERE id = ?", participantID).Scan(&count)
		if err != nil || count == 0 {
			http.Error(w, fmt.Sprintf("Invalid participant ID: %s", participantID), http.StatusBadRequest)
			return
		}
	}

	chatID := uuid.New().String()

	// Start transaction
	tx, err := config.DB.Begin()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		log.Printf("Error starting transaction: %v", err)
		return
	}
	defer tx.Rollback()

	// Create chat
	_, err = tx.Exec(`
		INSERT INTO chats (id, is_group, created_at)
		VALUES (?, false, NOW())
	`, chatID)
	if err != nil {
		http.Error(w, "Error creating chat", http.StatusInternalServerError)
		log.Printf("Error creating chat: %v", err)
		return
	}

	// Add participants
	for _, participantID := range append(req.ParticipantIDs, user.ID) {
		_, err = tx.Exec(`
			INSERT INTO chat_participants (chat_id, user_id, joined_at)
			VALUES (?, ?, NOW())
		`, chatID, participantID)
		if err != nil {
			http.Error(w, "Error adding participants", http.StatusInternalServerError)
			log.Printf("Error adding participant %s: %v", participantID, err)
			return
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		http.Error(w, "Error committing transaction", http.StatusInternalServerError)
		log.Printf("Error committing transaction: %v", err)
		return
	}

	// Get chat with participants
	var chat ChatResponse
	err = config.DB.QueryRow(`
		SELECT id, COALESCE(name, ''), is_group, created_at
		FROM chats WHERE id = ?
	`, chatID).Scan(&chat.ID, &chat.Name, &chat.IsGroup, &chat.CreatedAt)
	if err != nil {
		http.Error(w, "Error retrieving chat", http.StatusInternalServerError)
		log.Printf("Error retrieving chat: %v", err)
		return
	}

	chat.Participants = []models.User{}

	// Get participants
	rows, err := config.DB.Query(`
		SELECT u.id, u.username, u.email, COALESCE(u.avatar, ''), u.is_online, u.last_seen
		FROM users u
		JOIN chat_participants cp ON u.id = cp.user_id
		WHERE cp.chat_id = ?
	`, chatID)
	if err != nil {
		http.Error(w, "Error retrieving participants", http.StatusInternalServerError)
		log.Printf("Error retrieving participants: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var participant models.User
		err := rows.Scan(
			&participant.ID, &participant.Username, &participant.Email,
			&participant.Avatar, &participant.IsOnline, &participant.LastSeen,
		)
		if err != nil {
			http.Error(w, "Error scanning participants", http.StatusInternalServerError)
			log.Printf("Error scanning participant: %v", err)
			return
		}
		chat.Participants = append(chat.Participants, participant)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chat)
}

func SendMessage(w http.ResponseWriter, r *http.Request) {
	chatID := chi.URLParam(r, "id")
	user := r.Context().Value("user").(models.User)

	var req models.SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Content == "" {
		http.Error(w, "Message content is required", http.StatusBadRequest)
		return
	}

	// Verify chat exists and user is participant
	var count int
	err := config.DB.QueryRow(`
		SELECT COUNT(*) FROM chat_participants 
		WHERE chat_id = ? AND user_id = ?
	`, chatID, user.ID).Scan(&count)
	if err != nil || count == 0 {
		http.Error(w, "Chat not found or user not participant", http.StatusNotFound)
		return
	}

	// Insert message
	messageID := uuid.New().String()
	_, err = config.DB.Exec(`
		INSERT INTO messages (id, chat_id, sender_id, content, is_read, created_at)
		VALUES (?, ?, ?, ?, false, NOW())
	`, messageID, chatID, user.ID, req.Content)
	if err != nil {
		http.Error(w, "Error sending message", http.StatusInternalServerError)
		return
	}

	// Get message with timestamp
	var newMessage models.Message
	err = config.DB.QueryRow(`
		SELECT id, chat_id, sender_id, content, is_read, created_at
		FROM messages WHERE id = ?
	`, messageID).Scan(
		&newMessage.ID, &newMessage.ChatID, &newMessage.SenderID,
		&newMessage.Content, &newMessage.IsRead, &newMessage.Timestamp,
	)
	if err != nil {
		http.Error(w, "Error retrieving message", http.StatusInternalServerError)
		return
	}

	// Get chat participants for broadcasting
	rows, err := config.DB.Query(`
		SELECT user_id FROM chat_participants WHERE chat_id = ? AND user_id != ?
	`, chatID, user.ID)
	if err != nil {
		http.Error(w, "Error getting participants", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var participantIDs []string
	for rows.Next() {
		var participantID string
		if err := rows.Scan(&participantID); err != nil {
			continue
		}
		participantIDs = append(participantIDs, participantID)
	}

	// Broadcast to all participants via WebSocket
	for _, pid := range participantIDs {
		if conn, exists := store.GetConnection(pid); exists {
			msgJSON, _ := json.Marshal(map[string]interface{}{
				"type":    "message",
				"payload": newMessage,
			})
			conn.Send <- msgJSON
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(newMessage)
}

func GetMessages(w http.ResponseWriter, r *http.Request) {
	chatID := chi.URLParam(r, "id")
	user := r.Context().Value("user").(models.User)

	// Verify chat exists and user is participant
	var count int
	err := config.DB.QueryRow(`
		SELECT COUNT(*) FROM chat_participants 
		WHERE chat_id = ? AND user_id = ?
	`, chatID, user.ID).Scan(&count)
	if err != nil || count == 0 {
		http.Error(w, "Chat not found or user not participant", http.StatusNotFound)
		return
	}

	// Get messages for the chat
	rows, err := config.DB.Query(`
		SELECT id, chat_id, sender_id, content, is_read, created_at
		FROM messages 
		WHERE chat_id = ?
		ORDER BY created_at ASC
	`, chatID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error retrieving messages: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var messages []models.Message
	for rows.Next() {
		var msg models.Message
		err := rows.Scan(
			&msg.ID, &msg.ChatID, &msg.SenderID,
			&msg.Content, &msg.IsRead, &msg.Timestamp,
		)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error scanning message: %v", err), http.StatusInternalServerError)
			return
		}
		messages = append(messages, msg)
	}

	// Mark all unread messages from others as read
	_, err = config.DB.Exec(`
		UPDATE messages
		SET is_read = true
		WHERE chat_id = ? AND sender_id != ? AND is_read = false
	`, chatID, user.ID)
	if err != nil {
		log.Printf("Error marking messages as read: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}

type ChatResponse struct {
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	IsGroup      bool          `json:"isGroup"`
	Participants []models.User `json:"participants"`
	UnreadCount  int          `json:"unreadCount"`
	LastMessage  *models.Message `json:"lastMessage,omitempty"`
	CreatedAt    time.Time     `json:"createdAt"`
}
