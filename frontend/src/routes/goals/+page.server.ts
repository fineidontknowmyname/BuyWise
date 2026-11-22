import { fail, redirect } from '@sveltejs/kit';
import { resolve } from '$app/paths';
import { apiRequest, type Goal, type Product } from '#lib/server/api.js';
import { apiUrl } from '#lib/server/config.js';
import { parseRupees } from '#lib/money.js';
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

export const actions: Actions = {
	create: async ({ request, fetch }) => {
		const form = await request.formData();
		const field = (key: string) => String(form.get(key) ?? '').trim();
		const name = field('name');
		const productUrl = field('productUrl');
		const productName = field('productName');
		const variant = field('variant');
		const target = field('targetPrice');
		const targetPricePaise = target ? parseRupees(target) : 0;
		let productId = field('productId') || null;
		if (!name || targetPricePaise === null) {
			return fail(400, { name, error: 'Enter a goal name and a valid target price.' });
		}

		const base = apiUrl();
		if (!productId && (productUrl || productName)) {
			const created = await apiRequest<Product>(fetch, base, '/products/tracked', {
				method: 'POST',
				body: JSON.stringify({
					name: productName || name,
					variant,
					category: 'Shopping',
					notes: field('notes'),
					offer: productUrl
						? {
								listingUrl: productUrl,
								variant,
								variantConfirmed: field('variantConfirmed') === 'yes'
							}
						: null
				})
			});
			if (!created.ok) return fail(created.status, { name, error: created.error.message });
			productId = created.data.id;
		}
		const result = await apiRequest<Goal>(fetch, base, '/goals', {
			method: 'POST',
			body: JSON.stringify({
				name,
				targetPricePaise,
				targetDate: field('targetDate') || null,
				priority: field('priority') || 'medium',
				productId,
				status: 'active'
			})
		});
		if (!result.ok) return fail(result.status, { name, error: result.error.message });
		redirect(303, resolve('/goals/[id]/plan', { id: result.data.id }));
	}
};
