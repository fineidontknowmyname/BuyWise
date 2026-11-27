import { dev } from '$app/env';
import { error, fail, redirect } from '@sveltejs/kit';
import { resolve } from '$app/paths';
import { apiRequest } from '#lib/server/api.js';
import { apiUrl } from '#lib/server/config.js';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ cookies, fetch, setHeaders }) => {
	setHeaders({ 'Cache-Control': 'private, no-store' });
	if (!cookies.get('buywise_session')) return {};
	const session = await apiRequest<{ id: string }>(fetch, apiUrl(), '/auth/session');
	if (session.ok) redirect(303, resolve('/'));
	if (session.status !== 401) error(503, 'BuyWise is temporarily unavailable. Please try again.');
	cookies.delete('buywise_session', { path: '/' });
	return {};
};

export const actions = {
	default: async ({ request, fetch, cookies, url }) => {
		const data = await request.formData();
		const email = data.get('email');
		const password = data.get('password');
		const register = data.get('intent') === 'register';
		if (typeof email !== 'string' || typeof password !== 'string') {
			return fail(400, { message: 'Enter your email and password.', register });
		}
		const result = await apiRequest<{ token: string; expiresIn: number }>(
			fetch,
			apiUrl(),
			register ? '/auth/register' : '/auth/login',
			{ method: 'POST', body: JSON.stringify({ email, password }) }
		);
		if (!result.ok) return fail(result.status, { message: result.error.message, register });
		cookies.set('buywise_session', result.data.token, {
			path: '/',
			httpOnly: true,
			secure: !dev || !['localhost', '127.0.0.1', '[::1]'].includes(url.hostname),
			sameSite: 'lax',
			maxAge: result.data.expiresIn
		});
		redirect(303, resolve('/'));
	}
} satisfies Actions;
