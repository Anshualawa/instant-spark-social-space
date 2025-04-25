
package store

import (
	"chat-app/internal/models"
	"database/sql"
	"sync"
)

// WebSocket connections store
type WebSocketConnection struct {
	Send chan []byte
}

var (
	connections = make(map[string]*WebSocketConnection)
	users       = make(map[string]models.User)
	mutex       = &sync.RWMutex{}
)

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

// User store functions
func AddUser(user models.User) {
	mutex.Lock()
	defer mutex.Unlock()
	users[user.ID] = user
}

func RemoveUser(userID string) {
	mutex.Lock()
	defer mutex.Unlock()
	delete(users, userID)
}

func GetUser(userID string) (models.User, bool) {
	mutex.RLock()
	defer mutex.RUnlock()
	user, exists := users[userID]
	return user, exists
}

func GetUsers() map[string]models.User {
	mutex.RLock()
	defer mutex.RUnlock()
	return users
}
