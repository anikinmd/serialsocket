package main

import (
	"context"
	"os"
	"os/signal"
	"runtime"
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

	// Set GOMAXPROCS to limit CPU usage if needed
	// Uncomment and adjust if you want to limit CPU usage
	// runtime.GOMAXPROCS(2)

	// Enable GC tuning for better performance under constant load
	// This helps reduce GC pauses for continuous data streams
	// - 10000 reduces GC frequency - better for high-throughput UART data
	// - 10 gives more time to GC, which helps under high load
	os.Setenv("GOGC", "10000")
	os.Setenv("GODEBUG", "gctrace=0")

	// Channels for serial ↔ websocket with configurable sizes
	serialRx := make(chan []byte, cfg.ChannelSize)
	serialTx := make(chan []byte, cfg.ChannelSize)

	// Instantiate optimized components
	serialMgr := serial.NewManager(cfg.Device, cfg.Baud, cfg.SerialBufferSize)
	wsServer := ws.NewServer(cfg.WSPort, cfg.AllowOrigin)

	// Run
	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Log system info
	log.Info().
		Int("go_routines", runtime.NumGoroutine()).
		Int("cpus", runtime.NumCPU()).
		Str("device", cfg.Device).
		Int("baud", cfg.Baud).
		Int("ws-port", cfg.WSPort).
		Int("buffer_size", cfg.SerialBufferSize).
		Int("channel_size", cfg.ChannelSize).
		Msg("Serial WebSocket Bridge starting")

	// Run serial manager
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := serialMgr.Run(ctx, serialRx, serialTx); err != nil {
			log.Error().Err(err).Msg("Serial manager error")
		}
	}()

	// Run WebSocket server
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := wsServer.Run(ctx, serialRx, serialTx); err != nil {
			log.Error().Err(err).Msg("WebSocket server error")
		}
	}()

	// Periodically log system metrics
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				log.Debug().
					Int("goroutines", runtime.NumGoroutine()).
					Int("rx_channel_len", len(serialRx)).
					Int("tx_channel_len", len(serialTx)).
					Msg("System metrics")
			}
		}
	}()

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
