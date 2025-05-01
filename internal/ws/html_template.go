package ws

const terminalHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Serial Terminal</title>
    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/xterm@5.1.0/css/xterm.min.css" />
    <script src="https://cdn.jsdelivr.net/npm/xterm@5.1.0/lib/xterm.min.js"></script>
    <script src="https://cdn.jsdelivr.net/npm/xterm-addon-fit@0.7.0/lib/xterm-addon-fit.min.js"></script>
    <script src="https://cdn.jsdelivr.net/npm/xterm-addon-web-links@0.8.0/lib/xterm-addon-web-links.min.js"></script>
    <style>
        html, body {
            margin: 0;
            padding: 0;
            height: 100%;
            width: 100%;
            font-family: 'Menlo', 'DejaVu Sans Mono', 'Consolas', monospace;
            background-color: #1e1e1e;
            color: #f0f0f0;
            overflow: hidden;
        }
        
        body {
            display: flex;
            flex-direction: column;
        }
        
        .header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            background-color: #2d2d2d;
            padding: 4px 8px;
            border-bottom: 1px solid #3e3e3e;
            height: 32px;
        }
        
        .title {
            font-size: 14px;
            font-weight: bold;
            color: #f0f0f0;
        }
        
        .controls {
            display: flex;
            gap: 8px;
            align-items: center;
        }
        
        .status {
            padding: 4px 8px;
            border-radius: 4px;
            font-size: 12px;
            background-color: #5a1e1e;
        }
        
        .connected {
            background-color: #1e5a1e;
        }
        
        .connecting {
            background-color: #5a5a1e;
        }
        
        .button {
            padding: 4px 8px;
            border-radius: 4px;
            background-color: #3e3e3e;
            border: none;
            color: #f0f0f0;
            font-size: 12px;
            cursor: pointer;
        }
        
        .button:hover {
            background-color: #4e4e4e;
        }
        
        .terminal-container {
            flex: 1;
            padding: 4px;
            background-color: #1e1e1e;
            overflow: hidden;
        }
        
        #terminal {
            width: 100%;
            height: 100%;
        }
    </style>
