import { error, redirect } from '@sveltejs/kit';
import { apiRequest } from '#lib/server/api.js';
import { apiUrl } from '#lib/server/config.js';
import type { RequestHandler } from './$types';

export const POST: RequestHandler = async ({ fetch, cookies }) => {
	const result = await apiRequest(fetch, apiUrl(), '/auth/logout', { method: 'POST' });
	if (!result.ok) error(503, 'Sign-out could not be completed. Try again.');
	cookies.delete('buywise_session', { path: '/' });
	redirect(303, '/sign-in');
};
