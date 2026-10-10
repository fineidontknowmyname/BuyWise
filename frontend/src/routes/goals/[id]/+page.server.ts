import { error } from '@sveltejs/kit';
import { apiRequest, type Decision } from '#lib/server/api.js';
import { apiUrl } from '#lib/server/config.js';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ fetch, params }) => {
	const result = await apiRequest<Decision>(
		fetch,
		apiUrl(),
		`/goals/${encodeURIComponent(params.id)}/decision`
	);
	if (!result.ok) {
		error(result.status === 404 ? 404 : 503, result.error.message);
	}
	return { decision: result.data };
};
