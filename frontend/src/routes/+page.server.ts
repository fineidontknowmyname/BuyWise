import { fail, redirect } from '@sveltejs/kit';
import { resolve } from '$app/paths';
import {
	apiRequest,
	type Account,
	type BudgetReport,
	type Category,
	type Goal,
	type Transaction
} from '#lib/server/api.js';
import { apiUrl } from '#lib/server/config.js';
import { checkHealth } from '#lib/server/health.js';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ fetch }) => {
	const base = apiUrl();
	const [health, goalsRes, accountsRes, budgetsRes, categoriesRes] = await Promise.all([
		checkHealth(fetch, base),
		apiRequest<Goal[]>(fetch, base, '/goals'),
		apiRequest<Account[]>(fetch, base, '/accounts'),
		apiRequest<BudgetReport>(fetch, base, '/budgets'),
		apiRequest<Category[]>(fetch, base, '/categories')
	]);

	const goals = goalsRes.ok ? goalsRes.data : [];
	const accounts = accountsRes.ok ? accountsRes.data : [];
	const budgetReport = budgetsRes.ok ? budgetsRes.data : null;
	const categories = categoriesRes.ok ? categoriesRes.data : [];

	// Fetch transactions from accounts to show real recent activity
	let allTransactions: (Transaction & { accountName: string })[] = [];
	if (accounts.length > 0) {
		const txResults = await Promise.all(
			accounts.map(async (acc) => {
				const res = await apiRequest<Transaction[]>(
					fetch,
					base,
					`/accounts/${acc.id}/transactions`
				);
				if (res.ok) {
					return res.data.map((tx) => ({ ...tx, accountName: acc.name }));
				}
				return [];
			})
		);
		allTransactions = txResults.flat().filter((t) => !t.voided);
	}

	const currentMonthPrefix = budgetReport?.month || new Date().toISOString().slice(0, 7);

	let monthlyIncomePaise = 0;
	let monthlyExpensePaise = 0;

	for (const tx of allTransactions) {
		if (tx.occurredOn.startsWith(currentMonthPrefix)) {
			if (tx.kind === 'income') {
				monthlyIncomePaise += tx.amountPaise;
			} else if (tx.kind === 'expense') {
				monthlyExpensePaise += tx.amountPaise;
			}
		}
	}

	const totalSavingsPaise = goals.reduce((sum, g) => sum + (g.savedAmountPaise || 0), 0);
	const totalLiquidBalancePaise = accounts.reduce(
		(sum, a) => sum + Math.max(0, a.balancePaise || 0),
		0
	);

	const recentTransactions = [...allTransactions]
		.sort((a, b) => new Date(b.occurredOn).getTime() - new Date(a.occurredOn).getTime())
		.slice(0, 8);

	return {
		health,
		goals,
		accounts,
		budgetReport,
		categories,
		recentTransactions,
		monthlyIncomePaise,
		monthlyExpensePaise,
		totalSavingsPaise,
		totalLiquidBalancePaise,
		currentMonthPrefix
	};
};

export const actions: Actions = {
	createGoal: async ({ request, fetch }) => {
		const form = await request.formData();
		const name = String(form.get('name') ?? '').trim();
		if (!name) {
			return fail(400, { name, error: 'Please enter a goal name.' });
		}
		const result = await apiRequest<Goal>(fetch, apiUrl(), '/goals', {
			method: 'POST',
			body: JSON.stringify({ name })
		});
		if (!result.ok) {
			return fail(result.status, { name, error: result.error.message });
		}
		redirect(303, resolve('/goals/[id]', { id: result.data.id }));
	}
};
