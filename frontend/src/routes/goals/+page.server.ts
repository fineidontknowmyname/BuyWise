import { fail, redirect } from '@sveltejs/kit';
import { resolve } from '$app/paths';
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

export const actions: Actions = {
	create: async ({ request, fetch }) => {
		const form = await request.formData();
		const name = String(form.get('name') ?? '').trim();
		const productUrl = String(form.get('productUrl') ?? '').trim();
		const productName = String(form.get('productName') ?? '').trim();
		const brand = String(form.get('brand') ?? '').trim();
		const model = String(form.get('model') ?? '').trim();
		const variant = String(form.get('variant') ?? '').trim();
		const retailer = String(form.get('retailer') ?? '').trim();
		const imageUrl = String(form.get('imageUrl') ?? '').trim();
		const priority = String(form.get('priority') ?? 'medium');
		const targetDate = String(form.get('targetDate') ?? '') || null;
		const notes = String(form.get('notes') ?? '').trim();
		const specsRaw = String(form.get('specifications') ?? '{}');
		let productId = String(form.get('productId') ?? '') || null;

		let specifications: Record<string, string>;
		try {
			specifications = JSON.parse(specsRaw);
		} catch {
			specifications = {};
		}

		const targetPriceRupees = parseFloat(String(form.get('targetPrice') ?? '0'));
		const currentPriceRupees = parseFloat(String(form.get('currentPrice') ?? '0'));
		const originalPriceRupees = parseFloat(String(form.get('originalPrice') ?? '0'));

		const targetPricePaise =
			isNaN(targetPriceRupees) || targetPriceRupees <= 0 ? 0 : Math.round(targetPriceRupees * 100);
		const currentPricePaise =
			isNaN(currentPriceRupees) || currentPriceRupees <= 0
				? targetPricePaise
				: Math.round(currentPriceRupees * 100);
		const originalPricePaise =
			isNaN(originalPriceRupees) || originalPriceRupees <= 0
				? 0
				: Math.round(originalPriceRupees * 100);

		if (!name) {
			return fail(400, { name, error: 'Please enter a goal name.' });
		}

		const base = apiUrl();

		// If product URL was entered or product details provided, create or link product
		if ((productUrl || productName) && !productId) {
			const prodTitle = productName || name;
			const prodRes = await apiRequest<Product>(fetch, base, '/products', {
				method: 'POST',
				body: JSON.stringify({
					name: prodTitle,
					brand,
					model,
					variant,
					category: 'Shopping',
					currentPricePaise,
					originalPricePaise,
					targetPricePaise,
					specifications,
					url: productUrl,
					imageUrl,
					retailer: retailer || 'Online Store',
					notes,
					source: productUrl ? 'url_extraction' : 'manual',
					verificationStatus: productUrl ? 'verified' : 'manual_entry'
				})
			});

			if (prodRes.ok) {
				productId = prodRes.data.id;
				// Add initial price observation if price was set
				const observationPrice = currentPricePaise > 0 ? currentPricePaise : targetPricePaise;
				if (observationPrice > 0) {
					const today = new Date().toISOString().split('T')[0];
					await apiRequest(fetch, base, `/products/${productId}/prices`, {
						method: 'POST',
						body: JSON.stringify({
							pricePaise: observationPrice,
							observedOn: today,
							retailer: retailer || 'Store',
							source: productUrl ? 'url_extraction' : 'manual'
						})
					});
				}
			}
		}

		const result = await apiRequest<Goal>(fetch, base, '/goals', {
			method: 'POST',
			body: JSON.stringify({
				name,
				targetPricePaise: targetPricePaise || currentPricePaise,
				targetDate,
				priority,
				productId,
				status: 'active'
			})
		});

		if (!result.ok) {
			return fail(result.status, { name, error: result.error.message });
		}

		redirect(303, resolve('/goals/[id]/plan', { id: result.data.id }));
	}
};
