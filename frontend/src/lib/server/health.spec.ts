import { describe, expect, it } from 'vitest';
import { checkHealth, isHealthy } from './health';

const apiUrl = 'http://127.0.0.1:8080';

function respondingWith(status: number): typeof fetch {
	return async () => new Response(null, { status });
}

describe('checkHealth', () => {
	it('reports everything up when the API is ready', async () => {
		const status = await checkHealth(respondingWith(200), apiUrl);
		expect(status).toEqual({ api: 'up', database: 'up' });
		expect(isHealthy(status)).toBe(true);
	});

	it('reports the database down when the API answers 503', async () => {
		const status = await checkHealth(respondingWith(503), apiUrl);
		expect(status).toEqual({ api: 'up', database: 'down' });
		expect(isHealthy(status)).toBe(false);
	});

	it('reports the API down on an unexpected status', async () => {
		const status = await checkHealth(respondingWith(500), apiUrl);
		expect(status).toEqual({ api: 'down', database: 'unknown' });
	});

	it('reports the API down when it cannot be reached', async () => {
		const unreachable: typeof fetch = async () => {
			throw new TypeError('fetch failed');
		};
		const status = await checkHealth(unreachable, apiUrl);
		expect(status).toEqual({ api: 'down', database: 'unknown' });
	});

	it('calls the readiness endpoint on the configured API', async () => {
		let calledWith = '';
		const recording: typeof fetch = async (input) => {
			calledWith = String(input);
			return new Response(null, { status: 200 });
		};
		await checkHealth(recording, apiUrl);
		expect(calledWith).toBe('http://127.0.0.1:8080/readyz');
	});
});
