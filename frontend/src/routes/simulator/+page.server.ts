import {
	apiRequest,
	type Account,
	type Goal,
	type Product,
	type BudgetReport
} from '#lib/server/api.js';
import { apiUrl } from '#lib/server/config.js';
import { currentMonth } from '#lib/dates.js';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ fetch }) => {
	const month = currentMonth();

	const [accountsRes, budgetRes, goalsRes, productsRes] = await Promise.all([
		apiRequest<Account[]>(fetch, apiUrl(), '/accounts'),
		apiRequest<BudgetReport>(fetch, apiUrl(), `/budgets?month=${month}`),
		apiRequest<Goal[]>(fetch, apiUrl(), '/goals'),
		apiRequest<Product[]>(fetch, apiUrl(), '/products')
	]);

	const accounts = accountsRes.ok && Array.isArray(accountsRes.data) ? accountsRes.data : [];
	const totalLiquidPaise = accounts.reduce(
		(acc: number, a: Account) => acc + Math.max(0, a.balancePaise || 0),
		0
	);

	const budgetReport = budgetRes.ok ? budgetRes.data : null;
	const discretionarySurplusPaise = budgetReport
		? budgetReport.unbudgetedDiscretionaryPaise
		: 2500000; // fallback ₹25,000 if no budget set

	return {
		accounts,
		totalLiquidPaise,
		discretionarySurplusPaise,
		goals: goalsRes.ok && Array.isArray(goalsRes.data) ? goalsRes.data : [],
		products: productsRes.ok && Array.isArray(productsRes.data) ? productsRes.data : []
	};
};
