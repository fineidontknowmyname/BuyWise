// In-memory sliding-window rate limiter for hosting-friendly Groq API calls

interface RateLimitConfig {
	maxRequestsPerMinute: number;
	cooldownSecondsOn429: number;
}

const config: RateLimitConfig = {
	maxRequestsPerMinute: 20, // Free tier friendly
	cooldownSecondsOn429: 60
};

class RateLimiter {
	private timestamps: number[] = [];
	private cooldownUntil: number = 0;

	check(): { allowed: boolean; retryAfterSeconds: number } {
		const now = Date.now();

		if (now < this.cooldownUntil) {
			const wait = Math.ceil((this.cooldownUntil - now) / 1000);
			return { allowed: false, retryAfterSeconds: wait };
		}

		// Prune timestamps older than 60 seconds
		this.timestamps = this.timestamps.filter((ts) => now - ts < 60_000);

		if (this.timestamps.length >= config.maxRequestsPerMinute) {
			const oldest = this.timestamps[0];
			const wait = Math.ceil((60_000 - (now - oldest)) / 1000);
			return { allowed: false, retryAfterSeconds: Math.max(1, wait) };
		}

		return { allowed: true, retryAfterSeconds: 0 };
	}

	recordRequest() {
		this.timestamps.push(Date.now());
	}

	triggerCooldown(seconds = config.cooldownSecondsOn429) {
		this.cooldownUntil = Date.now() + seconds * 1000;
	}

	reset() {
		this.timestamps = [];
		this.cooldownUntil = 0;
	}
}

export const aiRateLimiter = new RateLimiter();