</head>
<body>
    <div class="header">
        <div class="title">Serial Terminal</div>
        <div class="controls">
            <div id="status" class="status">Disconnected</div>
            <button id="clearBtn" class="button">Clear</button>
            <button id="hexModeBtn" class="button">Hex Mode: Off</button>
            <button id="localEchoBtn" class="button">Local Echo: On</button>
        </div>
    </div>
    
    <div class="terminal-container">
        <div id="terminal"></div>
    </div>
    
    <script>
        // Terminal setup
        const term = new Terminal({
            cursorBlink: true,
            fontSize: 14,
            fontFamily: 'Menlo, "DejaVu Sans Mono", Consolas, monospace',
            theme: {
                background: '#1e1e1e',
                foreground: '#f0f0f0',
                cursor: '#f0f0f0',
                selectionBackground: '#4e4e4e'
            },
            scrollback: 5000,
            convertEol: true
        });
        
        // Terminal addons
        const fitAddon = new FitAddon.FitAddon();
        const webLinksAddon = new WebLinksAddon.WebLinksAddon();
        
        // Apply addons
        term.loadAddon(fitAddon);
        term.loadAddon(webLinksAddon);
        
        // Open terminal
        term.open(document.getElementById('terminal'));
        fitAddon.fit();
        
        // Handle window resize
        window.addEventListener('resize', () => {
            fitAddon.fit();
        });
        
        // Status and control elements
        const status = document.getElementById('status');
        const clearBtn = document.getElementById('clearBtn');
        const hexModeBtn = document.getElementById('hexModeBtn');
        const localEchoBtn = document.getElementById('localEchoBtn');
        
        // Variables
        let ws = null;
        let hexMode = false;
        let localEcho = false; // Default to off
        let inputBuffer = [];
        
        // Update initial button state for local echo
        localEchoBtn.textContent = 'Local Echo: Off';
        
        // Wait for terminal to initialize before showing welcome message
        setTimeout(() => {
            // Use direct ANSI escape sequences for colors
            term.write('\r\n\u001B[1;33mSerial Terminal - Xterm.js Emulation\u001B[0m\r\n');
            term.write('\u001B[32m• Type commands and press Enter to send\r\n');
            term.write('• Press Ctrl+C to interrupt\r\n');
            term.write('• Toggle Hex Mode for binary data\r\n');
            term.write('• Toggle Local Echo for input visibility\u001B[0m\r\n\r\n');
        }, 300); // Give terminal time to initialize
        
        // Connect to WebSocket
        function connect() {
            const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            const wsUrl = wsProtocol + '//' + window.location.host + '/ws';
            
            status.textContent = 'Connecting...';
            status.className = 'status connecting';
            
            ws = new WebSocket(wsUrl);
            ws.binaryType = 'arraybuffer';
            
            ws.onopen = () => {
                status.textContent = 'Connected';
                status.className = 'status connected';
                term.writeln('\\x1b[32mConnected to serial port\\x1b[0m');
            };
            
            ws.onclose = () => {
                status.textContent = 'Disconnected';
                status.className = 'status';
                term.writeln('\\x1b[31mDisconnected from serial port\\x1b[0m');
                
                // Try to reconnect after a delay
                setTimeout(connect, 3000);
            };
            
            ws.onerror = (error) => {
                term.writeln('\\x1b[31mConnection error\\x1b[0m');
            };
            
            ws.onmessage = (event) => {
                const data = new Uint8Array(event.data);
                
                if (hexMode) {
                    // Display as hex
                    const hexStr = Array.from(data)
                        .map(b => b.toString(16).padStart(2, '0'))
                        .join(' ');
                    term.write('\\x1b[34m' + hexStr + ' \\x1b[0m');
                } else {
                    // Handle as text with potential ANSI codes
                    const decoder = new TextDecoder('utf-8');
                    const text = decoder.decode(data);
                    term.write(text);
                }
            };
        }
        
        // Send data to serial port
        function sendData(data) {
            if (!ws || ws.readyState !== WebSocket.OPEN) {
                term.writeln('\\x1b[31mNot connected!\\x1b[0m');
                return;
            }
            
            ws.send(data);
        }
        
        // Handle terminal input
        term.onData(data => {
            const code = data.charCodeAt(0);
            
            // Only print character if local echo is on
            if (localEcho) {
                term.write(data);
            }
            
            // Special handling for control characters
            if (code === 13) {  // Enter key
                // Send CR to the device
                sendData(new Uint8Array([13]));
            } else if (code === 127 || code === 8) {  // Backspace/Delete
                // Handle backspace - most terminals send a backspace character
                sendData(new Uint8Array([8]));
            } else if (code === 3) {  // Ctrl+C
                // Send ETX character (End of Text, ASCII 3)
                sendData(new Uint8Array([3]));
            } else {
                // For normal characters, just send the raw data
                const encoder = new TextEncoder();
                sendData(encoder.encode(data));
            }
        });
        
        // Clear button handler
        clearBtn.addEventListener('click', () => {
            term.clear();
        });
        
        // Hex mode toggle
        hexModeBtn.addEventListener('click', () => {
            hexMode = !hexMode;
            hexModeBtn.textContent = 'Hex Mode: ' + (hexMode ? 'On' : 'Off');
            term.writeln('\\x1b[33mHex Mode ' + (hexMode ? 'Enabled' : 'Disabled') + '\\x1b[0m');
        });
        
        // Local echo toggle
        localEchoBtn.addEventListener('click', () => {
            localEcho = !localEcho;
            localEchoBtn.textContent = 'Local Echo: ' + (localEcho ? 'On' : 'Off');
            term.writeln('\\x1b[33mLocal Echo ' + (localEcho ? 'Enabled' : 'Disabled') + '\\x1b[0m');
        });
        
        // Initial connection
        connect();
        
        // Focus terminal on load
        window.addEventListener('load', () => {
            term.focus();
        });
    </script>
</body>
</html>
`
