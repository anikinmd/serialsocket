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
	return &server{
		wsPort:      wsPort,
		allowOrigin: allowOrigin,
		clients:     make(map[*websocket.Conn]bool), // Track active status
	}
}

type server struct {
	wsPort      int
	allowOrigin string
	clients     map[*websocket.Conn]bool
	mu          sync.RWMutex
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

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Error().Err(err).Msg("WebSocket upgrade failed")
			return
		}

		// Configure WebSocket for performance
		conn.SetReadLimit(4096) // Limit max message size
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		conn.SetPongHandler(func(string) error {
			conn.SetReadDeadline(time.Now().Add(60 * time.Second))
			return nil
		})

		s.mu.Lock()
		s.clients[conn] = true
		s.mu.Unlock()

		log.Info().Str("remote", conn.RemoteAddr().String()).Msg("Client connected")

		go func() {
			defer func() {
				s.mu.Lock()
				delete(s.clients, conn)
				s.mu.Unlock()
				conn.Close()
				log.Info().Str("remote", conn.RemoteAddr().String()).Msg("Client disconnected")
			}()

			// Start a ping ticker
			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()

			go func() {
				for {
					select {
					case <-ctx.Done():
						return
					case <-ticker.C:
						conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
						if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
							return
						}
					}
				}
			}()

			for {
				conn.SetReadDeadline(time.Now().Add(60 * time.Second))
				_, msg, err := conn.ReadMessage()
				if err != nil {
					if websocket.IsUnexpectedCloseError(err,
						websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
						log.Error().Err(err).Msg("WebSocket read error")
					}
					return
				}

				// Only send message if serialTx has room
				select {
				case serialTx <- msg:
					// Message sent successfully
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

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", s.wsPort),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error().Err(err).Msg("WebSocket server error")
		}
	}()

	log.Info().Int("port", s.wsPort).Msg("Starting WebSocket server")

	// Broadcast loop - optimized for small number of clients
	// No need for per-client goroutines with only 2-3 clients
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case data := <-serialRx:
				if len(data) == 0 {
					continue
				}

				// Efficient broadcasting with minimal locking
				s.mu.RLock()
				clientCount := len(s.clients)
				if clientCount == 0 {
					s.mu.RUnlock()
					continue
				}

				// For small number of clients, we broadcast directly without spawning goroutines
				for c, active := range s.clients {
					if !active {
						continue
					}

					c.SetWriteDeadline(time.Now().Add(5 * time.Second))
					if err := c.WriteMessage(websocket.BinaryMessage, data); err != nil {
						log.Error().Err(err).Msg("WebSocket write error")
						// Mark client for cleanup
						s.mu.RUnlock()
						s.mu.Lock()
						s.clients[c] = false
						s.mu.Unlock()
						s.mu.RLock()
					}
				}
				s.mu.RUnlock()

				// Clean up inactive clients occasionally
				go s.cleanupInactiveClients()
			}
		}
	}()

	<-ctx.Done()
	log.Info().Msg("Shutting down WebSocket server")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}

// Cleanup routine for removing inactive clients
func (s *server) cleanupInactiveClients() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for c, active := range s.clients {
		if !active {
			delete(s.clients, c)
			c.Close()
		}
	}
}
