package ws

import (
	"jcp-gestioninmobiliaria/internal/realtime"

	"github.com/gofiber/websocket/v2"
)

// WebSocket maneja conexiones WebSocket de clientes web para recibir
// actualizaciones en tiempo real de propiedades y noticias.
func WebSocket(hub *realtime.Hub) func(*websocket.Conn) {
	return func(c *websocket.Conn) {
		client := &realtime.Client{
			Conn: c,
			Send: make(chan []byte, 64),
		}

		hub.Register(client)
		defer hub.Unregister(client)

		// Writer goroutine: reenvía mensajes del hub al cliente
		go func() {
			for msg := range client.Send {
				if err := c.WriteMessage(websocket.TextMessage, msg); err != nil {
					return
				}
			}
		}()

		// Reader loop: mantiene la conexión viva
		for {
			if _, _, err := c.ReadMessage(); err != nil {
				break
			}
		}
	}
}
