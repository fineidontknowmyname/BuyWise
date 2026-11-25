import {
	apiRequest,
	type Account,
	type Goal,
	type Product,
	type BudgetReport,
	type Transaction,
	type Category
} from '#lib/server/api.js';
import { apiUrl } from '#lib/server/config.js';
import { currentMonth } from '#lib/dates.js';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ fetch }) => {
	const month = currentMonth();

	const [accountsRes, budgetRes, goalsRes, productsRes, categoriesRes] = await Promise.all([
		apiRequest<Account[]>(fetch, apiUrl(), '/accounts'),
		apiRequest<BudgetReport>(fetch, apiUrl(), `/budgets?month=${month}`),
		apiRequest<Goal[]>(fetch, apiUrl(), '/goals'),
		apiRequest<Product[]>(fetch, apiUrl(), '/products'),
		apiRequest<Category[]>(fetch, apiUrl(), '/categories')
	]);

	const accounts = accountsRes.ok ? accountsRes.data : [];
	const categories = categoriesRes.ok ? categoriesRes.data : [];

	// Fetch recent transactions for all accounts
	const txPromises = accounts.map((acc: Account) =>
		apiRequest<Transaction[]>(fetch, apiUrl(), `/accounts/${acc.id}/transactions`)
	);
	const txResults = await Promise.all(txPromises);
	const allTransactions: Transaction[] = [];
	for (const res of txResults) {
		if (res.ok) {
			allTransactions.push(...res.data);
		}
	}

	// Sort transactions descending
	allTransactions.sort(
		(a, b) => new Date(b.occurredOn).getTime() - new Date(a.occurredOn).getTime()
	);

	const totalLiquidPaise = accounts.reduce(
		(acc: number, a: Account) => acc + Math.max(0, a.balancePaise),
		0
	);

	return {
		accounts,
		categories,
		totalLiquidPaise,
		budget: budgetRes.ok ? budgetRes.data : null,
		goals: goalsRes.ok && Array.isArray(goalsRes.data) ? goalsRes.data : [],
		products: productsRes.ok && Array.isArray(productsRes.data) ? productsRes.data : [],
		transactions: allTransactions.slice(0, 30)
	};
};
