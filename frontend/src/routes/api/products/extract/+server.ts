import { json, type RequestHandler } from '@sveltejs/kit';
import { apiRequest } from '#lib/server/api.js';
import { apiUrl } from '#lib/server/config.js';

export const POST: RequestHandler = async ({ request, fetch }) => {
	const input = await request.json().catch(() => null);
	if (typeof input?.url !== 'string') {
		return json({ error: 'A supported HTTPS retailer link is required.' }, { status: 400 });
	}
	const result = await apiRequest<{ url: string; retailer: string }>(
		fetch,
		apiUrl(),
		'/products/validate-listing',
		{ method: 'POST', body: JSON.stringify({ url: input.url }) }
	);
	return result.ok
		? json({ ...result.data, manualDetailsRequired: true })
		: json({ error: result.error.message }, { status: result.status });
};
