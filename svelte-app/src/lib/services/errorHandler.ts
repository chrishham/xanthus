import { browser } from '$app/environment';

export interface ErrorDetails {
	message: string;
	code?: string | number;
	timestamp: number;
	context?: string;
	action?: string;
	retryable?: boolean;
}

export interface ErrorNotification {
	id: string;
	type: 'error' | 'warning' | 'info' | 'success';
	title: string;
	message: string;
	timestamp: number;
	dismissible: boolean;
	autoClose?: number; // seconds
	actions?: Array<{
		label: string;
		action: () => void;
		primary?: boolean;
	}>;
}

export class ErrorHandlerService {
	private notifications: ErrorNotification[] = [];
	private notificationListeners: Array<(notifications: ErrorNotification[]) => void> = [];
	private maxNotifications = 5;

	/**
	 * Handle API errors with contextual information
	 */
	async handleAPIError(error: any, context: string, retryAction?: () => Promise<void>): Promise<void> {
		console.error(`API Error in ${context}:`, error);

		const errorDetails: ErrorDetails = {
			message: this.extractErrorMessage(error),
			code: error.status || error.code,
			timestamp: Date.now(),
			context,
			retryable: this.isRetryableError(error)
		};

		// Show user-friendly notification
		await this.showErrorNotification(errorDetails, retryAction);

		// Log detailed error for debugging
		this.logError(errorDetails, error);
	}

	/**
	 * Handle network errors
	 */
	async handleNetworkError(context: string, retryAction?: () => Promise<void>): Promise<void> {
		const errorDetails: ErrorDetails = {
			message: 'Network connection failed. Please check your internet connection.',
			timestamp: Date.now(),
			context,
			retryable: true
		};

		await this.showErrorNotification(errorDetails, retryAction);
	}

	/**
	 * Handle validation errors
	 */
	async handleValidationError(errors: string[] | string, context: string): Promise<void> {
		const errorArray = Array.isArray(errors) ? errors : [errors];
		const message = errorArray.length === 1 ? errorArray[0] : `${errorArray.length} validation errors occurred`;

		const notification: ErrorNotification = {
			id: this.generateId(),
			type: 'warning',
			title: 'Validation Error',
			message,
			timestamp: Date.now(),
			dismissible: true,
			autoClose: 8
		};

		this.addNotification(notification);
	}

	/**
	 * Show success message
	 */
	showSuccess(title: string, message: string, autoClose = 5): void {
		const notification: ErrorNotification = {
			id: this.generateId(),
			type: 'success',
			title,
			message,
			timestamp: Date.now(),
			dismissible: true,
			autoClose
		};

		this.addNotification(notification);
	}

	/**
	 * Show info message
	 */
	showInfo(title: string, message: string, autoClose = 8): void {
		const notification: ErrorNotification = {
			id: this.generateId(),
			type: 'info',
			title,
			message,
			timestamp: Date.now(),
			dismissible: true,
			autoClose
		};

		this.addNotification(notification);
	}

	/**
	 * Show warning message
	 */
	showWarning(title: string, message: string, autoClose = 10): void {
		const notification: ErrorNotification = {
			id: this.generateId(),
			type: 'warning',
			title,
			message,
			timestamp: Date.now(),
			dismissible: true,
			autoClose
		};

		this.addNotification(notification);
	}

	/**
	 * Subscribe to notification updates
	 */
	subscribe(listener: (notifications: ErrorNotification[]) => void): () => void {
		this.notificationListeners.push(listener);
		listener([...this.notifications]);

		return () => {
			const index = this.notificationListeners.indexOf(listener);
			if (index > -1) {
				this.notificationListeners.splice(index, 1);
			}
		};
	}

	/**
	 * Dismiss a notification
	 */
	dismiss(id: string): void {
		this.notifications = this.notifications.filter(n => n.id !== id);
		this.notifyListeners();
	}

	/**
	 * Clear all notifications
	 */
	clearAll(): void {
		this.notifications = [];
		this.notifyListeners();
	}

