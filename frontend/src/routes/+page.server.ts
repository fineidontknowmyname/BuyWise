import { apiUrl } from '#lib/server/config.js';
import { checkHealth } from '#lib/server/health.js';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ fetch }) => {
	return { health: await checkHealth(fetch, apiUrl()) };
};
