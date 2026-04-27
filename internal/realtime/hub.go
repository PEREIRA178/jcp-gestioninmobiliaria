package realtime

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/gofiber/websocket/v2"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// ══════════════════════════════════════════════════════
//  MESSAGE TYPES
// ══════════════════════════════════════════════════════

type MessageType string

const (
	MsgPropiedadesUpdate MessageType = "propiedades_update"
	MsgNoticiasUpdate    MessageType = "noticias_update"
	MsgRefreshWeb        MessageType = "refresh_web"
	MsgRefreshAll        MessageType = "refresh_all"
)

type Message struct {
	Type    MessageType     `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// ══════════════════════════════════════════════════════
//  CLIENT
// ══════════════════════════════════════════════════════

type Client struct {
	Conn *websocket.Conn
	Send chan []byte
}

// ══════════════════════════════════════════════════════
//  HUB — Central WebSocket broadcaster
// ══════════════════════════════════════════════════════

type Hub struct {
	clients    map[*Client]bool
	mu         sync.RWMutex
	broadcast  chan Message
	register   chan *Client
	unregister chan *Client
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan Message, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("🔌 WS client connected (total: %d)", len(h.clients))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)
			}
			h.mu.Unlock()
			log.Printf("🔌 WS client disconnected")

		case msg := <-h.broadcast:
			data, err := json.Marshal(msg)
			if err != nil {
				log.Printf("❌ WS marshal error: %v", err)
				continue
			}
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.Send <- data:
				default:
					go func(c *Client) { h.unregister <- c }(client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *Hub) Register(c *Client)    { h.register <- c }
func (h *Hub) Unregister(c *Client)  { h.unregister <- c }
func (h *Hub) Broadcast(msg Message) { h.broadcast <- msg }

// BroadcastWeb envía un mensaje de actualización a todos los clientes conectados.
func (h *Hub) BroadcastWeb(msgType MessageType) {
	h.Broadcast(Message{Type: msgType})
}

// ══════════════════════════════════════════════════════
//  POCKETBASE HOOKS — propiedades y noticias únicamente
// ══════════════════════════════════════════════════════

var hubInstance *Hub

func RegisterPBHooks(pb *pocketbase.PocketBase) {
	pb.OnServe().BindFunc(func(se *core.ServeEvent) error {
		// Propiedades → actualiza listados públicos en tiempo real
		se.App.OnRecordAfterCreateSuccess("propiedades").BindFunc(func(e *core.RecordEvent) error {
			if hubInstance != nil {
				hubInstance.BroadcastWeb(MsgPropiedadesUpdate)
			}
			return e.Next()
		})
		se.App.OnRecordAfterUpdateSuccess("propiedades").BindFunc(func(e *core.RecordEvent) error {
			if hubInstance != nil {
				hubInstance.BroadcastWeb(MsgPropiedadesUpdate)
			}
			return e.Next()
		})
		se.App.OnRecordAfterDeleteSuccess("propiedades").BindFunc(func(e *core.RecordEvent) error {
			if hubInstance != nil {
				hubInstance.BroadcastWeb(MsgPropiedadesUpdate)
			}
			return e.Next()
		})

		// Content_blocks (noticias) → actualiza listados públicos en tiempo real
		se.App.OnRecordAfterCreateSuccess("content_blocks").BindFunc(func(e *core.RecordEvent) error {
			if hubInstance != nil {
				hubInstance.BroadcastWeb(MsgNoticiasUpdate)
			}
			return e.Next()
		})
		se.App.OnRecordAfterUpdateSuccess("content_blocks").BindFunc(func(e *core.RecordEvent) error {
			if hubInstance != nil {
				hubInstance.BroadcastWeb(MsgNoticiasUpdate)
			}
			return e.Next()
		})
		se.App.OnRecordAfterDeleteSuccess("content_blocks").BindFunc(func(e *core.RecordEvent) error {
			if hubInstance != nil {
				hubInstance.BroadcastWeb(MsgNoticiasUpdate)
			}
			return e.Next()
		})

		return se.Next()
	})
}

// SetHubInstance almacena la referencia del hub para los PB hooks.
func SetHubInstance(h *Hub) {
	hubInstance = h
}
