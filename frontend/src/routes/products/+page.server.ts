import { fail, redirect } from '@sveltejs/kit';
import { resolve } from '$app/paths';
import { apiRequest, type Product, type ProductStats } from '#lib/server/api.js';
import { apiUrl } from '#lib/server/config.js';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ fetch }) => {
	const productsRes = await apiRequest<Product[]>(fetch, apiUrl(), '/products');
	if (!productsRes.ok || !Array.isArray(productsRes.data)) {
		return { products: [], statsMap: {}, error: productsRes.ok ? null : productsRes.error.message };
	}

	const products = productsRes.data;
	const statsMap: Record<string, ProductStats> = {};

	// Fetch stats for products in parallel
	await Promise.all(
		products.map(async (p: Product) => {
			const sRes = await apiRequest<ProductStats>(fetch, apiUrl(), `/products/${p.id}/stats`);
			if (sRes.ok && sRes.data) {
				statsMap[p.id] = sRes.data;
			}
		})
	);

	return {
		products,
		statsMap,
		error: null
	};
};

export const actions: Actions = {
	create: async ({ request, fetch }) => {
		const form = await request.formData();
		const name = String(form.get('name') ?? '').trim();
		const brand = String(form.get('brand') ?? '').trim();
		const model = String(form.get('model') ?? '').trim();
		const category = String(form.get('category') ?? 'General').trim();
		const url = String(form.get('url') ?? '').trim();
		const imageUrl = String(form.get('imageUrl') ?? '').trim();
		const retailer = String(form.get('retailer') ?? '').trim();
		const notes = String(form.get('notes') ?? '').trim();

		const currentPriceRupees = parseFloat(String(form.get('currentPrice') ?? '0'));
		const targetPriceRupees = parseFloat(String(form.get('targetPrice') ?? '0'));

		const currentPricePaise =
			isNaN(currentPriceRupees) || currentPriceRupees < 0
				? 0
				: Math.round(currentPriceRupees * 100);
		const targetPricePaise =
			isNaN(targetPriceRupees) || targetPriceRupees < 0 ? 0 : Math.round(targetPriceRupees * 100);

		if (!name) {
			return fail(400, { error: 'Product name is required.' });
		}

		const createRes = await apiRequest<Product>(fetch, apiUrl(), '/products', {
			method: 'POST',
			body: JSON.stringify({
				name,
				brand,
				model,
				category,
				currentPricePaise,
				targetPricePaise,
				url,
				imageUrl,
				retailer,
				notes
			})
		});

		if (!createRes.ok) {
			return fail(createRes.status, { error: createRes.error.message });
		}

		const newProduct = createRes.data;

		// Record initial price observation if price provided
		if (currentPricePaise > 0) {
			const today = new Date().toISOString().split('T')[0];
			await apiRequest(fetch, apiUrl(), `/products/${newProduct.id}/prices`, {
				method: 'POST',
				body: JSON.stringify({
					pricePaise: currentPricePaise,
					observedOn: today,
					source: retailer || 'Initial entry'
				})
			});
		}

		redirect(303, resolve('/products/[id]', { id: newProduct.id }));
	},

	delete: async ({ request, fetch }) => {
		const form = await request.formData();
		const id = String(form.get('id') ?? '');
		if (!id) return fail(400, { error: 'Product ID required.' });

		const res = await apiRequest(fetch, apiUrl(), `/products/${id}`, {
			method: 'DELETE'
		});

		if (!res.ok) {
			return fail(res.status, { error: res.error.message });
		}

		return { success: true };
	}
};
