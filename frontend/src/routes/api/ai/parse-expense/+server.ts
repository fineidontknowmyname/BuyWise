import { json, type RequestHandler } from '@sveltejs/kit';
import { parseNaturalLanguageExpense } from '#lib/server/ai/groqClient.js';
import { apiRequest, type Category } from '#lib/server/api.js';
import { apiUrl } from '#lib/server/config.js';

export const POST: RequestHandler = async ({ request, fetch }) => {
	try {
		const { text } = await request.json();
		if (!text || typeof text !== 'string') {
			return json({ error: 'Text prompt is required.' }, { status: 400 });
		}

		// Retrieve categories to provide context to the LLM
		const categoriesRes = await apiRequest<Category[]>(fetch, apiUrl(), '/categories');
		const categories = categoriesRes.ok ? categoriesRes.data : [];
		const categoryNames = categories
			.filter((c: Category) => c.kind === 'expense')
			.map((c: Category) => c.name);

		const parsed = await parseNaturalLanguageExpense(text, categoryNames, fetch);

		// Match categoryId if available
		const matchedCategory = categories.find(
			(c: Category) => c.name.toLowerCase() === parsed.suggestedCategory.toLowerCase()
		);
		if (matchedCategory) {
			parsed.categoryId = matchedCategory.id;
		}

		return json(parsed);
	} catch (err: unknown) {
		const message =
			err instanceof Error ? err.message : 'Failed to parse natural language expense.';
		return json({ error: message }, { status: 500 });
	}
};
