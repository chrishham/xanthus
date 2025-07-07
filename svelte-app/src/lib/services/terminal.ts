import { browser } from '$app/environment';
import { api } from './api';
import { setTerminalConnected, setTerminalSessionId } from '$lib/stores/vps';
import type { VPS } from '../../../app';

let Terminal: any = null;
let FitAddon: any = null;
let WebLinksAddon: any = null;
let SearchAddon: any = null;

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
	
	if (!WebLinksAddon) {
		const { WebLinksAddon: XWebLinksAddon } = await import('@xterm/addon-web-links');
		WebLinksAddon = XWebLinksAddon;
	}
	
	if (!SearchAddon) {
		const { SearchAddon: XSearchAddon } = await import('@xterm/addon-search');
		SearchAddon = XSearchAddon;
	}
};

export interface TerminalSession {
	id: string;
	vps: VPS;
	terminal: any;
	fitAddon: any;
	webLinksAddon: any;
	searchAddon: any;
	webSocket: WebSocket | null;
	connected: boolean;
	element: HTMLElement;
	cleanup: () => void;
}

export class TerminalService {
	private sessions: Map<string, TerminalSession> = new Map();
	private reconnectAttempts: Map<string, number> = new Map();
	private maxReconnectAttempts = 5;
	private reconnectDelay = 1000;

	async createTerminalSession(vps: VPS, element: HTMLElement): Promise<TerminalSession> {
		if (!browser) {
			throw new Error('Terminal service only available in browser');
		}

		try {
			await loadXterm();

			if (!Terminal || !FitAddon || !WebLinksAddon || !SearchAddon) {
				throw new Error('Failed to load xterm.js and addons');
			}

			// Create terminal session via API
			const response = await api.post<{ session_id: string }>('/ws-terminal/create', {
				vps_id: vps.id
			});

			const sessionId = response.session_id;
			
			// Create xterm.js terminal
			const terminal = new Terminal({
				theme: {
					background: '#1a1a1a',
					foreground: '#ffffff',
					cursor: '#ffffff',
					cursorAccent: '#000000',
					selection: '#444444',
					black: '#000000',
					red: '#ff5555',
					green: '#50fa7b',
					yellow: '#f1fa8c',
					blue: '#bd93f9',
					magenta: '#ff79c6',
					cyan: '#8be9fd',
					white: '#f8f8f2',
					brightBlack: '#44475a',
					brightRed: '#ff5555',
					brightGreen: '#50fa7b',
					brightYellow: '#f1fa8c',
					brightBlue: '#bd93f9',
					brightMagenta: '#ff79c6',
					brightCyan: '#8be9fd',
					brightWhite: '#ffffff'
				},
				fontFamily: 'JetBrains Mono, Menlo, Monaco, "Courier New", monospace',
				fontSize: 14,
				fontWeight: 'normal',
				letterSpacing: 0,
				lineHeight: 1.2,
				cursorBlink: true,
				cursorStyle: 'block',
				scrollback: 1000,
				tabStopWidth: 4,
				allowTransparency: true,
				convertEol: true,
				disableStdin: false,
				windowsMode: false
			});

			// Set up addons
			const fitAddon = new FitAddon();
			const webLinksAddon = new WebLinksAddon();
			const searchAddon = new SearchAddon();

			terminal.loadAddon(fitAddon);
			terminal.loadAddon(webLinksAddon);
			terminal.loadAddon(searchAddon);

			// Open terminal in container
			terminal.open(element);
			fitAddon.fit();

			// Create session object
			const session: TerminalSession = {
				id: sessionId,
				vps,
				terminal,
				fitAddon,
				webLinksAddon,
				searchAddon,
				webSocket: null,
				connected: false,
				element,
				cleanup: () => this.cleanupSession(sessionId)
			};

			// Store session
			this.sessions.set(sessionId, session);
			
			// Connect WebSocket
			await this.connectWebSocket(sessionId);

			// Set up resize handler
			const resizeHandler = () => {
				if (session.connected) {
					fitAddon.fit();
					this.sendResize(sessionId, terminal.cols, terminal.rows);
				}
			};

			window.addEventListener('resize', resizeHandler);
			
			// Update cleanup to remove resize handler
			const originalCleanup = session.cleanup;
			session.cleanup = () => {
				window.removeEventListener('resize', resizeHandler);
				originalCleanup();
			};

			// Update store
			setTerminalSessionId(sessionId);
			
			return session;
		} catch (error) {
			console.error('Failed to create terminal session:', error);
			throw error;
		}
	}

