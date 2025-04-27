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
func NewManager(device string, baud int) Manager {
	return &manager{device: device, baud: baud}
}

type manager struct {
	device string
	baud   int
}

func (m *manager) Run(ctx context.Context, serialRx chan<- []byte, serialTx <-chan []byte) error {
	log.Info().Str("device", m.device).Int("baud", m.baud).Msg("Starting serial manager")
	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("Serial manager shutting down")
			return nil
		default:
		}

		port, err := m.openPort()
		if err != nil {
			log.Warn().Err(err).Msg("Waiting for serial device...")
			time.Sleep(time.Second)
			continue
		}

		log.Info().Str("device", m.device).Msg("Connected to serial port")
		portCtx, cancel := context.WithCancel(ctx)
		var wg sync.WaitGroup
		wg.Add(2)

		// Read
		go func() {
			defer wg.Done()
			buf := make([]byte, 1024)
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
					data := make([]byte, n)
					copy(data, buf[:n])
					select {
					case serialRx <- data:
					default:
						log.Warn().Msg("Serial RX buffer full, dropping data")
					}
				}
			}
		}()

		// Write
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
			time.Sleep(time.Second)
		}
	}
}

func (m *manager) openPort() (serial.Port, error) {
	mode := &serial.Mode{BaudRate: m.baud, DataBits: 8, Parity: serial.NoParity, StopBits: serial.OneStopBit}
	return serial.Open(m.device, mode)
}
