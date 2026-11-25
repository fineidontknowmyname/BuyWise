import { createPurchaseGoal } from '#lib/server/createPurchaseGoal.js';
import { apiRequest, type Goal, type Product } from '#lib/server/api.js';
import { apiUrl } from '#lib/server/config.js';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ fetch }) => {
	const base = apiUrl();
	const [goalsRes, productsRes] = await Promise.all([
		apiRequest<Goal[]>(fetch, base, '/goals'),
		apiRequest<Product[]>(fetch, base, '/products')
	]);

	return {
		goals: goalsRes.ok ? goalsRes.data : [],
		products: productsRes.ok ? productsRes.data : [],
		error: goalsRes.ok ? null : goalsRes.error.message
	};
};

export const actions: Actions = { create: createPurchaseGoal };