	private async showErrorNotification(error: ErrorDetails, retryAction?: () => Promise<void>): Promise<void> {
		const actions: Array<{ label: string; action: () => void; primary?: boolean }> = [];

		if (retryAction && error.retryable) {
			actions.push({
				label: 'Retry',
				action: retryAction,
				primary: true
			});
		}

		const notification: ErrorNotification = {
			id: this.generateId(),
			type: 'error',
			title: this.getErrorTitle(error),
			message: error.message,
			timestamp: error.timestamp,
			dismissible: true,
			actions: actions.length > 0 ? actions : undefined
		};

		this.addNotification(notification);

		// Show SweetAlert for critical errors
		if (this.isCriticalError(error) && browser) {
			try {
				const { default: Swal } = await import('sweetalert2');
				await Swal.fire({
					title: 'Critical Error',
					text: error.message,
					icon: 'error',
					confirmButtonColor: '#ef4444'
				});
			} catch (e) {
				console.error('Failed to show critical error dialog:', e);
			}
		}
	}

	private extractErrorMessage(error: any): string {
		if (typeof error === 'string') return error;
		if (error?.message) return error.message;
		if (error?.error) return error.error;
		if (error?.details) return error.details;
		if (error?.statusText) return error.statusText;
		
		// API error response format
		if (error?.response?.data?.error) return error.response.data.error;
		if (error?.response?.data?.message) return error.response.data.message;

		return 'An unexpected error occurred';
	}

	private isRetryableError(error: any): boolean {
		const status = error?.status || error?.response?.status;
		
		// Network errors
		if (!status) return true;
		
		// Server errors (5xx) are retryable
		if (status >= 500) return true;
		
		// Rate limiting
		if (status === 429) return true;
		
		// Timeout errors
		if (status === 408) return true;
		
		return false;
	}

	private isCriticalError(error: ErrorDetails): boolean {
		// Authentication errors
		if (error.code === 401 || error.code === 403) return true;
		
		// Server errors
		if (typeof error.code === 'number' && error.code >= 500) return true;
		
		// Context-specific critical errors
		if (error.context?.includes('deployment') && error.message.includes('failed')) return true;
		if (error.context?.includes('vps') && error.message.includes('unreachable')) return true;
		
		return false;
	}

	private getErrorTitle(error: ErrorDetails): string {
		if (error.code === 401) return 'Authentication Required';
		if (error.code === 403) return 'Access Denied';
		if (error.code === 404) return 'Resource Not Found';
		if (error.code === 429) return 'Rate Limited';
		if (typeof error.code === 'number' && error.code >= 500) return 'Server Error';
		
		if (error.context) {
			const contextMap: Record<string, string> = {
				'vps': 'VPS Error',
				'applications': 'Application Error',
				'dns': 'DNS Error',
				'version': 'Version Management Error',
				'auth': 'Authentication Error'
			};
			
			for (const [key, title] of Object.entries(contextMap)) {
				if (error.context.toLowerCase().includes(key)) {
					return title;
				}
			}
		}
		
		return 'Error';
	}

	private addNotification(notification: ErrorNotification): void {
		this.notifications.unshift(notification);
		
		// Limit number of notifications
		if (this.notifications.length > this.maxNotifications) {
			this.notifications = this.notifications.slice(0, this.maxNotifications);
		}
		
		// Auto-close if specified
		if (notification.autoClose) {
			setTimeout(() => {
				this.dismiss(notification.id);
			}, notification.autoClose * 1000);
		}
		
		this.notifyListeners();
	}

	private notifyListeners(): void {
		this.notificationListeners.forEach(listener => {
			listener([...this.notifications]);
		});
	}

	private logError(error: ErrorDetails, originalError?: any): void {
		const logData = {
			timestamp: new Date(error.timestamp).toISOString(),
			context: error.context,
			message: error.message,
			code: error.code,
			userAgent: browser ? navigator.userAgent : 'server',
			url: browser ? window.location.href : 'server',
			originalError: originalError ? JSON.stringify(originalError, null, 2) : undefined
		};

		console.group(`🚨 Error in ${error.context}`);
		console.error('Error Details:', logData);
		if (originalError) {
			console.error('Original Error:', originalError);
		}
		console.groupEnd();

		// In production, you might want to send this to a logging service
		if (browser && window.location.hostname !== 'localhost') {
			// Send to logging service
			// this.sendToLoggingService(logData);
		}
	}

	private generateId(): string {
		return Math.random().toString(36).substr(2, 9);
	}
}

export const errorHandler = new ErrorHandlerService();