import { json, type RequestHandler } from '@sveltejs/kit';
import { apiRequest, type Transaction, type Category } from '#lib/server/api.js';
import { apiUrl } from '#lib/server/config.js';

export const POST: RequestHandler = async ({ request, fetch }) => {
	try {
		const body = await request.json();
		const { accountId, categoryId, amountPaise, occurredOn, note } = body;

		if (!accountId || !categoryId || !amountPaise || amountPaise <= 0) {
			return json(
				{ error: 'Account, Category, and a positive amount are required.' },
				{ status: 400 }
			);
		}

		// Retrieve category to verify kind
		const categoriesRes = await apiRequest<Category[]>(fetch, apiUrl(), '/categories');
		const categories = categoriesRes.ok ? categoriesRes.data : [];
		const cat = categories.find((c: Category) => c.id === categoryId);
		const kind = cat ? cat.kind : 'expense';

		const res = await apiRequest<Transaction>(fetch, apiUrl(), '/transactions', {
			method: 'POST',
			headers: { 'Idempotency-Key': crypto.randomUUID() },
			body: JSON.stringify({
				accountId,
				categoryId,
				kind,
				amountPaise,
				occurredOn: occurredOn || new Date().toISOString().split('T')[0],
				note: note || ''
			})
		});

		if (!res.ok) {
			return json({ error: res.error.message }, { status: res.status });
		}

		return json(res.data);
	} catch (err: unknown) {
		const message = err instanceof Error ? err.message : 'Failed to create transaction.';
		return json({ error: message }, { status: 500 });
	}
};
