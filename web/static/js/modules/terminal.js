// WebSocket Terminal Module - xterm.js implementation
export function webSocketTerminal() {
    return {
        terminal: null,
        fitAddon: null,
        webLinksAddon: null,
        socket: null,
        sessionId: null,
        isConnected: false,
        isConnecting: false,
        connectionAttempts: 0,
        maxReconnectAttempts: 5,
        reconnectDelay: 1000,

        // Initialize terminal with xterm.js
        initTerminal(containerId) {
            // Load xterm.js and addons if not already loaded
            if (typeof Terminal === 'undefined') {
                console.error('xterm.js not loaded');
                return false;
            }

            // Create terminal instance
            this.terminal = new Terminal({
                cursorBlink: true,
                theme: {
                    background: '#000000',
                    foreground: '#ffffff',
                    cursor: '#ffffff',
                    selection: '#ffffff40',
                },
                fontSize: 14,
                fontFamily: 'Consolas, "Courier New", monospace',
                rows: 24,
                cols: 80,
            });

            // Load addons
            if (typeof FitAddon !== 'undefined') {
                this.fitAddon = new FitAddon.FitAddon();
                this.terminal.loadAddon(this.fitAddon);
            }

            if (typeof WebLinksAddon !== 'undefined') {
                this.webLinksAddon = new WebLinksAddon.WebLinksAddon();
                this.terminal.loadAddon(this.webLinksAddon);
            }

            // Open terminal in container
            const container = document.getElementById(containerId);
            if (!container) {
                console.error('Terminal container not found:', containerId);
                return false;
            }

            this.terminal.open(container);

            // Fit terminal to container
            if (this.fitAddon) {
                this.fitAddon.fit();
            }

            // Handle terminal input
            this.terminal.onData((data) => {
                this.sendInput(data);
            });

            // Handle terminal resize
            this.terminal.onResize((size) => {
                this.sendResize(size.cols, size.rows);
            });

            // Handle window resize
            window.addEventListener('resize', () => {
                if (this.fitAddon) {
                    this.fitAddon.fit();
                }
            });

            console.log('Terminal initialized');
            return true;
        },

        // Connect to WebSocket terminal session
        async connectToSession(sessionId) {
            if (this.isConnecting || this.isConnected) {
                console.log('Already connecting or connected');
                return;
            }

            this.sessionId = sessionId;
            this.isConnecting = true;
            this.connectionAttempts++;

            try {
                // Determine WebSocket URL
                const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
                const host = window.location.host;
                const wsUrl = `${protocol}//${host}/ws/terminal/${sessionId}`;

                // Create WebSocket connection
                this.socket = new WebSocket(wsUrl);

                this.socket.onopen = () => {
                    this.isConnected = true;
                    this.isConnecting = false;
                    this.connectionAttempts = 0;
                    
                    if (this.terminal) {
                        this.terminal.write('\r\n\x1b[32mConnected to terminal session\x1b[0m\r\n');
                    }
                };

                this.socket.onmessage = (event) => {
                    this.handleMessage(event.data);
                };

                this.socket.onclose = (event) => {
                    this.isConnected = false;
                    this.isConnecting = false;

                    if (this.terminal && !event.wasClean) {
                        this.terminal.write('\r\n\x1b[31mConnection lost\x1b[0m\r\n');
                    }

                    // Attempt reconnection if not a clean close
                    if (!event.wasClean && this.connectionAttempts < this.maxReconnectAttempts) {
                        setTimeout(() => {
                            this.connectToSession(sessionId);
                        }, this.reconnectDelay * this.connectionAttempts);
                    }
                };

                this.socket.onerror = (error) => {
                    this.isConnecting = false;
                    
                    if (this.terminal) {
                        this.terminal.write('\r\n\x1b[31mConnection error\x1b[0m\r\n');
                    }
                };

            } catch (error) {
                this.isConnecting = false;
                
                if (this.terminal) {
                    this.terminal.write('\r\n\x1b[31mFailed to connect\x1b[0m\r\n');
                }
            }
        },

        // Handle incoming WebSocket messages
        handleMessage(data) {
            try {
                const message = JSON.parse(data);
                
                switch (message.type) {
                    case 'output':
                        if (this.terminal) {
                            this.terminal.write(message.data);
                        }
                        break;
                    
                    case 'ready':
                        if (this.terminal) {
                            this.terminal.write(`\r\n\x1b[32m${message.message}\x1b[0m\r\n`);
                        }
                        break;
                    
                    case 'error':
                        if (this.terminal) {
                            this.terminal.write(`\r\n\x1b[31mError: ${message.message}\x1b[0m\r\n`);
                        }
                        break;
                }
            } catch (error) {
                // Silently handle parse errors
            }
        },

        // Send input to terminal
        sendInput(data) {
            if (this.socket && this.isConnected) {
                const message = {
                    type: 'input',
                    data: data
                };
                this.socket.send(JSON.stringify(message));
            }
        },

        // Send terminal resize event
        sendResize(cols, rows) {
            if (this.socket && this.isConnected) {
                const message = {
                    type: 'resize',
                    data: JSON.stringify({ cols, rows })
                };
                this.socket.send(JSON.stringify(message));
            }
        },

        // Disconnect from terminal session
        disconnect() {
            if (this.socket) {
                this.socket.close(1000, 'User initiated disconnect');
                this.socket = null;
            }
            
            this.isConnected = false;
            this.isConnecting = false;
            this.sessionId = null;
            this.connectionAttempts = 0;
        },

        // Cleanup terminal instance
        destroy() {
            this.disconnect();
            
            if (this.terminal) {
                this.terminal.dispose();
                this.terminal = null;
            }
            
            this.fitAddon = null;
            this.webLinksAddon = null;
        },

        // Create a new terminal session
        async createTerminalSession(serverData) {
            try {
                const response = await fetch('/ws-terminal/create', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify({
                        server_id: serverData.serverId,
                        host: serverData.host,
                        user: serverData.user,
                        private_key: serverData.privateKey
                    })
                });

                if (!response.ok) {
                    const error = await response.json();
                    throw new Error(error.error || 'Failed to create terminal session');
                }

                const sessionData = await response.json();
                return sessionData;

            } catch (error) {
                throw error;
            }
        },

        // Stop a terminal session
        async stopTerminalSession(sessionId) {
            try {
                const response = await fetch(`/ws-terminal/${sessionId}`, {
                    method: 'DELETE',
                    credentials: 'include'
                });

                if (!response.ok) {
                    const error = await response.json();
                    throw new Error(error.error || 'Failed to stop terminal session');
                }

                return await response.json();

            } catch (error) {
                throw error;
            }
        }
    };
}