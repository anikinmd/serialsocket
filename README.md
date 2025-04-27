# Serial WebSocket Bridge

A lightweight, resilient bridge that forwards bytes bidirectionally between a local serial interface and WebSocket clients.

## Features

- **Transparent bridge** - Raw byte stream in both directions, no extra framing
- **Robust hot-plug** - Never exits when the serial device vanishes
- **Simple CLI** - Three required flags: `--device`, `--baud`, `--ws-port`


## Installation

### From Source

```bash
cd serialsocket
go mod tidy
go build -o serialsocket cmd/serialsocket/main.go
```

## Usage

Basic usage with required flags:

```bash
serialsocket --device /dev/ttyUSB0 --baud 115200 --ws-port 8080
```

Full options:

```bash
serialsocket \
  --device /dev/ttyUSB0 \
  --baud 115200 \
  --ws-port 8080 \
  --log-level info \
  --log-format console \
  --allow-origin "*"
```

### Environment Variables

All command-line flags can also be specified via environment variables:

```bash
SERIAL_WS_DEVICE=/dev/ttyUSB0 \
SERIAL_WS_BAUD=115200 \
SERIAL_WS_PORT=8080 \
serialsocket
```
## Using the Serial Terminal

The bridge includes a built-in xterm.js terminal emulation that you can access with any web browser:

1. Start the serial bridge:
   ```bash
   serialsocket --device /dev/ttyUSB0 --baud 115200 --ws-port 8080
   ```

2. Open your browser and navigate to:
   ```
   http://localhost:8080/
   ```

3. You'll see a full terminal emulator with features including:
   - ANSI color support and cursor positioning
   - View incoming serial data in real-time
   - Type commands and press Enter to send them to the serial device
   - Support for special keys (Ctrl+C to interrupt)
   - Toggle between text and hex mode for binary data
   - Toggle local echo on/off
   - Clear the terminal with one click
   - Automatic reconnection if the connection is lost

The terminal uses xterm.js for a complete terminal emulation experience.

## API Endpoints

- `/` - Web-based xterm.js terminal emulator
- `/ws` - WebSocket endpoint for raw data transfer
- `/healthz` - Health check endpoint (returns HTTP 200 OK)

## License

MIT