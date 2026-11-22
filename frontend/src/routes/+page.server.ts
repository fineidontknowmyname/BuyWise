import { fail, redirect } from '@sveltejs/kit';
import { resolve } from '$app/paths';
import {
	apiRequest,
	type Account,
	type BudgetReport,
	type Category,
	type Decision,
	type Goal,
	type Transaction
} from '#lib/server/api.js';
import { apiUrl } from '#lib/server/config.js';
import { checkHealth } from '#lib/server/health.js';
import { currentMonth, todayInIndia } from '#lib/dates.js';
import type { ActivityPage, Foundation } from '#lib/workspace.js';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ fetch }) => {
	const base = apiUrl();
	const [health, goalsRes, accountsRes, budgetsRes, categoriesRes, foundation, activity] =
		await Promise.all([
			checkHealth(fetch, base),
			apiRequest<Goal[]>(fetch, base, '/goals'),
			apiRequest<Account[]>(fetch, base, '/accounts'),
			apiRequest<BudgetReport>(fetch, base, '/budgets'),
			apiRequest<Category[]>(fetch, base, '/categories'),
			apiRequest<Foundation>(fetch, base, '/financial-profile'),
			apiRequest<ActivityPage>(fetch, base, '/activity')
		]);

	const goals = goalsRes.ok ? goalsRes.data.filter((goal) => goal.status === 'active') : [];
	const now = new Date();
	const today = todayInIndia(now);
	const hour = Number(
		new Intl.DateTimeFormat('en-GB', {
			hour: '2-digit',
			hourCycle: 'h23',
			timeZone: 'Asia/Kolkata'
		}).format(now)
	);
	const milestone =
		goals
			.filter((goal) => goal.targetDate && goal.targetDate >= today)
			.sort((a, b) => a.targetDate!.localeCompare(b.targetDate!))[0] ??
		goals[0] ??
		null;
	const decision = milestone
		? await apiRequest<Decision>(fetch, base, `/goals/${encodeURIComponent(milestone.id)}/decision`)
		: null;
	const accounts = accountsRes.ok ? accountsRes.data : [];
	const budgetReport = budgetsRes.ok ? budgetsRes.data : null;
	const categories = categoriesRes.ok ? categoriesRes.data : [];

	// Fetch transactions from accounts to show real recent activity
	let allTransactions: (Transaction & { accountName: string })[] = [];
	let ledgerComplete = accountsRes.ok;
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
				ledgerComplete = false;
				return [];
			})
		);
		allTransactions = txResults.flat().filter((t) => !t.voided);
	}

	const currentMonthPrefix = budgetReport?.month || currentMonth();

	let monthlyIncomePaise = 0;
	let monthlyExpensePaise = 0;
	const [year, month] = currentMonthPrefix.split('-').map(Number);
	const days = new Date(Date.UTC(year, month, 0)).getUTCDate();
	const weeks = Array.from({ length: Math.ceil(days / 7) }, (_, index) => ({
		label: `Days ${index * 7 + 1}–${Math.min(days, index * 7 + 7)}`,
		income: 0,
		expenses: 0
	}));

	for (const tx of allTransactions) {
		if (tx.occurredOn.startsWith(currentMonthPrefix) && tx.occurredOn <= today) {
			const week = weeks[Math.floor((Number(tx.occurredOn.slice(-2)) - 1) / 7)];
			if (tx.kind === 'income') {
				monthlyIncomePaise += tx.amountPaise;
				week.income += tx.amountPaise;
			} else if (tx.kind === 'expense') {
				monthlyExpensePaise += tx.amountPaise;
				week.expenses += tx.amountPaise;
			}
		}
	}

	const totalSavingsPaise = goals.reduce((sum, g) => sum + (g.savedAmountPaise || 0), 0);
	const totalLiquidBalancePaise = accounts.reduce((sum, a) => sum + a.balancePaise, 0);

	return {
		today,
		hour,
		weeks: ledgerComplete ? weeks : null,
		accountsAvailable: accountsRes.ok,
		goalsAvailable: goalsRes.ok,
		foundation: foundation.ok ? foundation.data : null,
		activity: activity.ok ? activity.data.events.slice(0, 6) : null,
		milestone,
		decision: decision?.ok ? decision.data : null,
		loadIncomplete: !goalsRes.ok || !ledgerComplete || !budgetsRes.ok,
		health,
		goals,
		accounts,
		budgetReport,
		categories,
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
