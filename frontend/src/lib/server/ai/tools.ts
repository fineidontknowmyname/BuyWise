import {
	apiRequest,
	type Account,
	type BudgetReport,
	type Category,
	type Goal,
	type Product,
	type ProductStats,
	type Transaction
} from '#lib/server/api.js';
import { apiUrl } from '#lib/server/config.js';
import { formatRupees } from '#lib/money.js';

export const AI_TOOLS_DEFINITIONS = [
	{
		type: 'function',
		function: {
			name: 'get_financial_snapshot',
			description:
				'Retrieve current total liquid balance, active goals count, and monthly budget summary.',
			parameters: { type: 'object', properties: {} }
		}
	},
	{
		type: 'function',
		function: {
			name: 'get_budget_summary',
			description:
				'Retrieve category monthly spending limits, amount spent so far, and remaining discretionary budget.',
			parameters: { type: 'object', properties: {} }
		}
	},
	{
		type: 'function',
		function: {
			name: 'get_active_goals',
			description:
				'Retrieve all active purchase goals with their target amounts, saved progress, priority, and purchase window.',
			parameters: { type: 'object', properties: {} }
		}
	},
	{
		type: 'function',
		function: {
			name: 'get_recent_transactions',
			description: 'Retrieve recent posted transactions across accounts with category and amount.',
			parameters: {
				type: 'object',
				properties: {
					limit: { type: 'number', description: 'Number of recent transactions to return (max 10)' }
				}
			}
		}
	},
	{
		type: 'function',
		function: {
			name: 'get_product_price_summary',
			description:
				'Retrieve price history statistics (current, lowest, average) for products being tracked.',
			parameters: {
				type: 'object',
				properties: {
					productName: { type: 'string', description: 'Name of the product to look up' }
				}
			}
		}
	},
	{
		type: 'function',
		function: {
			name: 'simulate_emi_scenario',
			description:
				'Calculate deterministic monthly EMI installment, total interest, and comparison with saving upfront.',
			parameters: {
				type: 'object',
				required: ['priceRupees', 'annualRatePercent', 'months'],
				properties: {
					priceRupees: { type: 'number', description: 'Total price of product in rupees' },
					downPaymentRupees: {
						type: 'number',
						description: 'Upfront down payment in rupees (default 0)'
					},
					annualRatePercent: {
						type: 'number',
						description: 'Annual interest rate percentage (e.g. 14 for 14%)'
					},
					months: { type: 'number', description: 'Tenure in months (e.g. 6, 12, 24)' }
				}
			}
		}
	}
];

