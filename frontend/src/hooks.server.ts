import { error, redirect } from '@sveltejs/kit';
import type { Handle, HandleFetch } from '@sveltejs/kit/hooks';
import { apiRequest } from '#lib/server/api.js';
import { apiUrl } from '#lib/server/config.js';

export const sessionCookie = 'buywise_session';
const assistantApi = '/api/ai/chat';

function sessionToken(event: Parameters<HandleFetch>[0]['event']) {
	const cookie = event.cookies.get(sessionCookie);
	if (cookie || event.url.pathname !== assistantApi) return cookie;
	return /^Bearer ([A-Za-z0-9_-]{43})$/.exec(event.request.headers.get('authorization') ?? '')?.[1];
}

export const handleFetch: HandleFetch = ({ event, request, fetch }) => {
	const base = apiUrl().replace(/\/$/, '');
	const token = sessionToken(event);
	if (token && request.url.startsWith(base + '/')) {
		const headers = new Headers(request.headers);
		headers.set('Authorization', 'Bearer ' + token);
		request = new Request(request, { headers, redirect: 'error' });
	}
	return fetch(request);
};

export const handle: Handle = async ({ event, resolve }) => {
	const path = event.url.pathname;
	if (path.startsWith('/api/mobile/')) return resolve(event);
	const cookie = event.cookies.get(sessionCookie);
	const token = sessionToken(event);
	if (!['GET', 'HEAD', 'OPTIONS'].includes(event.request.method)) {
		const origin = event.request.headers.get('origin');
		const nativeAssistant = path === assistantApi && !cookie && !!token && origin === null;
		if (origin !== event.url.origin && !nativeAssistant) {
			error(403, 'Invalid request origin.');
		}
	}
	if (path === '/sign-in' || path === '/sign-out' || path === '/health') return resolve(event);
	if (token) {
		const session = await apiRequest<{ id: string }>(event.fetch, apiUrl(), '/auth/session');
		if (session.ok) {
			event.locals.userId = session.data.id;
			const response = await resolve(event);
			const headers = new Headers(response.headers);
			headers.set('Cache-Control', 'private, no-store');
			return new Response(response.body, { status: response.status, headers });
		}
		if (session.status !== 401) error(503, 'BuyWise is temporarily unavailable.');
		if (cookie) event.cookies.delete(sessionCookie, { path: '/' });
	}
	if (path.startsWith('/api/')) error(401, 'Sign in to continue.');
	redirect(303, '/sign-in');
};
