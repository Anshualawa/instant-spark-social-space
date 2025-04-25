
package handlers

import (
	"chat-app/internal/models"
	"chat-app/internal/store"
	"encoding/json"
	"log"
	"net/http"
	"time"

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

	user.IsOnline = true
	user.LastSeen = time.Now()
	store.SetUser(user.ID, user)

	// Broadcast user status
	broadcastUserStatus(user.ID, true)

	go handleWebSocketMessages(user.ID, conn, wsConn)
	go writePump(user.ID, conn, wsConn)
}

func handleWebSocketMessages(userID string, conn *websocket.Conn, wsConn *store.WebSocketConnection) {
	defer func() {
		conn.Close()
		store.RemoveConnection(userID)

		if user, ok := store.GetUsers()[userID]; ok {
			user.IsOnline = false
			user.LastSeen = time.Now()
			store.SetUser(userID, user)
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

				if chat, ok := store.GetChats()[chatID]; ok {
					for _, pid := range chat.ParticipantIDs {
						if pid != userID {
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
	msgJSON, _ := json.Marshal(WSMessage{
		Type: "status",
		Payload: map[string]interface{}{
			"userId":   userID,
			"isOnline": isOnline,
		},
	})

	connections := store.GetAllConnections()
	for pid, conn := range connections {
		if pid != userID {
			conn.Send <- msgJSON
		}
	}
}
