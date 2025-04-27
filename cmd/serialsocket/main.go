package main

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/anikinmd/serialsocket/internal/config"
	"github.com/anikinmd/serialsocket/internal/serial"
	"github.com/anikinmd/serialsocket/internal/ws"
)

func main() {
	// Load configuration and setup logging
	cfg := config.New()
	cfg.SetupLogger()

	// Channels for serial ↔ websocket
	serialRx := make(chan []byte, 1024)
	serialTx := make(chan []byte, 1024)

	// Instantiate components
	serialMgr := serial.NewManager(cfg.Device, cfg.Baud)
	wsServer := ws.NewServer(cfg.WSPort, cfg.AllowOrigin)

	// Run
	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := serialMgr.Run(ctx, serialRx, serialTx); err != nil {
			log.Error().Err(err).Msg("Serial manager error")
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := wsServer.Run(ctx, serialRx, serialTx); err != nil {
			log.Error().Err(err).Msg("WebSocket server error")
		}
	}()

	log.Info().
		Str("device", cfg.Device).
		Int("baud", cfg.Baud).
		Int("ws-port", cfg.WSPort).
		Msg("Serial WebSocket Bridge started")

	// Graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	log.Info().Msg("Received shutdown signal")
	cancel()

	// Wait or timeout
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()

	select {
	case <-done:
		log.Info().Msg("Shutdown complete")
	case <-time.After(5 * time.Second):
		log.Warn().Msg("Forced shutdown after timeout")
	}
}
