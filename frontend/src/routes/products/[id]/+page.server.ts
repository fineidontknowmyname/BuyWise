import { fail, redirect } from '@sveltejs/kit';
import { resolve } from '$app/paths';
import {
	apiRequest,
	type Product,
	type PriceObservation,
	type ProductStats,
	type Goal
} from '#lib/server/api.js';
import { apiUrl } from '#lib/server/config.js';
import { parsePriceHistoryCSV, type ParsedPriceRow } from '#lib/csvPriceParser.js';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ params, fetch }) => {
	const productId = params.id;

	const [productRes, pricesRes, statsRes, goalsRes] = await Promise.all([
		apiRequest<Product>(fetch, apiUrl(), `/products/${productId}`),
		apiRequest<PriceObservation[]>(fetch, apiUrl(), `/products/${productId}/prices`),
		apiRequest<ProductStats>(fetch, apiUrl(), `/products/${productId}/stats`),
		apiRequest<Goal[]>(fetch, apiUrl(), '/goals')
	]);

	if (!productRes.ok) {
		redirect(303, resolve('/products'));
	}

	return {
		product: productRes.data,
		prices: pricesRes.ok ? pricesRes.data : [],
		stats: statsRes.ok ? statsRes.data : null,
		goals: goalsRes.ok ? goalsRes.data : []
	};
};

export const actions: Actions = {
	addPrice: async ({ params, request, fetch }) => {
		const productId = params.id;
		const form = await request.formData();
		const priceRupees = parseFloat(String(form.get('price') ?? '0'));
		const observedOn = String(form.get('observedOn') ?? new Date().toISOString().split('T')[0]);
		const source = String(form.get('source') ?? 'Manual entry').trim();

		if (isNaN(priceRupees) || priceRupees <= 0) {
			return fail(400, { error: 'Please enter a valid price greater than zero.' });
		}

		const pricePaise = Math.round(priceRupees * 100);

		const res = await apiRequest(fetch, apiUrl(), `/products/${productId}/prices`, {
			method: 'POST',
			body: JSON.stringify({
				pricePaise,
				observedOn,
				source
			})
		});

		if (!res.ok) {
			return fail(res.status, { error: res.error.message });
		}

		return { success: true };
	},

	importCsv: async ({ params, request, fetch }) => {
		const productId = params.id;
		const form = await request.formData();
		const csvFile = form.get('csvFile') as File | null;
		const csvText = String(form.get('csvText') ?? '');

		let rawContent: string;
		let filename: string;

		if (csvFile && csvFile.size > 0) {
			rawContent = await csvFile.text();
			filename = csvFile.name;
		} else if (csvText.trim()) {
			rawContent = csvText.trim();
			filename = 'manual-paste.csv';
		} else {
			return fail(400, { error: 'Please upload a CSV file or paste CSV text.' });
		}

		const parsed = parsePriceHistoryCSV(rawContent);

		if (parsed.errorRows.length > 0 && parsed.validRows.length === 0) {
			return fail(400, {
				error: `CSV parsing failed: ${parsed.errorRows
					.slice(0, 3)
					.map((e) => e.error)
					.join(', ')}`
			});
		}

		if (parsed.validRows.length === 0) {
			return fail(400, { error: 'No valid price rows found in the CSV.' });
		}

		const importPayload = {
			filename,
			observations: parsed.validRows.map((o: ParsedPriceRow) => ({
				pricePaise: o.pricePaise,
				observedOn: o.observedOn,
				source: o.source || 'CSV Import'
			}))
		};

		const res = await apiRequest(fetch, apiUrl(), `/products/${productId}/prices/import`, {
			method: 'POST',
			body: JSON.stringify(importPayload)
		});

		if (!res.ok) {
			return fail(res.status, { error: res.error.message });
		}

		return {
			success: true,
			importedCount: parsed.validRows.length,
			duplicatesSkipped: parsed.duplicateCount,
			warnings:
				parsed.errorRows.length > 0
					? parsed.errorRows.slice(0, 5).map((e) => `Row ${e.line}: ${e.error}`)
					: []
		};
	},

	update: async ({ params, request, fetch }) => {
		const productId = params.id;
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

		if (!name) return fail(400, { error: 'Name is required.' });

		const res = await apiRequest(fetch, apiUrl(), `/products/${productId}`, {
			method: 'PUT',
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

		if (!res.ok) return fail(res.status, { error: res.error.message });
		return { success: true };
	},

	delete: async ({ params, fetch }) => {
		const productId = params.id;
		const res = await apiRequest(fetch, apiUrl(), `/products/${productId}`, {
			method: 'DELETE'
		});

		if (!res.ok) {
			return fail(res.status, { error: res.error.message });
		}

		redirect(303, resolve('/products'));
	},

	createGoal: async ({ params, request, fetch }) => {
		const productId = params.id;
		const form = await request.formData();
		const name = String(form.get('name') ?? '').trim();
		const targetPriceRupees = parseFloat(String(form.get('targetPrice') ?? '0'));
		const priority = String(form.get('priority') ?? 'medium');
		const targetDate = String(form.get('targetDate') ?? '') || null;

		const targetPricePaise =
			isNaN(targetPriceRupees) || targetPriceRupees <= 0 ? 0 : Math.round(targetPriceRupees * 100);

		const res = await apiRequest(fetch, apiUrl(), '/goals', {
			method: 'POST',
			body: JSON.stringify({
				name: name || 'Purchase Goal',
				targetPricePaise,
				savedAmountPaise: 0,
				priority,
				targetDate,
				productId,
				status: 'active'
			})
		});

		if (!res.ok) {
			return fail(res.status, { error: res.error.message });
		}

		return { goalCreated: true };
	}
};
