import { fail } from '@sveltejs/kit';
import {
	apiRequest,
	type Account,
	type BudgetReport,
	type Category,
	type Transaction
} from '#lib/server/api.js';
import { apiUrl } from '#lib/server/config.js';
import { parseRupees } from '#lib/money.js';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ fetch, url }) => {
	const base = apiUrl();
	const month = url.searchParams.get('month') || '';
	const budgetPath = month ? `/budgets?month=${encodeURIComponent(month)}` : '/budgets';

	const [categoriesRes, budgetsRes, accountsRes] = await Promise.all([
		apiRequest<Category[]>(fetch, base, '/categories'),
		apiRequest<BudgetReport>(fetch, base, budgetPath),
		apiRequest<Account[]>(fetch, base, '/accounts')
	]);

	const categories = categoriesRes.ok ? categoriesRes.data : [];
	const budgetReport = budgetsRes.ok ? budgetsRes.data : null;
	const accounts = accountsRes.ok ? accountsRes.data : [];

	// Fetch all transactions to get exact spending & counts for all categories
	const categorySpending = new Map<string, number>();
	const categoryTxCount = new Map<string, number>();

	const currentMonthPrefix = budgetReport?.month || new Date().toISOString().slice(0, 7);

	if (accounts.length > 0) {
		await Promise.all(
			accounts.map(async (acc) => {
				const txRes = await apiRequest<Transaction[]>(
					fetch,
					base,
					`/accounts/${acc.id}/transactions`
				);
				if (txRes.ok) {
					for (const tx of txRes.data) {
						if (tx.voided) continue;
						// Total transaction count (lifetime) for safe deletion / archiving check
						categoryTxCount.set(tx.categoryId, (categoryTxCount.get(tx.categoryId) || 0) + 1);

						// Month spending
						if (tx.occurredOn.startsWith(currentMonthPrefix)) {
							const current = categorySpending.get(tx.categoryId) || 0;
							categorySpending.set(tx.categoryId, current + tx.amountPaise);
						}
					}
				}
			})
		);
	}

	return {
		categories,
		budgetReport,
		spendingMap: Object.fromEntries(categorySpending),
		txCountMap: Object.fromEntries(categoryTxCount),
		selectedMonth: currentMonthPrefix,
		error: !categoriesRes.ok
			? categoriesRes.error.message
			: !budgetsRes.ok
				? budgetsRes.error.message
				: null
	};
};

export const actions: Actions = {
	createCategory: async ({ request, fetch }) => {
		const form = await request.formData();
		const name = String(form.get('name') ?? '').trim();
		const type = String(form.get('type') ?? 'discretionary'); // discretionary, committed, income
		const monthlyLimitRupees = String(form.get('monthlyLimit') ?? '').trim();

		const kind: 'income' | 'expense' = type === 'income' ? 'income' : 'expense';
		const committed = type === 'committed';

		if (!name) {
			return fail(400, { error: 'Please enter a category name.' });
		}

		const base = apiUrl();
		const createRes = await apiRequest<Category>(fetch, base, '/categories', {
			method: 'POST',
			body: JSON.stringify({ name, kind, committed })
		});

		if (!createRes.ok) {
			return fail(createRes.status, { error: createRes.error.message });
		}

		// If a monthly limit was provided for a discretionary expense, set budget immediately
		if (monthlyLimitRupees && kind === 'expense' && !committed) {
			const limitPaise = parseRupees(monthlyLimitRupees);
			if (limitPaise !== null && limitPaise >= 0) {
				await apiRequest(fetch, base, `/budgets/${createRes.data.id}`, {
					method: 'PUT',
					body: JSON.stringify({ limitPaise })
				});
			}
		}

		return { success: true, created: createRes.data.name };
	},

	updateCategory: async ({ request, fetch }) => {
		const form = await request.formData();
		const id = String(form.get('id') ?? '');
		const name = String(form.get('name') ?? '').trim();
		const type = String(form.get('type') ?? 'discretionary');
		const archived = form.get('archived') === 'true' || form.get('archived') === 'on';

		const kind: 'income' | 'expense' = type === 'income' ? 'income' : 'expense';
		const committed = type === 'committed';

		if (!id || !name) {
			return fail(400, { error: 'Category ID and name are required.' });
		}

		const base = apiUrl();
		const res = await apiRequest<Category>(fetch, base, `/categories/${id}`, {
			method: 'PUT',
			body: JSON.stringify({ name, kind, committed, archived })
		});

		if (!res.ok) {
			return fail(res.status, { error: res.error.message });
		}

		return { success: true, updated: res.data.name };
	},

	setBudget: async ({ request, fetch }) => {
		const form = await request.formData();
		const categoryId = String(form.get('categoryId') ?? '');
		const limit = String(form.get('limit') ?? '');

		if (!categoryId) {
			return fail(400, { error: 'Category ID is required.' });
		}

		// Empty string or "none" means remove budget limit
		if (limit === '' || limit.toLowerCase() === 'none') {
			const delRes = await apiRequest(
				fetch,
				apiUrl(),
				`/budgets/${encodeURIComponent(categoryId)}`,
				{
					method: 'DELETE'
				}
			);
			if (!delRes.ok) {
				return fail(delRes.status, { error: delRes.error.message });
			}
			return { success: true, removed: true };
		}

		const limitPaise = parseRupees(limit);
		if (limitPaise === null || limitPaise < 0) {
			return fail(400, { error: 'Enter a valid amount in rupees (e.g. 5000 or 5000.50).' });
		}

		const res = await apiRequest(fetch, apiUrl(), `/budgets/${encodeURIComponent(categoryId)}`, {
			method: 'PUT',
			body: JSON.stringify({ limitPaise })
		});

		if (!res.ok) {
			return fail(res.status, { error: res.error.message });
		}

		return { success: true };
	},

	archiveCategory: async ({ request, fetch }) => {
		const form = await request.formData();
		const id = String(form.get('id') ?? '');
		const name = String(form.get('name') ?? '');
		const kind = String(form.get('kind') ?? 'expense') as 'income' | 'expense';
		const committed = form.get('committed') === 'true';
		const archived = form.get('archived') !== 'false'; // defaults to true

		const base = apiUrl();
		const res = await apiRequest<Category>(fetch, base, `/categories/${id}`, {
			method: 'PUT',
			body: JSON.stringify({ name, kind, committed, archived })
		});

		if (!res.ok) {
			return fail(res.status, { error: res.error.message });
		}

		return { success: true, archived };
	},

	deleteCategory: async ({ request, fetch }) => {
		const form = await request.formData();
		const id = String(form.get('id') ?? '');
		if (!id) return fail(400, { error: 'Category ID is required.' });

		const res = await apiRequest(fetch, apiUrl(), `/categories/${id}`, {
			method: 'DELETE'
		});

		if (!res.ok) {
			return fail(res.status, { error: res.error.message });
		}

		return { success: true, deleted: true };
	}
};
