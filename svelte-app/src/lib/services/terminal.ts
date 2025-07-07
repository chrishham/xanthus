import { browser } from '$app/environment';

let Terminal: any = null;
let FitAddon: any = null;

// Dynamic imports for xterm (only in browser)
const loadXterm = async () => {
	if (!browser) return;
	
	if (!Terminal) {
		const { Terminal: XTerminal } = await import('@xterm/xterm');
		Terminal = XTerminal;
	}
	
	if (!FitAddon) {
		const { FitAddon: XFitAddon } = await import('@xterm/addon-fit');
		FitAddon = XFitAddon;
	}
};

export class TerminalService {
	private terminal: any = null;
	private fitAddon: any = null;
	private websocket: WebSocket | null = null;
	private reconnectAttempts = 0;
	private maxReconnectAttempts = 5;
	private reconnectDelay = 1000;

	async initialize(element: HTMLElement, serverId: string): Promise<void> {
		if (!browser) return;

		await loadXterm();

		if (!Terminal || !FitAddon) {
			throw new Error('Failed to load xterm.js');
		}

		// Create terminal instance
		this.terminal = new Terminal({
			cursorBlink: true,
			theme: {
				background: '#1a1a1a',
				foreground: '#ffffff',
				cursor: '#ffffff'
			},
			fontSize: 14,
			fontFamily: 'Menlo, Monaco, "Courier New", monospace'
		});

		// Create fit addon
		this.fitAddon = new FitAddon();
		this.terminal.loadAddon(this.fitAddon);

		// Open terminal in the provided element
		this.terminal.open(element);
		this.fitAddon.fit();

		// Handle resize
		window.addEventListener('resize', () => {
			if (this.fitAddon) {
				this.fitAddon.fit();
			}
		});

		// Connect to WebSocket
		await this.connect(serverId);
	}

	private async connect(serverId: string): Promise<void> {
		if (!browser) return;

		const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
		const wsUrl = `${protocol}//${window.location.host}/vps/${serverId}/terminal/ws`;

		try {
			this.websocket = new WebSocket(wsUrl);

			this.websocket.onopen = () => {
				console.log('Terminal WebSocket connected');
				this.reconnectAttempts = 0;
				if (this.terminal) {
					this.terminal.write('\r\n\x1b[32mConnected to server terminal\x1b[0m\r\n');
				}
			};

			this.websocket.onmessage = (event) => {
				if (this.terminal && event.data) {
					this.terminal.write(event.data);
				}
			};

			this.websocket.onclose = () => {
				console.log('Terminal WebSocket disconnected');
				if (this.terminal) {
					this.terminal.write('\r\n\x1b[31mConnection lost\x1b[0m\r\n');
				}
				this.attemptReconnect(serverId);
			};

			this.websocket.onerror = (error) => {
				console.error('Terminal WebSocket error:', error);
				if (this.terminal) {
					this.terminal.write('\r\n\x1b[31mConnection error\x1b[0m\r\n');
				}
			};

			// Handle terminal input
			if (this.terminal) {
				this.terminal.onData((data: string) => {
					if (this.websocket && this.websocket.readyState === WebSocket.OPEN) {
						this.websocket.send(data);
					}
				});
			}
		} catch (error) {
			console.error('Failed to connect to terminal WebSocket:', error);
			if (this.terminal) {
				this.terminal.write('\r\n\x1b[31mFailed to connect to terminal\x1b[0m\r\n');
			}
		}
	}

	private attemptReconnect(serverId: string): void {
		if (this.reconnectAttempts >= this.maxReconnectAttempts) {
			if (this.terminal) {
				this.terminal.write('\r\n\x1b[31mMax reconnection attempts reached\x1b[0m\r\n');
			}
			return;
		}

		this.reconnectAttempts++;
		const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1);

		if (this.terminal) {
			this.terminal.write(`\r\n\x1b[33mReconnecting in ${delay/1000}s... (attempt ${this.reconnectAttempts}/${this.maxReconnectAttempts})\x1b[0m\r\n`);
		}

		setTimeout(() => {
			this.connect(serverId);
		}, delay);
	}

	resize(): void {
		if (this.fitAddon) {
			this.fitAddon.fit();
		}
	}

	cleanup(): void {
		if (this.websocket) {
			this.websocket.close();
			this.websocket = null;
		}

		if (this.terminal) {
			this.terminal.dispose();
			this.terminal = null;
		}

		this.fitAddon = null;
		this.reconnectAttempts = 0;
	}

	isConnected(): boolean {
		return this.websocket !== null && this.websocket.readyState === WebSocket.OPEN;
	}
}

// Singleton instance
let terminalService: TerminalService | null = null;

export const getTerminalService = (): TerminalService => {
	if (!terminalService) {
		terminalService = new TerminalService();
	}
	return terminalService;
};