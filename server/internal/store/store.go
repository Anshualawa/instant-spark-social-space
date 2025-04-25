
package store

import (
	"sync"
)

// WebSocket connections store
type WebSocketConnection struct {
	Send chan []byte
}

var (
	connections = make(map[string]*WebSocketConnection)
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

