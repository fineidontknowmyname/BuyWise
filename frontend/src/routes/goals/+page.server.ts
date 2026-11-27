import { createPurchaseGoal } from '#lib/server/createPurchaseGoal.js';
import { apiRequest, type Goal, type Product } from '#lib/server/api.js';
import { apiUrl } from '#lib/server/config.js';
import type { MoneyPlan } from '#lib/moneyPlan.js';
import { todayInIndia } from '#lib/dates.js';
import { addSavingMonths } from '#lib/reverseEmi.js';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ fetch, url }) => {
	const base = apiUrl();
	const [goalsRes, productsRes, moneyRes, accountsRes] = await Promise.all([
		apiRequest<Goal[]>(fetch, base, '/goals'),
		apiRequest<Product[]>(fetch, base, '/products'),
		apiRequest<MoneyPlan>(fetch, base, '/money-plan'),
		apiRequest<import('#lib/server/api.js').Account[]>(fetch, base, '/accounts')
	]);

	return {
		draft: {
			open: url.searchParams.get('create') === 'yes',
			target: url.searchParams.get('target') ?? '',
			saved: url.searchParams.get('saved') ?? '0',
			monthly: url.searchParams.get('monthly') ?? '',
			firstDate:
				url.searchParams.get('firstDate') || addSavingMonths(todayInIndia(), 1) || todayInIndia(),
			productId: url.searchParams.get('productId') ?? ''
		},
		money: moneyRes.ok ? moneyRes.data : null,
		accounts: accountsRes.ok ? accountsRes.data : [],
		goals: goalsRes.ok ? goalsRes.data : [],
		products: productsRes.ok ? productsRes.data : [],
		error: goalsRes.ok ? null : goalsRes.error.message
	};
};

export const actions: Actions = { create: createPurchaseGoal };
