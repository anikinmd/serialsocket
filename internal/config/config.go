package config

import (
	"flag"
	"fmt"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Config holds CLI and env settings
type Config struct {
	Device           string
	Baud             int
	WSPort           int
	AllowOrigin      string
	LogLevel         string
	LogFormat        string
	SerialBufferSize int // New: Serial read buffer size
	ChannelSize      int // New: Size for RX/TX channels
}

// New parses flags and environment variables
func New() *Config {
	cfg := &Config{}
	flag.StringVar(&cfg.Device, "device", "/dev/ttyUSB0", "Serial device path")
	flag.IntVar(&cfg.Baud, "baud", 115200, "Baud rate")
	flag.IntVar(&cfg.WSPort, "ws-port", 8080, "WebSocket server port")
	flag.StringVar(&cfg.AllowOrigin, "allow-origin", "*", "Allowed origins for CORS")
	flag.StringVar(&cfg.LogLevel, "log-level", "info", "Log level (debug, info, warn, error)")
	flag.StringVar(&cfg.LogFormat, "log-format", "json", "Log format (json or console)")

	// New optimized parameters
	flag.IntVar(&cfg.SerialBufferSize, "serial-buffer", 4096, "Serial read buffer size")
	flag.IntVar(&cfg.ChannelSize, "channel-size", 4096, "Size for internal communication channels")

	flag.Parse()

	// Override with env vars if set
	if env := os.Getenv("SERIAL_WS_DEVICE"); env != "" {
		cfg.Device = env
	}
	if env := os.Getenv("SERIAL_WS_BAUD"); env != "" {
		fmt.Sscanf(env, "%d", &cfg.Baud)
	}
	if env := os.Getenv("SERIAL_WS_PORT"); env != "" {
		fmt.Sscanf(env, "%d", &cfg.WSPort)
	}
	if env := os.Getenv("SERIAL_WS_ALLOW_ORIGIN"); env != "" {
		cfg.AllowOrigin = env
	}
	if env := os.Getenv("SERIAL_WS_LOG_LEVEL"); env != "" {
		cfg.LogLevel = env
	}
	if env := os.Getenv("SERIAL_WS_LOG_FORMAT"); env != "" {
		cfg.LogFormat = env
	}
	if env := os.Getenv("SERIAL_WS_BUFFER_SIZE"); env != "" {
		fmt.Sscanf(env, "%d", &cfg.SerialBufferSize)
	}
	if env := os.Getenv("SERIAL_WS_CHANNEL_SIZE"); env != "" {
		fmt.Sscanf(env, "%d", &cfg.ChannelSize)
	}

	// Validate required parameters
	if cfg.Device == "" || cfg.Baud <= 0 || cfg.WSPort <= 0 || cfg.WSPort > 65535 {
		flag.Usage()
		log.Fatal().Msg("missing or invalid required parameters")
	}

	// Validate optimization parameters
	if cfg.SerialBufferSize <= 0 {
		cfg.SerialBufferSize = 4096 // Default to 4K if invalid
	}
	if cfg.ChannelSize <= 0 {
		cfg.ChannelSize = 4096 // Default to 4K if invalid
	}

	return cfg
}

// SetupLogger configures zerolog
func (cfg *Config) SetupLogger() {
	level := zerolog.InfoLevel
	switch cfg.LogLevel {
	case "debug":
		level = zerolog.DebugLevel
	case "warn":
		level = zerolog.WarnLevel
	case "error":
		level = zerolog.ErrorLevel
	}
	zerolog.SetGlobalLevel(level)
	if cfg.LogFormat == "console" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}
}
