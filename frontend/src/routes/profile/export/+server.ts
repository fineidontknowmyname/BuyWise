import { error, json, type RequestHandler } from '@sveltejs/kit';
import { apiRequest } from '#lib/server/api.js';
import { apiUrl } from '#lib/server/config.js';

export const GET: RequestHandler = async ({ fetch }) => {
	const resources = {
		profile: '/financial-profile',
		preferences: '/profile/preferences',
		goals: '/goals',
		products: '/products',
		budget: '/budgets'
	};
	const rows = await Promise.all(
		Object.entries(resources).map(async ([key, path]) => {
			const result = await apiRequest(fetch, apiUrl(), path);
			if (!result.ok) error(503, 'The workspace snapshot could not be exported.');
			return [key, result.data];
		})
	);
	return json(
		{ exportedAt: new Date().toISOString(), ...Object.fromEntries(rows) },
		{
			headers: {
				'Content-Disposition': 'attachment; filename="buywise-workspace.json"',
				'Cache-Control': 'private, no-store'
			}
		}
	);
};
