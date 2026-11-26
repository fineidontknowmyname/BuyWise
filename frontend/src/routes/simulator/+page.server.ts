import { apiRequest, type Goal, type Product } from '#lib/server/api.js';
import { apiUrl } from '#lib/server/config.js';
import { todayInIndia } from '#lib/dates.js';
import { addSavingMonths } from '#lib/reverseEmi.js';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ fetch }) => {
	const [goals, products] = await Promise.all([
		apiRequest<Goal[]>(fetch, apiUrl(), '/goals'),
		apiRequest<Product[]>(fetch, apiUrl(), '/products')
	]);
	const today = todayInIndia();
	return {
		today,
		firstSavingDate: addSavingMonths(today, 1) ?? today,
		goals: goals.ok
			? goals.data.filter((goal) => goal.status === 'active' || goal.status === 'paused')
			: [],
		products: products.ok ? products.data : [],
		error:
			!goals.ok || !products.ok
				? 'Some saved information is unavailable. You can still calculate a draft.'
				: null
	};
};
