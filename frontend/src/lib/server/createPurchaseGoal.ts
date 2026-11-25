import { fail, redirect } from '@sveltejs/kit';
import { resolve } from '$app/paths';
import { apiRequest, type Goal, type Product } from '#lib/server/api.js';
import { apiUrl } from '#lib/server/config.js';
import { parseRupees } from '#lib/money.js';
import type { Action } from '@sveltejs/kit';

export const createPurchaseGoal: Action = async ({ request, fetch }) => {
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
				brand: field('brand'),
				imageUrl: field('imageUrl'),
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
};
