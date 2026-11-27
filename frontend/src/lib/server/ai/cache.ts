// Smart in-memory TTL response cache to minimize AI token usage

interface CacheEntry<T> {
	value: T;
	expiresAt: number;
}

class AICache {
	private store = new Map<string, CacheEntry<unknown>>();
	private defaultTTLMs = 5 * 60 * 1000; // 5 minutes

	get<T>(key: string): T | null {
		const entry = this.store.get(key);
		if (!entry) return null;
		if (Date.now() > entry.expiresAt) {
			this.store.delete(key);
			return null;
		}
		return entry.value as T;
	}

	set<T>(key: string, value: T, ttlMs = this.defaultTTLMs): void {
		// Prune if cache gets too large (hosting-friendly memory management)
		if (this.store.size > 200) {
			const now = Date.now();
			for (const [k, v] of this.store.entries()) {
				if (now > v.expiresAt) this.store.delete(k);
			}
		}

		this.store.set(key, {
			value,
			expiresAt: Date.now() + ttlMs
		});
	}

	delete(key: string): void {
		this.store.delete(key);
	}

	clear(): void {
		this.store.clear();
	}
}

export const aiCache = new AICache();
