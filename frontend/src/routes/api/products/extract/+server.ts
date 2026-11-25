import { json, type RequestHandler } from '@sveltejs/kit';
import { apiRequest } from '#lib/server/api.js';
import { apiUrl } from '#lib/server/config.js';
import type { ProductListing } from '#lib/productListing.js';

export const POST: RequestHandler = async ({ request, fetch }) => {
	const input = await request.json().catch(() => null);
	if (typeof input?.url !== 'string') {
		return json({ error: 'A supported HTTPS retailer link is required.' }, { status: 400 });
	}
	const result = await apiRequest<ProductListing>(fetch, apiUrl(), '/products/extract', {
		method: 'POST',
		body: JSON.stringify({ url: input.url }),
		signal: AbortSignal.timeout(25000)
	});
	return result.ok
		? json(result.data)
		: json({ error: result.error.message }, { status: result.status });
};
