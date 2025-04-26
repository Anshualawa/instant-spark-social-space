package handlers

import (
	"chat-app/internal/config"
	"chat-app/internal/models"
	"chat-app/internal/store"
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type WSMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(models.User)

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Error upgrading to WebSocket: %v", err)
		return
	}

	wsConn := &store.WebSocketConnection{
		Send: make(chan []byte, 256),
	}

	store.SetConnection(user.ID, wsConn)

	// Update user's online status in database
	_, err = config.DB.Exec(`
		UPDATE users 
		SET is_online = true, last_seen = NOW() 
		WHERE id = ?
	`, user.ID)
	if err != nil {
		log.Printf("Error updating online status: %v", err)
	}

	// Broadcast user's online status to others
	broadcastUserStatus(user.ID, true)

	go handleWebSocketMessages(user.ID, conn, wsConn)
	go writePump(user.ID, conn, wsConn)
}

func handleWebSocketMessages(userID string, conn *websocket.Conn, wsConn *store.WebSocketConnection) {
	defer func() {
		conn.Close()
		store.RemoveConnection(userID)

		// Update user's offline status in database
		_, err := config.DB.Exec(`
			UPDATE users 
			SET is_online = false, last_seen = NOW() 
			WHERE id = ?
		`, userID)
		if err != nil {
			log.Printf("Error updating offline status: %v", err)
		}

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

				// Get chat participants
				rows, err := config.DB.Query(`
					SELECT user_id 
					FROM chat_participants 
					WHERE chat_id = ? AND user_id != ?
				`, chatID, userID)
				if err != nil {
					continue
				}
				defer rows.Close()

				for rows.Next() {
					var pid string
					if err := rows.Scan(&pid); err != nil {
						continue
					}
					if conn, exists := store.GetConnection(pid); exists {
						msgJSON, _ := json.Marshal(WSMessage{
							Type: "typing",
							Payload: map[string]interface{}{
								"chatId":   chatID,
								"userId":   userID,
								"isTyping": isTyping,
							},
						})
						conn.Send <- msgJSON
					}
				}
			}
		}
	}
}

func writePump(userID string, conn *websocket.Conn, wsConn *store.WebSocketConnection) {
	for {
		message, ok := <-wsConn.Send
		if !ok {
			conn.WriteMessage(websocket.CloseMessage, []byte{})
			return
		}

		err := conn.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			return
		}
	}
}

func broadcastUserStatus(userID string, isOnline bool) {
	// Get all users who have chats with this user
	rows, err := config.DB.Query(`
		SELECT DISTINCT user_id 
		FROM chat_participants 
		WHERE chat_id IN (
			SELECT chat_id 
			FROM chat_participants 
			WHERE user_id = ?
		) AND user_id != ?
	`, userID, userID)
	if err != nil {
		log.Printf("Error getting chat participants: %v", err)
		return
	}
	defer rows.Close()

	msgJSON, _ := json.Marshal(WSMessage{
		Type: "status",
		Payload: map[string]interface{}{
			"userId":   userID,
			"isOnline": isOnline,
		},
	})

	for rows.Next() {
		var pid string
		if err := rows.Scan(&pid); err != nil {
			continue
		}
		if conn, exists := store.GetConnection(pid); exists {
			conn.Send <- msgJSON
		}
	}
}
