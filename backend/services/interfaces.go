package services

// WebSocketBroadcaster interface for WebSocket broadcasting
type WebSocketBroadcaster interface {
	BroadcastToAll(msgType string, data interface{})
}
