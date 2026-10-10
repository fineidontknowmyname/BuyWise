import { error, fail, redirect } from '@sveltejs/kit';
import { resolve } from '$app/paths';
import {
	apiRequest,
	type Account,
	type ApiResult,
	type Category,
	type Transaction
} from '#lib/server/api.js';
import { apiUrl } from '#lib/server/config.js';
import { todayInIndia } from '#lib/dates.js';
import { parseRupees } from '#lib/money.js';
import type { Actions, PageServerLoad } from './$types';

function unwrap<T>(result: ApiResult<T>): T {
	if (!result.ok) {
		error(result.status === 404 ? 404 : 503, result.error.message);
	}
	return result.data;
}

export const load: PageServerLoad = async ({ fetch, params }) => {
	const id = encodeURIComponent(params.id);
	const [account, transactions, categories] = await Promise.all([
		apiRequest<Account>(fetch, apiUrl(), `/accounts/${id}`),
		apiRequest<Transaction[]>(fetch, apiUrl(), `/accounts/${id}/transactions`),
		apiRequest<Category[]>(fetch, apiUrl(), '/categories')
	]);
	return {
		account: unwrap(account),
		transactions: unwrap(transactions),
		categories: unwrap(categories),
		today: todayInIndia(),
		idempotencyKey: crypto.randomUUID()
	};
};

export const actions: Actions = {
	add: async ({ request, fetch, params }) => {
		const form = await request.formData();
		const values = {
			categoryId: String(form.get('categoryId') ?? ''),
			amount: String(form.get('amount') ?? ''),
			occurredOn: String(form.get('occurredOn') ?? ''),
			note: String(form.get('note') ?? '')
		};
		const key = String(form.get('idempotencyKey') ?? '');

		const amountPaise = parseRupees(values.amount);
		if (amountPaise === null || amountPaise < 1) {
			return fail(400, { ...values, error: 'Enter an amount above zero, like 600 or 600.50.' });
		}
		const categories = await apiRequest<Category[]>(fetch, apiUrl(), '/categories');
		if (!categories.ok) {
			return fail(categories.status, { ...values, error: categories.error.message });
		}
		const category = categories.data.find((c) => c.id === values.categoryId);
		if (!category) {
			return fail(400, { ...values, error: 'Choose a category.' });
		}

		const result = await apiRequest<Transaction>(fetch, apiUrl(), '/transactions', {
			method: 'POST',
			headers: { 'Idempotency-Key': key },
			body: JSON.stringify({
				accountId: params.id,
				categoryId: category.id,
				kind: category.kind,
				amountPaise,
				occurredOn: values.occurredOn,
				note: values.note
			})
		});
		if (!result.ok) {
			return fail(result.status, { ...values, error: result.error.message });
		}
		redirect(303, resolve('/accounts/[id]', { id: params.id }));
	},

	void: async ({ request, fetch, params }) => {
		const form = await request.formData();
		const transactionId = String(form.get('transactionId') ?? '');
		const result = await apiRequest<Transaction>(
			fetch,
			apiUrl(),
			`/transactions/${encodeURIComponent(transactionId)}/void`,
			{ method: 'POST', body: JSON.stringify({}) }
		);
		if (!result.ok && result.error.code !== 'already_voided') {
			return fail(result.status, { error: result.error.message });
		}
		redirect(303, resolve('/accounts/[id]', { id: params.id }));
	}
};