	private async connectWebSocket(sessionId: string): Promise<void> {
		const session = this.sessions.get(sessionId);
		if (!session) {
			throw new Error(`Session ${sessionId} not found`);
		}

		const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
		const wsUrl = `${protocol}//${window.location.host}/ws/terminal/${sessionId}`;

		try {
			const webSocket = new WebSocket(wsUrl);
			session.webSocket = webSocket;

			webSocket.onopen = () => {
				console.log(`WebSocket connected for session ${sessionId}`);
				session.connected = true;
				setTerminalConnected(true);
				this.reconnectAttempts.set(sessionId, 0);
				
				// Send initial resize
				this.sendResize(sessionId, session.terminal.cols, session.terminal.rows);
				
				// Show connection message
				session.terminal.writeln('\r\n\x1b[32mConnected to server\x1b[0m\r\n');
			};

			webSocket.onmessage = (event) => {
				try {
					const data = JSON.parse(event.data);
					
					if (data.type === 'output') {
						session.terminal.write(data.data);
					} else if (data.type === 'error') {
						session.terminal.writeln(`\r\n\x1b[31mError: ${data.message}\x1b[0m\r\n`);
					} else if (data.type === 'resize') {
						// Handle resize confirmation
						console.log(`Terminal resized to ${data.cols}x${data.rows}`);
					}
				} catch (error) {
					// Handle raw output for backward compatibility
					session.terminal.write(event.data);
				}
			};

			webSocket.onclose = (event) => {
				console.log(`WebSocket closed for session ${sessionId}:`, event.code, event.reason);
				session.connected = false;
				setTerminalConnected(false);
				
				// Show disconnection message
				session.terminal.writeln('\r\n\x1b[31mConnection lost\x1b[0m\r\n');
				
				// Attempt reconnection if not intentional
				if (event.code !== 1000) {
					this.attemptReconnection(sessionId);
				}
			};

			webSocket.onerror = (error) => {
				console.error(`WebSocket error for session ${sessionId}:`, error);
				session.connected = false;
				setTerminalConnected(false);
				session.terminal.writeln('\r\n\x1b[31mConnection error\x1b[0m\r\n');
			};

			// Set up input handler
			session.terminal.onData((data: string) => {
				if (session.connected && session.webSocket) {
					session.webSocket.send(JSON.stringify({
						type: 'input',
						data: data
					}));
				}
			});

			// Set up key handler for special keys
			session.terminal.onKey((key: any) => {
				if (session.connected && session.webSocket) {
					// Handle special keys like Ctrl+C, Ctrl+D, etc.
					if (key.domEvent.ctrlKey) {
						const keyCode = key.domEvent.keyCode;
						if (keyCode === 67) { // Ctrl+C
							session.webSocket.send(JSON.stringify({
								type: 'signal',
								signal: 'SIGINT'
							}));
						} else if (keyCode === 68) { // Ctrl+D
							session.webSocket.send(JSON.stringify({
								type: 'signal',
								signal: 'SIGTERM'
							}));
						}
					}
				}
			});

		} catch (error) {
			console.error('Failed to connect WebSocket:', error);
			throw error;
		}
	}

	private async attemptReconnection(sessionId: string): Promise<void> {
		const session = this.sessions.get(sessionId);
		if (!session) return;

		const attempts = this.reconnectAttempts.get(sessionId) || 0;
		if (attempts >= this.maxReconnectAttempts) {
			session.terminal.writeln('\r\n\x1b[31mMax reconnection attempts reached\x1b[0m\r\n');
			return;
		}

		this.reconnectAttempts.set(sessionId, attempts + 1);
		
		session.terminal.writeln(`\r\n\x1b[33mReconnecting... (${attempts + 1}/${this.maxReconnectAttempts})\x1b[0m\r\n`);
		
		setTimeout(async () => {
			try {
				await this.connectWebSocket(sessionId);
			} catch (error) {
				console.error('Reconnection failed:', error);
				this.attemptReconnection(sessionId);
			}
		}, this.reconnectDelay * Math.pow(2, attempts)); // Exponential backoff
	}

	private sendResize(sessionId: string, cols: number, rows: number): void {
		const session = this.sessions.get(sessionId);
		if (session && session.connected && session.webSocket) {
			session.webSocket.send(JSON.stringify({
				type: 'resize',
				cols,
				rows
			}));
		}
	}

	async destroySession(sessionId: string): Promise<void> {
		const session = this.sessions.get(sessionId);
		if (!session) return;

		try {
			// Close WebSocket
			if (session.webSocket) {
				session.webSocket.close(1000, 'Session terminated');
			}

			// Dispose terminal
			session.terminal.dispose();

			// Clean up session via API
			await api.delete(`/ws-terminal/${sessionId}`);

			// Remove from sessions
			this.sessions.delete(sessionId);
			this.reconnectAttempts.delete(sessionId);

			// Update store
			setTerminalConnected(false);
			setTerminalSessionId('');

		} catch (error) {
			console.error('Failed to destroy terminal session:', error);
		}
	}

	private cleanupSession(sessionId: string): void {
		const session = this.sessions.get(sessionId);
		if (session) {
			if (session.webSocket) {
				session.webSocket.close(1000, 'Session cleanup');
			}
			session.terminal.dispose();
			this.sessions.delete(sessionId);
			this.reconnectAttempts.delete(sessionId);
		}
	}

	// Session management methods
	getSession(sessionId: string): TerminalSession | undefined {
		return this.sessions.get(sessionId);
	}

	getAllSessions(): TerminalSession[] {
		return Array.from(this.sessions.values());
	}

	resizeSession(sessionId: string): void {
		const session = this.sessions.get(sessionId);
		if (session && session.connected) {
			session.fitAddon.fit();
			this.sendResize(sessionId, session.terminal.cols, session.terminal.rows);
		}
	}

	focusSession(sessionId: string): void {
		const session = this.sessions.get(sessionId);
		if (session) {
			session.terminal.focus();
		}
	}

	clearSession(sessionId: string): void {
		const session = this.sessions.get(sessionId);
		if (session) {
			session.terminal.clear();
		}
	}

	searchInSession(sessionId: string, query: string): boolean {
		const session = this.sessions.get(sessionId);
		if (session && session.searchAddon) {
			return session.searchAddon.findNext(query);
		}
		return false;
	}

	// Legacy methods for backward compatibility
	async initialize(element: HTMLElement, serverId: string): Promise<void> {
		console.warn('initialize() is deprecated. Use createTerminalSession() instead.');
		const vps = { id: serverId } as VPS;
		await this.createTerminalSession(vps, element);
	}

	cleanup(): void {
		// Clean up all sessions
		for (const [sessionId] of this.sessions) {
			this.cleanupSession(sessionId);
		}
	}

	isConnected(): boolean {
		// Check if any session is connected
		for (const session of this.sessions.values()) {
			if (session.connected) {
				return true;
			}
		}
		return false;
	}
}

// Export singleton instance
export const terminalService = new TerminalService();