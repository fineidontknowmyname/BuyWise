import { describe, expect, it } from 'vitest';
import { apiRequest } from './api';

const base = 'http://127.0.0.1:8080';

function returning(status: number, body: unknown): typeof fetch {
	return async () => new Response(body === undefined ? null : JSON.stringify(body), { status });
}

describe('apiRequest', () => {
	it('returns data on success', async () => {
		const result = await apiRequest<{ id: string }>(returning(200, { id: 'x' }), base, '/goals/x');
		expect(result).toEqual({ ok: true, data: { id: 'x' } });
	});

	it('passes the API error through', async () => {
		const error = { code: 'invalid_name', message: 'Bad name.' };
		const result = await apiRequest(returning(400, { error }), base, '/goals');
		expect(result).toEqual({ ok: false, status: 400, error });
	});

	it('reports an unexpected response when the error body is missing', async () => {
		const result = await apiRequest(returning(500, undefined), base, '/goals');
		expect(result.ok).toBe(false);
		if (!result.ok) expect(result.error.code).toBe('unexpected_response');
	});

	it('reports the API as unreachable when fetch throws', async () => {
		const failing: typeof fetch = async () => {
			throw new TypeError('fetch failed');
		};
		const result = await apiRequest(failing, base, '/goals');
		expect(result).toMatchObject({ ok: false, status: 503, error: { code: 'api_unreachable' } });
	});

	it('sends JSON to the given path', async () => {
		let seenUrl = '';
		let seenType = '';
		const recording: typeof fetch = async (input, init) => {
			seenUrl = String(input);
			seenType = new Headers(init?.headers).get('Content-Type') ?? '';
			return new Response('{}', { status: 201 });
		};
		await apiRequest(recording, base, '/goals', { method: 'POST', body: '{}' });
		expect(seenUrl).toBe('http://127.0.0.1:8080/goals');
		expect(seenType).toBe('application/json');
	});
});
