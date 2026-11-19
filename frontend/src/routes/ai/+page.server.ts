import { fail, redirect } from '@sveltejs/kit';
import { resolve } from '$app/paths';
import {
	apiRequest,
	type Account,
	type BudgetReport,
	type Category,
	type Goal,
	type Product,
	type Transaction
} from '#lib/server/api.js';
import { apiUrl, groqApiKey, groqModel } from '#lib/server/config.js';
import { currentMonth } from '#lib/dates.js';
import {
	DEFAULT_MODEL,
	generatePlanProposal,
	refinePlanProposal,
	generateFinancialOverviewAndInsights,
	type ProposedPlan
} from '#lib/server/ai/groqClient.js';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ fetch }) => {
	const base = apiUrl();
	const month = currentMonth();

	const [accountsRes, budgetRes, goalsRes, categoriesRes, productsRes] = await Promise.all([
		apiRequest<Account[]>(fetch, base, '/accounts'),
		apiRequest<BudgetReport>(fetch, base, `/budgets?month=${month}`),
		apiRequest<Goal[]>(fetch, base, '/goals'),
		apiRequest<Category[]>(fetch, base, '/categories'),
		apiRequest<Product[]>(fetch, base, '/products')
	]);

	const accounts = accountsRes.ok ? accountsRes.data : [];
	const budgetReport = budgetRes.ok ? budgetRes.data : null;
	const goals = goalsRes.ok ? goalsRes.data : [];
	const categories = categoriesRes.ok ? categoriesRes.data : [];
	const products = productsRes.ok ? productsRes.data : [];

	// Fetch recent transactions across accounts
	let recentTransactions: Transaction[] = [];
	if (accounts.length > 0) {
		const txResults = await Promise.all(
			accounts.map(async (acc) => {
				const res = await apiRequest<Transaction[]>(
					fetch,
					base,
					`/accounts/${acc.id}/transactions`
				);
				return res.ok ? res.data : [];
			})
		);
		recentTransactions = txResults
			.flat()
			.filter((t) => !t.voided)
			.sort((a, b) => new Date(b.occurredOn).getTime() - new Date(a.occurredOn).getTime())
			.slice(0, 15);
	}

	const totalLiquidPaise = accounts.reduce((sum, a) => sum + Math.max(0, a.balancePaise || 0), 0);

	const discretionaryLeftPaise = budgetReport?.totalRemainingPaise ?? 0;

	// Generate baseline overview insights
	const overviewInsights = await generateFinancialOverviewAndInsights(
		{
			accounts: accounts.map((a) => ({ name: a.name, balancePaise: a.balancePaise })),
			budgetLines: (budgetReport?.lines || []).map((l) => ({
				categoryName: l.categoryName,
				limitPaise: l.limitPaise,
				spentPaise: l.spentPaise
			})),
			recentTransactions: recentTransactions.map((t) => ({
				kind: t.kind,
				amountPaise: t.amountPaise,
				note: t.note,
				occurredOn: t.occurredOn
			})),
			totalLiquidPaise,
			discretionaryLeftPaise
		},
		fetch
	);

	return {
		accounts,
		budgetReport,
		goals,
		categories,
		products,
		recentTransactions,
		totalLiquidPaise,
		discretionaryLeftPaise,
		overviewInsights,
		aiConfig: {
			configured: Boolean(groqApiKey()),
			model: groqModel() || DEFAULT_MODEL,
			availableCapabilities: [
				{
					id: 'financial_health',
					name: 'Financial Health Analysis',
					description: 'Liquidity, savings ratio, and debt buffer'
				},
				{
					id: 'goal_optimization',
					name: 'Goal & Savings Plan Optimization',
					description: 'Priority ordering and cash flow feasibility'
				},
				{
					id: 'purchase_timing',
					name: 'Purchase Timing Recommendations',
					description: 'Optimal buying window and discount forecasting'
				},
				{
					id: 'price_history_analysis',
					name: 'Product Price & History Analysis',
					description: 'Volatility and historical markdown trends'
				},
				{
					id: 'expense_entry',
					name: 'Natural-Language Expense Entry',
					description: 'Convert conversational speech to ledger entries'
				},
				{
					id: 'transaction_categorization',
					name: 'Transaction Categorization Assistant',
					description: 'Auto-detect categories for posted transactions'
				},
				{
					id: 'budget_coaching',
					name: 'Budget Coaching',
					description: 'Tactical guidance to stay below category spending caps'
				},
				{
					id: 'anomaly_analysis',
					name: 'Spending Anomaly Analysis',
					description: 'Identify sudden surges and unusual debits'
				},
				{
					id: 'monthly_summary',
					name: 'Monthly Financial Summaries',
					description: 'Executive recap of monthly cash flow'
				},
				{
					id: 'scenario_planning',
					name: 'Scenario Planning & What-If Analysis',
					description: 'Simulate income drops, loans, or major buys'
				}
			]
		}
	};
};

export const actions: Actions = {
	generatePlan: async ({ request, fetch }) => {
		const form = await request.formData();
		const prompt = String(form.get('prompt') ?? '').trim();
		const liquidPaise = parseInt(String(form.get('liquidPaise') ?? '0'), 10);
		const discretionaryPaise = parseInt(String(form.get('discretionaryPaise') ?? '0'), 10);

		if (!prompt) {
			return fail(400, { error: 'Please describe your financial goal.' });
		}

		const plan = await generatePlanProposal(
			prompt,
			{
				liquidBalancePaise: liquidPaise,
				discretionaryLeftPaise: discretionaryPaise,
				monthlyIncomePaise: 0,
				existingGoalsCount: 0
			},
			fetch
		);

		return { success: true, plan };
	},

	refinePlan: async ({ request, fetch }) => {
		const form = await request.formData();
		const instruction = String(form.get('instruction') ?? '').trim();
		const planJson = String(form.get('currentPlan') ?? '');
		const discretionaryPaise = parseInt(String(form.get('discretionaryPaise') ?? '0'), 10);
		const liquidPaise = parseInt(String(form.get('liquidPaise') ?? '0'), 10);

		if (!instruction || !planJson) {
			return fail(400, { error: 'Instruction and plan are required.' });
		}

		let currentPlan: ProposedPlan;
		try {
			currentPlan = JSON.parse(planJson);
		} catch {
			return fail(400, { error: 'Invalid current plan.' });
		}

		const refined = await refinePlanProposal(
			currentPlan,
			instruction,
			{
				discretionaryLeftPaise: discretionaryPaise,
				liquidBalancePaise: liquidPaise
			},
			fetch
		);

		return { success: true, refinement: refined };
	},

	confirmAndSaveGoal: async ({ request, fetch }) => {
		const form = await request.formData();
		const name = String(form.get('name') ?? '').trim();
		const targetPriceRupees = parseFloat(String(form.get('targetPrice') ?? '0'));
		const targetDate = String(form.get('targetDate') ?? '') || null;
		const priority = String(form.get('priority') ?? 'medium');

		if (!name || isNaN(targetPriceRupees) || targetPriceRupees <= 0) {
			return fail(400, { error: 'Valid goal name and target amount are required.' });
		}

		const targetPricePaise = Math.round(targetPriceRupees * 100);

		const result = await apiRequest<Goal>(fetch, apiUrl(), '/goals', {
			method: 'POST',
			body: JSON.stringify({
				name,
				targetPricePaise,
				savedAmountPaise: 0,
				targetDate,
				priority,
				status: 'active'
			})
		});

		if (!result.ok) {
			return fail(result.status, { error: result.error.message });
		}

		redirect(303, resolve('/goals/[id]', { id: result.data.id }));
	}
};
