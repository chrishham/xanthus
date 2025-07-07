import { browser } from '$app/environment';

export class AutoRefreshService {
	private intervalId: number | null = null;
	private countdownId: number | null = null;
	private refreshCallback: (() => Promise<void>) | null = null;
	private countdownCallback: ((countdown: number) => void) | null = null;
	private interval = 30000; // 30 seconds default

	constructor() {
		if (browser) {
			this.setupVisibilityHandling();
		}
	}

	start(
		refreshFn: () => Promise<void>,
		countdownFn: (countdown: number) => void,
		interval = 30000
	) {
		this.refreshCallback = refreshFn;
		this.countdownCallback = countdownFn;
		this.interval = interval;

		this.stop(); // Stop any existing intervals

		if (!browser) return;

		this.startCountdown();

		this.intervalId = window.setInterval(async () => {
			if (!document.hidden && this.refreshCallback) {
				await this.refreshCallback();
				this.startCountdown();
			}
		}, this.interval);
	}

	stop() {
		if (this.intervalId !== null) {
			clearInterval(this.intervalId);
			this.intervalId = null;
		}
		if (this.countdownId !== null) {
			clearInterval(this.countdownId);
			this.countdownId = null;
		}
		if (this.countdownCallback) {
			this.countdownCallback(0);
		}
	}

	private startCountdown() {
		if (this.countdownId !== null) {
			clearInterval(this.countdownId);
		}

		let countdown = Math.floor(this.interval / 1000);
		if (this.countdownCallback) {
			this.countdownCallback(countdown);
		}

		this.countdownId = window.setInterval(() => {
			countdown--;
			if (this.countdownCallback) {
				this.countdownCallback(countdown);
			}
			if (countdown <= 0) {
				if (this.countdownId !== null) {
					clearInterval(this.countdownId);
					this.countdownId = null;
				}
			}
		}, 1000);
	}

	private setupVisibilityHandling() {
		document.addEventListener('visibilitychange', () => {
			if (document.hidden) {
				this.stop();
			} else if (this.refreshCallback && this.countdownCallback) {
				// Restart auto-refresh when page becomes visible
				this.start(this.refreshCallback, this.countdownCallback, this.interval);
				// Refresh immediately when page becomes visible
				this.refreshCallback();
			}
		});
	}

	isRunning(): boolean {
		return this.intervalId !== null;
	}
}