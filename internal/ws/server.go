package ws

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

// Server broadcasts serial data and forwards client input
type Server interface {
	Run(ctx context.Context, serialRx <-chan []byte, serialTx chan<- []byte) error
}

// NewServer creates a WS server on wsPort with CORS allowOrigin
func NewServer(wsPort int, allowOrigin string) Server {
	return &server{wsPort: wsPort, allowOrigin: allowOrigin}
}

type server struct {
	wsPort      int
	allowOrigin string
}

func (s *server) Run(ctx context.Context, serialRx <-chan []byte, serialTx chan<- []byte) error {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			if s.allowOrigin == "*" {
				return true
			}
			return r.Header.Get("Origin") == s.allowOrigin
		},
	}

	clients := make(map[*websocket.Conn]struct{})
	var mu sync.RWMutex

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Error().Err(err).Msg("WebSocket upgrade failed")
			return
		}
		mu.Lock()
		clients[conn] = struct{}{}
		mu.Unlock()
		log.Info().Str("remote", conn.RemoteAddr().String()).Msg("Client connected")

		go func() {
			defer func() {
				mu.Lock()
				delete(clients, conn)
				mu.Unlock()
				conn.Close()
				log.Info().Str("remote", conn.RemoteAddr().String()).Msg("Client disconnected")
			}()
			for {
				_, msg, err := conn.ReadMessage()
				if err != nil {
					if websocket.IsUnexpectedCloseError(err,
						websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
						log.Error().Err(err).Msg("WebSocket read error")
					}
					return
				}
				select {
				case serialTx <- msg:
				default:
					log.Warn().Msg("Serial TX buffer full, dropping data")
				}
			}
		}()
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(terminalHTML))
	})

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	srv := &http.Server{Addr: fmt.Sprintf(":%d", s.wsPort), Handler: mux}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error().Err(err).Msg("WebSocket server error")
		}
	}()
	log.Info().Int("port", s.wsPort).Msg("Starting WebSocket server")

	// Broadcast loop
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case data := <-serialRx:
				mu.RLock()
				for c := range clients {
					go func(c *websocket.Conn, d []byte) {
						if err := c.WriteMessage(websocket.BinaryMessage, d); err != nil {
							log.Error().Err(err).Msg("WebSocket write error")
						}
					}(c, data)
				}
				mu.RUnlock()
			}
		}
	}()

	<-ctx.Done()
	log.Info().Msg("Shutting down WebSocket server")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}
