
package store

import (
	"chat-app/internal/models"
	"sync"
)

// In-memory data store (replace with MySQL in production)
var (
	users       = make(map[string]models.User)
	chats       = make(map[string]models.Chat)
	messages    = make(map[string][]models.Message)
	usersByAuth = make(map[string]models.User) // Maps tokens to users
	connections = make(map[string]*WebSocketConnection)
	mutex       = &sync.RWMutex{}
)

type WebSocketConnection struct {
	Send chan []byte
}

func GetUsers() map[string]models.User {
	mutex.RLock()
	defer mutex.RUnlock()
	return users
}

func SetUser(id string, user models.User) {
	mutex.Lock()
	defer mutex.Unlock()
	users[id] = user
}

func GetUserByEmail(email string) (models.User, bool) {
	mutex.RLock()
	defer mutex.RUnlock()
	for _, u := range users {
		if u.Email == email {
			return u, true
		}
	}
	return models.User{}, false
}

func GetChats() map[string]models.Chat {
	mutex.RLock()
	defer mutex.RUnlock()
	return chats
}

func SetChat(id string, chat models.Chat) {
	mutex.Lock()
	defer mutex.Unlock()
	chats[id] = chat
}

func GetMessages(chatID string) []models.Message {
	mutex.RLock()
	defer mutex.RUnlock()
	return messages[chatID]
}

func AddMessage(chatID string, msg models.Message) {
	mutex.Lock()
	defer mutex.Unlock()
	messages[chatID] = append(messages[chatID], msg)
}

func SetConnection(userID string, conn *WebSocketConnection) {
	mutex.Lock()
	defer mutex.Unlock()
	connections[userID] = conn
}

func RemoveConnection(userID string) {
	mutex.Lock()
	defer mutex.Unlock()
	delete(connections, userID)
}

func GetConnection(userID string) (*WebSocketConnection, bool) {
	mutex.RLock()
	defer mutex.RUnlock()
	conn, exists := connections[userID]
	return conn, exists
}

func GetAllConnections() map[string]*WebSocketConnection {
	mutex.RLock()
	defer mutex.RUnlock()
	return connections
}
