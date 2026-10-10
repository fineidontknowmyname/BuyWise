import { json } from '@sveltejs/kit';
import { apiUrl } from '#lib/server/config.js';
import { checkHealth, isHealthy } from '#lib/server/health.js';
import type { RequestHandler } from './$types';

export const GET: RequestHandler = async ({ fetch }) => {
	const health = await checkHealth(fetch, apiUrl());
	return json(health, { status: isHealthy(health) ? 200 : 503 });
};