export async function executeToolCall(
	fetchFn: typeof fetch,
	name: string,
	args: Record<string, unknown>
): Promise<string> {
	const base = apiUrl();

	switch (name) {
		case 'get_financial_snapshot': {
			const [accRes, goalsRes, budgetRes] = await Promise.all([
				apiRequest<Account[]>(fetchFn, base, '/accounts'),
				apiRequest<Goal[]>(fetchFn, base, '/goals'),
				apiRequest<BudgetReport>(fetchFn, base, '/budgets')
			]);

			const accounts = accRes.ok ? accRes.data : [];
			const goals = goalsRes.ok ? goalsRes.data : [];
			const budget = budgetRes.ok ? budgetRes.data : null;

			const totalBalancePaise = accounts.reduce((sum, a) => sum + a.balancePaise, 0);

			return JSON.stringify({
				totalLiquidBalance: formatRupees(totalBalancePaise),
				activeAccountsCount: accounts.length,
				accounts: accounts.map((a) => ({ name: a.name, balance: formatRupees(a.balancePaise) })),
				activeGoalsCount: goals.length,
				monthlyBudgetLimit: budget ? formatRupees(budget.totalLimitPaise) : 'Not configured',
				monthlySpentSoFar: budget ? formatRupees(budget.totalSpentPaise) : '₹0.00',
				discretionaryRemaining: budget ? formatRupees(budget.totalRemainingPaise) : '₹0.00'
			});
		}

		case 'get_budget_summary': {
			const budgetRes = await apiRequest<BudgetReport>(fetchFn, base, '/budgets');
			if (!budgetRes.ok) return JSON.stringify({ error: budgetRes.error.message });

			const b = budgetRes.data;
			return JSON.stringify({
				month: b.month,
				totalLimit: formatRupees(b.totalLimitPaise),
				totalSpent: formatRupees(b.totalSpentPaise),
				totalRemaining: formatRupees(b.totalRemainingPaise),
				unbudgetedDiscretionary: formatRupees(b.unbudgetedDiscretionaryPaise),
				categories: b.lines.map((l) => ({
					category: l.categoryName,
					limit: formatRupees(l.limitPaise),
					spent: formatRupees(l.spentPaise),
					remaining: formatRupees(l.remainingPaise),
					isOverBudget: l.remainingPaise < 0
				}))
			});
		}

		case 'get_active_goals': {
			const goalsRes = await apiRequest<Goal[]>(fetchFn, base, '/goals');
			if (!goalsRes.ok) return JSON.stringify({ error: goalsRes.error.message });

			return JSON.stringify(
				goalsRes.data.map((g) => ({
					id: g.id,
					name: g.name,
					targetPrice: g.targetPricePaise ? formatRupees(g.targetPricePaise) : 'Not entered',
					savedAmount: g.savedAmountPaise ? formatRupees(g.savedAmountPaise) : '₹0.00',
					priority: g.priority || 'medium',
					targetDate: g.targetDate || 'None',
					status: g.status || 'active'
				}))
			);
		}

		case 'get_recent_transactions': {
			const accRes = await apiRequest<Account[]>(fetchFn, base, '/accounts');
			if (!accRes.ok || accRes.data.length === 0) return JSON.stringify([]);

			const catRes = await apiRequest<Category[]>(fetchFn, base, '/categories');
			const catMap = new Map((catRes.ok ? catRes.data : []).map((c) => [c.id, c.name]));

			const limit = Math.min(10, Number(args.limit) || 8);
			const txResults = await Promise.all(
				accRes.data.map(async (acc) => {
					const res = await apiRequest<Transaction[]>(
						fetchFn,
						base,
						`/accounts/${acc.id}/transactions`
					);
					return res.ok ? res.data.map((t) => ({ ...t, account: acc.name })) : [];
				})
			);

			const flattened = txResults
				.flat()
				.filter((t) => !t.voided)
				.sort((a, b) => new Date(b.occurredOn).getTime() - new Date(a.occurredOn).getTime())
				.slice(0, limit);

			return JSON.stringify(
				flattened.map((t) => ({
					date: t.occurredOn,
					category: catMap.get(t.categoryId) || 'General',
					account: t.account,
					type: t.kind,
					amount: formatRupees(t.amountPaise),
					note: t.note
				}))
			);
		}

		case 'get_product_price_summary': {
			const prodRes = await apiRequest<Product[]>(fetchFn, base, '/products');
			if (!prodRes.ok) return JSON.stringify({ error: prodRes.error.message });

			const query = String(args.productName || '')
				.toLowerCase()
				.trim();
			const products = prodRes.data.filter((p) => !query || p.name.toLowerCase().includes(query));

			const statsList = await Promise.all(
				products.slice(0, 3).map(async (p) => {
					const sRes = await apiRequest<ProductStats>(fetchFn, base, `/products/${p.id}/stats`);
					const s = sRes.ok ? sRes.data : null;
					return {
						name: p.name,
						currentPrice: formatRupees(p.currentPricePaise),
						targetPrice: p.targetPricePaise > 0 ? formatRupees(p.targetPricePaise) : 'None',
						lowestObserved: s && s.lowestPricePaise > 0 ? formatRupees(s.lowestPricePaise) : 'N/A',
						averageObserved:
							s && s.averagePricePaise > 0 ? formatRupees(s.averagePricePaise) : 'N/A',
						observationsCount: s ? s.observationCount : 0
					};
				})
			);

			return JSON.stringify(statsList);
		}

		case 'simulate_emi_scenario': {
			const price = Number(args.priceRupees) || 0;
			const downPayment = Number(args.downPaymentRupees) || 0;
			const annualRate = Number(args.annualRatePercent) || 0;
			const months = Math.max(1, Math.round(Number(args.months) || 12));

			const principal = Math.max(0, price - downPayment);
			let monthlyInstallment: number;
			let totalInterest = 0;

			if (annualRate <= 0) {
				monthlyInstallment = principal / months;
			} else {
				const r = annualRate / (12 * 100);
				monthlyInstallment =
					(principal * r * Math.pow(1 + r, months)) / (Math.pow(1 + r, months) - 1);
				totalInterest = monthlyInstallment * months - principal;
			}

			const totalRepayment = downPayment + principal + totalInterest;

			return JSON.stringify({
				price: `₹${price.toLocaleString('en-IN')}`,
				principalFinanced: `₹${principal.toLocaleString('en-IN')}`,
				monthlyInstallment: `₹${Math.round(monthlyInstallment).toLocaleString('en-IN')}/mo`,
				totalInterest: `₹${Math.round(totalInterest).toLocaleString('en-IN')}`,
				totalRepayment: `₹${Math.round(totalRepayment).toLocaleString('en-IN')}`,
				tenureMonths: months,
				annualRatePercent: `${annualRate}%`
			});
		}

		default:
			return JSON.stringify({ error: `Unknown tool call: ${name}` });
	}
}
