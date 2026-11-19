import { json, type RequestHandler } from '@sveltejs/kit';
import { generatePriceAnalysis } from '#lib/server/ai/groqClient.js';

export const POST: RequestHandler = async ({ request, fetch }) => {
	try {
		const { productName, stats } = await request.json();
		if (!productName || !stats) {
			return json({ error: 'productName and stats are required.' }, { status: 400 });
		}

		const analysis = await generatePriceAnalysis(productName, stats, fetch);
		return json({ analysis });
	} catch (err: unknown) {
		const message = err instanceof Error ? err.message : 'Failed to generate price analysis.';
		return json({ error: message }, { status: 500 });
	}
};
