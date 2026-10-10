export type ServiceState = 'up' | 'down' | 'unknown';

export type HealthStatus = {
	api: ServiceState;
	database: ServiceState;
};

const timeoutMs = 2000;

export async function checkHealth(fetchFn: typeof fetch, apiUrl: string): Promise<HealthStatus> {
	try {
		const response = await fetchFn(`${apiUrl}/readyz`, {
			signal: AbortSignal.timeout(timeoutMs)
		});
		if (response.ok) {
			return { api: 'up', database: 'up' };
		}
		if (response.status === 503) {
			return { api: 'up', database: 'down' };
		}
		return { api: 'down', database: 'unknown' };
	} catch {
		return { api: 'down', database: 'unknown' };
	}
}

export function isHealthy(status: HealthStatus): boolean {
	return status.api === 'up' && status.database === 'up';
}
