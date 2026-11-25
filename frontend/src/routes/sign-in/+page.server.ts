import { dev } from '$app/env';
import { fail, redirect } from '@sveltejs/kit';
import { apiRequest } from '#lib/server/api.js';
import { apiUrl } from '#lib/server/config.js';
import type { Actions } from './$types';

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
		redirect(303, '/');
	}
} satisfies Actions;
