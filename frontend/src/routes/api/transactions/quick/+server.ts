import { json, type RequestHandler } from '@sveltejs/kit';
import { apiRequest, type Transaction } from '#lib/server/api.js';
import { todayInIndia } from '#lib/dates.js';
import { apiUrl } from '#lib/server/config.js';

export const POST: RequestHandler = async ({ request, fetch }) => {
	const body = await request.json().catch(() => null);
	if (!body || typeof body !== 'object' || Array.isArray(body)) {
		return json({ error: 'Enter the expense details.' }, { status: 400 });
	}
	const { accountId, categoryId, amountPaise, occurredOn, note } = body;
	const result = await apiRequest<Transaction>(fetch, apiUrl(), '/expenses', {
		method: 'POST',
		headers: { 'Idempotency-Key': request.headers.get('Idempotency-Key') ?? '' },
		body: JSON.stringify({
			accountId,
			categoryId,
			amountPaise,
			occurredOn: occurredOn ?? todayInIndia(),
			note: note ?? ''
		})
	});
	return result.ok
		? json(result.data)
		: json({ error: result.error.message, code: result.error.code }, { status: result.status });
};
