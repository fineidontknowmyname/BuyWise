import { apiRequest, type Goal, type Product } from '#lib/server/api.js';
import { apiUrl } from '#lib/server/config.js';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ fetch }) => {
	const base = apiUrl();
	const [goals, products] = await Promise.all([
		apiRequest<Goal[]>(fetch, base, '/goals'),
		apiRequest<Product[]>(fetch, base, '/products')
	]);
	return {
		goals: goals.ok ? goals.data.map(({ id, name }) => ({ id, name })) : null,
		products: products.ok ? products.data.map(({ id, name }) => ({ id, name })) : null
	};
};
