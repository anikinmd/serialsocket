package config

import (
	"flag"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNew_Config(t *testing.T) {
	tests := []struct {
		name            string
		args            []string
		env             map[string]string
		wantDevice      string
		wantBaud        int
		wantWSPort      int
		wantAllow       string
		wantBufferSize  int
		wantChannelSize int
	}{
		{
			name:            "defaults",
			args:            []string{"cmd"},
			env:             map[string]string{},
			wantDevice:      "/dev/ttyUSB0",
			wantBaud:        115200,
			wantWSPort:      8080,
			wantAllow:       "*",
			wantBufferSize:  4096,
			wantChannelSize: 4096,
		},
		{
			name: "flags override",
			args: []string{
				"cmd", "-device", "/dev/ttyS1",
				"-baud", "9600",
				"-ws-port", "9000",
				"-allow-origin", "http://example.com",
				"-serial-buffer", "8192",
				"-channel-size", "2048",
			},
			env:             map[string]string{},
			wantDevice:      "/dev/ttyS1",
			wantBaud:        9600,
			wantWSPort:      9000,
			wantAllow:       "http://example.com",
			wantBufferSize:  8192,
			wantChannelSize: 2048,
		},
		{
			name: "env override",
			args: []string{"cmd"},
			env: map[string]string{
				"SERIAL_WS_DEVICE":       "/dev/ttyS2",
				"SERIAL_WS_BAUD":         "4800",
				"SERIAL_WS_PORT":         "8000",
				"SERIAL_WS_ALLOW_ORIGIN": "http://test.com",
				"SERIAL_WS_BUFFER_SIZE":  "16384",
				"SERIAL_WS_CHANNEL_SIZE": "1024",
			},
			wantDevice:      "/dev/ttyS2",
			wantBaud:        4800,
			wantWSPort:      8000,
			wantAllow:       "http://test.com",
			wantBufferSize:  16384,
			wantChannelSize: 1024,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Reset flag.CommandLine
			flag.CommandLine = flag.NewFlagSet(tc.name, flag.ExitOnError)

			// Clear all relevant env vars
			for k := range tc.env {
				os.Unsetenv(k)
			}
			// Set test envs
			for k, v := range tc.env {
				os.Setenv(k, v)
			}

			// Set CLI args
			os.Args = tc.args

			cfg := New()
			require.Equal(t, tc.wantDevice, cfg.Device)
			require.Equal(t, tc.wantBaud, cfg.Baud)
			require.Equal(t, tc.wantWSPort, cfg.WSPort)
			require.Equal(t, tc.wantAllow, cfg.AllowOrigin)
			require.Equal(t, tc.wantBufferSize, cfg.SerialBufferSize)
			require.Equal(t, tc.wantChannelSize, cfg.ChannelSize)
		})
	}
}
