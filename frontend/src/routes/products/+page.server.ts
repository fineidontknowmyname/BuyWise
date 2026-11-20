import { fail, redirect } from '@sveltejs/kit';
import { resolve } from '$app/paths';
import { apiRequest, type Product, type ProductStats } from '#lib/server/api.js';
import { apiUrl } from '#lib/server/config.js';
import { parseRupees } from '#lib/money.js';
import { marketDate } from '#lib/dates.js';
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
		const field = (key: string) => String(form.get(key) ?? '').trim();
		const variant = field('variant');
		const url = field('url');
		const retailer = field('retailer');
		const seller = field('seller');
		const price = field('currentPrice');
		const target = field('targetPrice');
		const itemPaise = price ? parseRupees(price) : null;
		const maximumPaise = target ? parseRupees(target) : null;
		if ((price && !itemPaise) || (target && !maximumPaise)) {
			return fail(400, { error: 'Enter valid positive prices.' });
		}
		const offer =
			url || retailer || seller
				? {
						listingUrl: url,
						retailer,
						seller,
						variant,
						variantConfirmed: field('variantConfirmed') === 'yes'
					}
				: null;
		const createRes = await apiRequest<Product>(fetch, apiUrl(), '/products/tracked', {
			method: 'POST',
			body: JSON.stringify({
				name: field('name'),
				brand: field('brand'),
				model: field('model'),
				variant,
				category: field('category') || 'General',
				notes: field('notes'),
				offer,
				maximumPaise,
				openingPrice: itemPaise
					? { itemPaise, observedOn: marketDate(), availability: 'unknown' }
					: null
			})
		});
		if (!createRes.ok) return fail(createRes.status, { error: createRes.error.message });
		redirect(303, resolve('/products/[id]/offers', { id: createRes.data.id }));
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
