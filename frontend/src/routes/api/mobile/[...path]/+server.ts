import { json, type RequestHandler } from '@sveltejs/kit';
import { apiUrl } from '#lib/server/config.js';

const id = '[a-fA-F0-9-]{36}';
const reads = new RegExp(
	`^(auth/session|accounts|categories|goals|financial-profile|budgets|expenses/candidates|goals/${id}/(decision|plan))$`
);
const writes = new RegExp(
	`^(auth/(login|refresh|logout)|expenses|expenses/candidates/(notifications|${id}/(confirm|reject)))$`
);

const forward: RequestHandler = async ({ params, request, fetch, url }) => {
	const path = params.path ?? '';
	if (
		url.search ||
		request.headers.has('origin') ||
		request.headers.has('cookie') ||
		!(request.method === 'GET' ? reads : writes).test(path)
	)
		return json({ error: 'Unknown mobile endpoint.' }, { status: 404 });
	const token = request.headers.get('authorization');
	const login = request.method === 'POST' && path === 'auth/login';
	if (!login && (!token || !/^Bearer [A-Za-z0-9_-]{43}$/.test(token)))
		return json({ error: 'Sign in to continue.' }, { status: 401 });
	const reader = request.body?.getReader();
	let body: string | undefined;
	if (reader) {
		const decoder = new TextDecoder('utf-8', { fatal: true });
		let size = 0;
		body = '';
		try {
			while (true) {
				const { value, done } = await reader.read();
				if (done) break;
				size += value.length;
				if (size > 8192) {
					void reader.cancel().catch(() => {});
					return json({ error: 'Request is too large.' }, { status: 413 });
				}
				body += decoder.decode(value, { stream: true });
			}
			body += decoder.decode();
		} catch {
			return json({ error: 'Unreadable request.' }, { status: 400 });
		}
	}
	try {
		const headers = new Headers({ 'Content-Type': 'application/json' });
		if (!login && token) headers.set('Authorization', token);
		const key = request.headers.get('idempotency-key');
		if (key && path === 'expenses') headers.set('Idempotency-Key', key);
		const upstream = await fetch(`${apiUrl().replace(/\/$/, '')}/${path}`, {
			method: request.method,
			headers,
			body,
			redirect: 'error',
			signal: AbortSignal.timeout(8000)
		});
		return new Response(upstream.body, {
			status: upstream.status,
			headers: { 'Content-Type': 'application/json', 'Cache-Control': 'private, no-store' }
		});
	} catch {
		return json({ error: 'BuyWise is temporarily unavailable.' }, { status: 503 });
	}
};

export const GET = forward;
export const POST = forward;
