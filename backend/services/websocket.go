package services

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// TODO: Configure properly in production with allowed origins
		return true
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// WebSocket connection manager
type ConnectionManager struct {
	connections map[string]*websocket.Conn // userID -> connection
	mutex       sync.RWMutex
	broadcast   chan Message
}

var manager = &ConnectionManager{
	connections: make(map[string]*websocket.Conn),
	broadcast:   make(chan Message, 100), // Buffered channel
}

// Message types
type Message struct {
	Type           string                 `json:"type"` // "file_share", "code_share", "ping", "notification", "collaboration_request"
	SenderID       string                 `json:"sender_id,omitempty"`
	SenderEmail    string                 `json:"sender_email,omitempty"`
	RecipientID    string                 `json:"recipient_id,omitempty"`
	RecipientEmail string                 `json:"recipient_email,omitempty"`
	ProjectID      string                 `json:"project_id,omitempty"`
	ProjectName    string                 `json:"project_name,omitempty"`
	FileName       string                 `json:"file_name,omitempty"`
	FileContent    string                 `json:"file_content,omitempty"`
	FileType       string                 `json:"file_type,omitempty"`
	Message        string                 `json:"message,omitempty"`
	Timestamp      string                 `json:"timestamp"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// Initialize broadcast handler
func init() {
	go manager.handleBroadcast()
}

// Handle broadcast messages
func (cm *ConnectionManager) handleBroadcast() {
	for msg := range cm.broadcast {
		cm.mutex.RLock()
		conn, exists := cm.connections[msg.RecipientID]
		cm.mutex.RUnlock()

		if exists {
			// Set write deadline to prevent hanging
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			err := conn.WriteJSON(msg)
			if err != nil {
				log.Printf("Error sending message to %s: %v", msg.RecipientID, err)
				cm.removeConnection(msg.RecipientID)
			}
		} else {
			log.Printf("Recipient %s not connected, message queued or dropped", msg.RecipientID)
			// TODO: Store message in database for offline delivery
		}
	}
}

// Add connection
func (cm *ConnectionManager) addConnection(userID string, conn *websocket.Conn) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// Close existing connection if any
	if existingConn, exists := cm.connections[userID]; exists {
		existingConn.Close()
		log.Printf("Closed existing connection for user %s", userID)
	}

	cm.connections[userID] = conn
	log.Printf("User %s connected. Total connections: %d", userID, len(cm.connections))
}

// Remove connection
func (cm *ConnectionManager) removeConnection(userID string) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	if conn, exists := cm.connections[userID]; exists {
		conn.Close()
		delete(cm.connections, userID)
		log.Printf("User %s disconnected. Total connections: %d", userID, len(cm.connections))
	}
}

// Check if user is online
func (cm *ConnectionManager) isUserOnline(userID string) bool {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	_, exists := cm.connections[userID]
	return exists
}

// Get all online user IDs
func (cm *ConnectionManager) getOnlineUserIDs() []string {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	userIDs := make([]string, 0, len(cm.connections))
	for userID := range cm.connections {
		userIDs = append(userIDs, userID)
	}
	return userIDs
}

// WebSocket handler - establishes connection
func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "user_id is required", http.StatusBadRequest)
		return
	}

	// Validate UUID
	if _, err := uuid.Parse(userID); err != nil {
		http.Error(w, "Invalid user_id format", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	manager.addConnection(userID, conn)

	// Send connection success message
	successMsg := Message{
		Type:      "connection_success",
		Message:   "Connected to real-time collaboration server",
		Timestamp: getCurrentTimestamp(),
	}
	conn.WriteJSON(successMsg)

	// Configure ping/pong for connection health
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// Start ping ticker
	go sendPing(userID, conn)

	// Listen for incoming messages
	handleClientMessages(userID, conn)
}

// Send periodic ping to keep connection alive
func sendPing(userID string, conn *websocket.Conn) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if !manager.isUserOnline(userID) {
			return
		}

		conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
			log.Printf("Ping error for user %s: %v", userID, err)
			manager.removeConnection(userID)
			return
		}
	}
}

// Handle incoming messages from client
func handleClientMessages(userID string, conn *websocket.Conn) {
	defer manager.removeConnection(userID)

	for {
		var msg Message
		err := conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error for user %s: %v", userID, err)
			}
			break
		}

		// Set sender ID and timestamp
		msg.SenderID = userID
		msg.Timestamp = getCurrentTimestamp()

		// Validate message
		if msg.Type == "" {
			log.Printf("Invalid message from user %s: missing type", userID)
			continue
		}

		// Process message based on type
		switch msg.Type {
		case "file_share", "code_share":
			if msg.RecipientID == "" {
				log.Printf("Invalid %s message: missing recipient_id", msg.Type)
				continue
			}
			// Broadcast to recipient
			manager.broadcast <- msg

		case "ping":
			// Respond with pong
			pong := Message{
				Type:      "pong",
				Message:   "Server alive",
				Timestamp: getCurrentTimestamp(),
			}
			conn.WriteJSON(pong)

		case "typing":
			// Forward typing indicator to recipient
			if msg.RecipientID != "" {
				manager.broadcast <- msg
			}

		default:
			log.Printf("Unknown message type from user %s: %s", userID, msg.Type)
		}
	}
}

// Send notification to user (can be called from other services)
func SendNotificationToUser(recipientID, notificationType, message string, metadata map[string]interface{}) {
	msg := Message{
		Type:        notificationType,
		RecipientID: recipientID,
		Message:     message,
		Timestamp:   getCurrentTimestamp(),
		Metadata:    metadata,
	}
	manager.broadcast <- msg
}

// Broadcast message to multiple users
func BroadcastToUsers(userIDs []string, msgType, message string, metadata map[string]interface{}) {
	for _, userID := range userIDs {
		msg := Message{
			Type:        msgType,
			RecipientID: userID,
			Message:     message,
			Timestamp:   getCurrentTimestamp(),
			Metadata:    metadata,
		}
		manager.broadcast <- msg
	}
}

// Get online users
func GetOnlineUsers(w http.ResponseWriter, r *http.Request) {
	userIDs := manager.getOnlineUserIDs()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":      true,
		"online_users": userIDs,
		"total_online": len(userIDs),
	})
}

// Check user online status
func CheckUserOnlineStatus(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "user_id is required",
		})
		return
	}

	isOnline := manager.isUserOnline(userID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"user_id":   userID,
		"is_online": isOnline,
	})
}

// Get current timestamp in RFC3339 format
func getCurrentTimestamp() string {
	return time.Now().Format(time.RFC3339)
}
