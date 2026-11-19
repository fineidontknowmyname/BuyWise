import { json, type RequestHandler } from '@sveltejs/kit';
import { extractProductFromUrl } from '#lib/server/products/linkExtractor.js';

export const POST: RequestHandler = async ({ request, fetch }) => {
	try {
		const { url } = await request.json();
		if (!url || typeof url !== 'string') {
			return json({ error: 'A valid URL is required.' }, { status: 400 });
		}

		const extracted = await extractProductFromUrl(url, fetch);
		return json(extracted);
	} catch (err: unknown) {
		const message = err instanceof Error ? err.message : 'Failed to extract product metadata.';
		return json({ error: message }, { status: 500 });
	}
};
