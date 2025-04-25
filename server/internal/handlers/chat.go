
package handlers

import (
	"chat-app/internal/models"
	"chat-app/internal/store"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func GetChats(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(models.User)
	chats := store.GetChats()
	
	userChats := []map[string]interface{}{}
	users := store.GetUsers()

	for _, chat := range chats {
		isParticipant := false
		for _, pid := range chat.ParticipantIDs {
			if pid == user.ID {
				isParticipant = true
				break
			}
		}

		if !isParticipant {
			continue
		}

		participants := []models.User{}
		for _, pid := range chat.ParticipantIDs {
			if participant, ok := users[pid]; ok {
				participants = append(participants, participant)
			}
		}

		chatMessages := store.GetMessages(chat.ID)
		var lastMessage *models.Message
		var unreadCount int

		if len(chatMessages) > 0 {
			lastMsg := chatMessages[len(chatMessages)-1]
			lastMessage = &lastMsg

			for _, m := range chatMessages {
				if !m.IsRead && m.SenderID != user.ID {
					unreadCount++
				}
			}
		}

		chatResponse := map[string]interface{}{
			"id":           chat.ID,
			"name":         chat.Name,
			"isGroup":      chat.IsGroup,
			"participants": participants,
			"unreadCount":  unreadCount,
		}

		if lastMessage != nil {
			chatResponse["lastMessage"] = lastMessage
		}

		userChats = append(userChats, chatResponse)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(userChats)
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

	newChat := models.Chat{
		ID:             uuid.New().String(),
		IsGroup:        false,
		ParticipantIDs: append(req.ParticipantIDs, user.ID),
		CreatedAt:      time.Now(),
	}

	store.SetChat(newChat.ID, newChat)

	users := store.GetUsers()
	participants := []models.User{}
	for _, pid := range newChat.ParticipantIDs {
		if participant, ok := users[pid]; ok {
			participants = append(participants, participant)
		}
	}

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

	chat, exists := store.GetChats()[chatID]
	if !exists {
		http.Error(w, "Chat not found", http.StatusNotFound)
		return
	}

	newMessage := models.Message{
		ID:        uuid.New().String(),
		ChatID:    chatID,
		SenderID:  user.ID,
		Content:   req.Content,
		Timestamp: time.Now(),
		IsRead:    false,
	}

	store.AddMessage(chatID, newMessage)

	// Broadcast message to all participants
	for _, pid := range chat.ParticipantIDs {
		if pid != user.ID {
			if conn, exists := store.GetConnection(pid); exists {
				msgJSON, _ := json.Marshal(map[string]interface{}{
					"type":    "message",
					"payload": newMessage,
				})
				conn.Send <- msgJSON
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(newMessage)
}
