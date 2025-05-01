package serial

import (
	"context"
	"io"
	"os"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"go.bug.st/serial"
)

// Manager controls serial I/O
type Manager interface {
	Run(ctx context.Context, serialRx chan<- []byte, serialTx <-chan []byte) error
}

// NewManager returns a Manager for device and baud
func NewManager(device string, baud int, bufferSize int) Manager {
	return &manager{
		device:     device,
		baud:       baud,
		bufferSize: bufferSize,
		bufferPool: sync.Pool{
			New: func() interface{} {
				// Pre-allocate buffers with the expected size for better performance
				return make([]byte, bufferSize)
			},
		},
	}
}

type manager struct {
	device     string
	baud       int
	bufferSize int
	bufferPool sync.Pool
}

func (m *manager) Run(ctx context.Context, serialRx chan<- []byte, serialTx <-chan []byte) error {
	log.Info().Str("device", m.device).Int("baud", m.baud).Msg("Starting serial manager")

	// Backoff strategy variables
	backoffTime := 100 * time.Millisecond
	maxBackoff := 2 * time.Second

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("Serial manager shutting down")
			return nil
		default:
		}

		port, err := m.openPort()
		if err != nil {
			log.Warn().Err(err).Dur("backoff", backoffTime).Msg("Waiting for serial device...")
			time.Sleep(backoffTime)

			// Increase backoff time for next attempt (simple exponential backoff)
			backoffTime *= 2
			if backoffTime > maxBackoff {
				backoffTime = maxBackoff
			}
			continue
		}

		// Reset backoff on successful connection
		backoffTime = 100 * time.Millisecond

		log.Info().Str("device", m.device).Msg("Connected to serial port")
		portCtx, cancel := context.WithCancel(ctx)
		var wg sync.WaitGroup
		wg.Add(2)

		// Read from serial port
		go func() {
			defer wg.Done()
			// Get a buffer from the pool
			buf := m.bufferPool.Get().([]byte)
			defer m.bufferPool.Put(buf)

			for {
				select {
				case <-portCtx.Done():
					return
				default:
				}

				n, err := port.Read(buf)
				if err != nil {
					if err == io.EOF || err == io.ErrClosedPipe || os.IsNotExist(err) {
						log.Warn().Err(err).Msg("Serial port disconnected")
					} else {
						log.Error().Err(err).Msg("Error reading from serial port")
					}
					cancel()
					return
				}

				if n > 0 {
					// Create a copy of just the data we need to send
					// This is necessary since the buffer will be reused
					data := make([]byte, n)
					copy(data, buf[:n])

					// Try to send data with non-blocking check to avoid goroutine blocking
					select {
					case serialRx <- data:
						// Data sent successfully
					case <-portCtx.Done():
						return
					default:
						// Channel full, log message and continue
						log.Warn().Msg("Serial RX buffer full, dropping data")
					}
				}
			}
		}()

		// Write to serial port
		go func() {
			defer wg.Done()
			for {
				select {
				case <-portCtx.Done():
					return
				case data := <-serialTx:
					if _, err := port.Write(data); err != nil {
						log.Error().Err(err).Msg("Error writing to serial port")
						cancel()
						return
					}
					log.Debug().Int("bytes", len(data)).Msg("Data sent to serial port")
				}
			}
		}()

		wg.Wait()
		port.Close()

		select {
		case <-ctx.Done():
			return nil
		default:
			log.Info().Msg("Port closed, reconnecting...")
			time.Sleep(200 * time.Millisecond) // Short delay before reconnection attempt
		}
	}
}

func (m *manager) openPort() (serial.Port, error) {
	mode := &serial.Mode{
		BaudRate: m.baud,
		DataBits: 8,
		Parity:   serial.NoParity,
		StopBits: serial.OneStopBit,
	}
	return serial.Open(m.device, mode)
}
