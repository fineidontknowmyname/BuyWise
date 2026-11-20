import { error, fail } from '@sveltejs/kit';
import { randomUUID } from 'node:crypto';
import { parseRupees } from '#lib/money.js';
import { todayInIndia } from '#lib/dates.js';
import type { PurchaseInput, PurchaseLifecycle, PurchasePreview } from '#lib/purchase.js';
import {
	apiRequest,
	type Decision,
	type Goal,
	type Product,
	type PriceObservation,
	type ProductStats,
	type DocumentUpload
} from '#lib/server/api.js';
import { apiUrl } from '#lib/server/config.js';
import { processDocumentUpload } from '#lib/server/documents/service.js';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ fetch, params }) => {
	const base = apiUrl();
	const goalId = params.id;

	const [decisionRes, goalRes, historyRes, purchaseRes] = await Promise.all([
		apiRequest<Decision>(fetch, base, `/goals/${encodeURIComponent(goalId)}/decision`),
		apiRequest<Goal>(fetch, base, `/goals/${encodeURIComponent(goalId)}`),
		apiRequest<Decision[]>(fetch, base, `/goals/${encodeURIComponent(goalId)}/decision/history`),
		apiRequest<PurchaseLifecycle>(fetch, base, `/goals/${encodeURIComponent(goalId)}/purchase`)
	]);

	if (!decisionRes.ok) {
		error(decisionRes.status === 404 ? 404 : 503, decisionRes.error.message);
	}

	const goal = goalRes.ok ? goalRes.data : null;

	let product: Product | null = null;
	let prices: PriceObservation[] = [];
	let stats: ProductStats | null = null;
	let documents: DocumentUpload[] = [];

	if (goal?.productId) {
		const [productRes, pricesRes, statsRes, docsRes] = await Promise.all([
			apiRequest<Product>(fetch, base, `/products/${goal.productId}`),
			apiRequest<PriceObservation[]>(fetch, base, `/products/${goal.productId}/prices`),
			apiRequest<ProductStats>(fetch, base, `/products/${goal.productId}/stats`),
			apiRequest<DocumentUpload[]>(fetch, base, `/goals/${encodeURIComponent(goalId)}/documents`)
		]);

		if (productRes.ok) product = productRes.data;
		if (pricesRes.ok) prices = pricesRes.data;
		if (statsRes.ok) stats = statsRes.data;
		if (docsRes.ok) documents = docsRes.data;
	} else {
		const docsRes = await apiRequest<DocumentUpload[]>(
			fetch,
			base,
			`/goals/${encodeURIComponent(goalId)}/documents`
		);
		if (docsRes.ok) documents = docsRes.data;
	}

	return {
		purchase: purchaseRes.ok ? purchaseRes.data : null,
		purchaseToday: todayInIndia(),
		allocationKey: randomUUID(),
		decision: decisionRes.data,
		decisionHistory: historyRes.ok ? historyRes.data : null,
		goal,
		product,
		prices,
		stats,
		documents
	};
};

const field = (form: FormData, key: string) => String(form.get(key) ?? '');

function purchaseInput(form: FormData, costPaise: number): PurchaseInput {
	return {
		date: field(form, 'date'),
		seller: field(form, 'seller'),
		costPaise,
		reference: field(form, 'reference'),
		paymentAccountId: field(form, 'paymentAccountId'),
		categoryId: field(form, 'categoryId')
	};
}

export const actions: Actions = {
	previewPurchase: async ({ params, request, fetch }) => {
		const form = await request.formData();
		const cost = parseRupees(field(form, 'cost'));
		const input = purchaseInput(form, cost ?? 0);
		if (!cost)
			return fail(400, { purchaseError: 'Enter a positive landed cost.', purchaseDraft: input });
		const result = await apiRequest<Omit<PurchasePreview, 'key'>>(
			fetch,
			apiUrl(),
			`/goals/${encodeURIComponent(params.id)}/purchase/preview`,
			{ method: 'POST', body: JSON.stringify(input) }
		);
		return result.ok
			? { purchasePreview: { ...result.data, key: randomUUID() }, purchaseDraft: input }
			: fail(result.status, { purchaseError: result.error.message, purchaseDraft: input });
	},
	completePurchase: async ({ params, request, fetch }) => {
		const form = await request.formData();
		const input = purchaseInput(form, Number(field(form, 'costPaise')));
		const result = await apiRequest(
			fetch,
			apiUrl(),
			`/goals/${encodeURIComponent(params.id)}/purchase`,
			{
				method: 'POST',
				body: JSON.stringify({
					...input,
					key: field(form, 'key'),
					token: field(form, 'token'),
					confirmed: form.get('confirmed') === 'yes'
				})
			}
		);
		return result.ok
			? { purchaseSaved: true }
			: fail(result.status, { purchaseError: result.error.message, purchaseDraft: input });
	},
	allocateSurplus: async ({ params, request, fetch }) => {
		const form = await request.formData();
		const accounts = form.getAll('accountId');
		const goals = form.getAll('goalId');
		const amounts = form.getAll('amount');
		const lines = accounts.map((account, index) => ({
			accountId: String(account),
			goalId: String(goals[index] ?? ''),
			amount: String(amounts[index] ?? '')
		}));
		const allocationDraft = lines.length <= 10 ? lines : [];
		const result = await apiRequest(
			fetch,
			apiUrl(),
			`/goals/${encodeURIComponent(params.id)}/purchase/allocations`,
			{
				method: 'POST',
				body: JSON.stringify({
					key: field(form, 'key'),
					confirmed: form.get('confirmed') === 'yes',
					lines: lines
						.filter(({ accountId, goalId, amount }) => accountId || goalId || amount.trim())
						.map(({ accountId, goalId, amount }) => ({
							accountId,
							goalId,
							amountPaise: parseRupees(amount) ?? 0
						}))
				})
			}
		);
		return result.ok
			? { allocationSaved: true }
			: fail(result.status, { allocationError: result.error.message, allocationDraft });
	},
	reassess: async ({ params, fetch }) => {
		const result = await apiRequest<Decision>(
			fetch,
			apiUrl(),
			`/goals/${encodeURIComponent(params.id)}/decision`,
			{ method: 'POST' }
		);
		return result.ok
			? { assessmentSaved: true }
			: fail(result.status, { assessmentError: result.error.message });
	},
	updateTargetPrice: async ({ params, request, fetch }) => {
		const goalId = params.id;
		const form = await request.formData();
		const targetPriceRupees = parseFloat(String(form.get('targetPrice') ?? '0'));
		const productId = String(form.get('productId') ?? '');

		if (isNaN(targetPriceRupees) || targetPriceRupees <= 0) {
			return fail(400, { error: 'Please enter a valid target price greater than 0.' });
		}

		const targetPricePaise = Math.round(targetPriceRupees * 100);
		const base = apiUrl();

		// Fetch current goal
		const goalRes = await apiRequest<Goal>(fetch, base, `/goals/${goalId}`);
		if (!goalRes.ok) {
			return fail(goalRes.status, { error: goalRes.error.message });
		}

		const goal = goalRes.data;
		const updateRes = await apiRequest<Goal>(fetch, base, `/goals/${goalId}`, {
			method: 'PUT',
			body: JSON.stringify({
				...goal,
				targetPricePaise
			})
		});

		if (!updateRes.ok) {
			return fail(updateRes.status, { error: updateRes.error.message });
		}

		// Update product target price if linked
		if (productId) {
			const prodRes = await apiRequest<Product>(fetch, base, `/products/${productId}`);
			if (prodRes.ok) {
				await apiRequest(fetch, base, `/products/${productId}`, {
					method: 'PUT',
					body: JSON.stringify({
						...prodRes.data,
						targetPricePaise
					})
				});
			}
		}

		return { success: true, updatedPrice: targetPriceRupees };
	},

	updateProductDetails: async ({ params, request, fetch }) => {
		const goalId = params.id;
		const form = await request.formData();
		const productId = String(form.get('productId') ?? '');
		const name = String(form.get('name') ?? '').trim();
		const brand = String(form.get('brand') ?? '').trim();
		const model = String(form.get('model') ?? '').trim();
		const variant = String(form.get('variant') ?? '').trim();
		const retailer = String(form.get('retailer') ?? '').trim();
		const url = String(form.get('url') ?? '').trim();
		const imageUrl = String(form.get('imageUrl') ?? '').trim();
		const notes = String(form.get('notes') ?? '').trim();
		const availability = String(form.get('availability') ?? 'in_stock');
		const currency = String(form.get('currency') ?? 'INR');
		const specsRaw = String(form.get('specifications') ?? '{}');

		const currentPriceRupees = parseFloat(String(form.get('currentPrice') ?? '0'));
		const originalPriceRupees = parseFloat(String(form.get('originalPrice') ?? '0'));
		const targetPriceRupees = parseFloat(String(form.get('targetPrice') ?? '0'));

		const currentPricePaise =
			isNaN(currentPriceRupees) || currentPriceRupees < 0
				? 0
				: Math.round(currentPriceRupees * 100);
		const originalPricePaise =
			isNaN(originalPriceRupees) || originalPriceRupees < 0
				? 0
				: Math.round(originalPriceRupees * 100);
		const targetPricePaise =
			isNaN(targetPriceRupees) || targetPriceRupees < 0 ? 0 : Math.round(targetPriceRupees * 100);

		let specifications: Record<string, string>;
		try {
			specifications = JSON.parse(specsRaw);
		} catch {
			specifications = {};
		}

		if (!name) {
			return fail(400, { error: 'Product name is required.' });
		}

		const base = apiUrl();
		let targetProdId = productId;

		if (!targetProdId) {
			// Create product if not linked yet
			const createRes = await apiRequest<Product>(fetch, base, '/products', {
				method: 'POST',
				body: JSON.stringify({
					name,
					brand,
					model,
					variant,
					category: 'Shopping',
					currentPricePaise,
					originalPricePaise,
					targetPricePaise,
					currency,
					availability,
					specifications,
					url,
					imageUrl,
					retailer,
					notes,
					source: 'manual',
					verificationStatus: 'manual_entry'
				})
			});
			if (!createRes.ok) {
				return fail(createRes.status, { error: createRes.error.message });
			}
			targetProdId = createRes.data.id;

			// Link to goal
			const goalRes = await apiRequest<Goal>(fetch, base, `/goals/${goalId}`);
			if (goalRes.ok) {
				await apiRequest(fetch, base, `/goals/${goalId}`, {
					method: 'PUT',
					body: JSON.stringify({
						...goalRes.data,
						productId: targetProdId,
						targetPricePaise: targetPricePaise || currentPricePaise || goalRes.data.targetPricePaise
					})
				});
			}
		} else {
			// Update existing product
			const updateRes = await apiRequest<Product>(fetch, base, `/products/${targetProdId}`, {
				method: 'PUT',
				body: JSON.stringify({
					id: targetProdId,
					name,
					brand,
					model,
					variant,
					category: 'Shopping',
					currentPricePaise,
					originalPricePaise,
					targetPricePaise,
					currency,
					availability,
					specifications,
					url,
					imageUrl,
					retailer,
					notes,
					source: 'manual',
					verificationStatus: 'manual_entry'
				})
			});
			if (!updateRes.ok) {
				return fail(updateRes.status, { error: updateRes.error.message });
			}
		}

		return { success: true, updatedProductId: targetProdId };
	},

	addManualPrice: async ({ request, fetch }) => {
		const form = await request.formData();
		const productId = String(form.get('productId') ?? '');
		const priceRupees = parseFloat(String(form.get('price') ?? '0'));
		const observedOn = String(form.get('observedOn') ?? '').trim();
		const retailer = String(form.get('retailer') ?? '').trim();
		const priceType = String(form.get('priceType') ?? 'regular');
		const notes = String(form.get('notes') ?? '').trim();

		if (!productId) {
			return fail(400, { error: 'Linked product is required.' });
		}
		if (isNaN(priceRupees) || priceRupees <= 0) {
			return fail(400, { error: 'Please enter a valid price greater than zero.' });
		}
		if (!observedOn) {
			return fail(400, { error: 'Observation date is required.' });
		}

		const pricePaise = Math.round(priceRupees * 100);
		const base = apiUrl();

		const res = await apiRequest(fetch, base, `/products/${productId}/prices`, {
			method: 'POST',
			body: JSON.stringify({
				pricePaise,
				observedOn,
				retailer: retailer || 'Manual Entry',
				priceType,
				source: 'manual',
				provenance: 'manual',
				notes
			})
		});

		if (!res.ok) {
			return fail(res.status, { error: res.error.message });
		}

		return { success: true };
	},

	uploadDocument: async ({ params, request, fetch }) => {
		const goalId = params.id;
		const form = await request.formData();
		const file = form.get('docFile') as File | null;

		if (!file || file.size === 0) {
			return fail(400, { error: 'Please select a document file to upload.' });
		}

		// Enforce size limit (10MB max)
		if (file.size > 10 * 1024 * 1024) {
			return fail(400, { error: 'File size exceeds maximum supported limit of 10MB.' });
		}

		const base = apiUrl();
		const goalRes = await apiRequest<Goal>(fetch, base, `/goals/${goalId}`);
		const goalName = goalRes.ok ? goalRes.data.name : 'Purchase Goal';

		const arrayBuffer = await file.arrayBuffer();
		const buffer = Buffer.from(arrayBuffer);

		try {
			const preview = await processDocumentUpload(
				buffer,
				file.name,
				file.type,
				goalId,
				goalName,
				fetch
			);

			if (preview.validRows.length === 0) {
				const errMsg =
					preview.errorRows.length > 0
						? `No valid price observations found (${preview.errorRows.length} rows failed date or price validation).`
						: 'No price observation records could be found in this document.';
				return fail(400, { error: errMsg, preview });
			}

			// Record upload on Go backend
			await apiRequest(fetch, base, `/goals/${goalId}/documents`, {
				method: 'POST',
				body: JSON.stringify({
					filename: preview.filename,
					fileType: preview.fileType,
					fileSize: preview.fileSize,
					purpose: preview.purpose,
					status: 'previewed',
					parsedData: {
						validRows: preview.validRows,
						errorRows: preview.errorRows,
						detectedColumns: preview.detectedColumns,
						columnMappings: preview.columnMappings
					},
					summary: preview.summary
				})
			});

			return {
				success: true,
				preview
			};
		} catch (err: unknown) {
			const msg = err instanceof Error ? err.message : 'Failed to parse document.';
			return fail(500, { error: msg });
		}
	},

	confirmDocumentImport: async ({ params, request, fetch }) => {
		const goalId = params.id;
		const form = await request.formData();
		const productId = String(form.get('productId') ?? '');
		const rowsJson = String(form.get('rowsJson') ?? '[]');
		const filename = String(form.get('filename') ?? 'imported_document');

		let rows: Array<{
			observedOn: string;
			pricePaise: number;
			source?: string;
			retailer?: string;
			notes?: string;
		}>;
		try {
			rows = JSON.parse(rowsJson);
		} catch {
			return fail(400, { error: 'Invalid rows payload.' });
		}

		if (rows.length === 0) {
			return fail(400, { error: 'No observations to import.' });
		}

		const base = apiUrl();

		let targetProdId = productId;
		if (!targetProdId) {
			// Create a tracked product if not already attached
			const prodRes = await apiRequest<Product>(fetch, base, '/products', {
				method: 'POST',
				body: JSON.stringify({
					name: filename.replace(/\.[^/.]+$/, ''),
					brand: '',
					model: '',
					category: 'Shopping',
					currentPricePaise: rows[rows.length - 1].pricePaise,
					targetPricePaise: rows[rows.length - 1].pricePaise,
					url: '',
					imageUrl: '',
					retailer: rows[0].retailer || 'Document Import',
					source: 'document_upload',
					verificationStatus: 'verified'
				})
			});
			if (!prodRes.ok) {
				return fail(prodRes.status, { error: prodRes.error.message });
			}
			targetProdId = prodRes.data.id;

			// Link to goal
			const goalRes = await apiRequest<Goal>(fetch, base, `/goals/${goalId}`);
			if (goalRes.ok) {
				await apiRequest(fetch, base, `/goals/${goalId}`, {
					method: 'PUT',
					body: JSON.stringify({
						...goalRes.data,
						productId: targetProdId,
						targetPricePaise: rows[rows.length - 1].pricePaise
					})
				});
			}
		}

		// Import batch observations
		const importRes = await apiRequest(fetch, base, `/products/${targetProdId}/prices/import`, {
			method: 'POST',
			body: JSON.stringify({
				filename,
				observations: rows.map((r) => ({
					pricePaise: r.pricePaise,
					observedOn: r.observedOn,
					source: r.source || r.retailer || 'document_upload'
				}))
			})
		});

		if (!importRes.ok) {
			return fail(importRes.status, { error: importRes.error.message });
		}

		return { success: true, count: rows.length };
	},

	linkProduct: async ({ params, request, fetch }) => {
		const goalId = params.id;
		const form = await request.formData();
		const productUrl = String(form.get('productUrl') ?? '').trim();
		const productName = String(form.get('productName') ?? '').trim();

		const variant = String(form.get('variant') ?? '').trim();
		if (form.get('variantConfirmed') !== 'yes')
			return fail(400, { error: 'Confirm the listing matches the product variant.' });
		if (!productUrl) {
			return fail(400, { error: 'Product URL is required.' });
		}

		const base = apiUrl();
		if (!productName) return fail(400, { error: 'Enter the product’s exact name or variant.' });
		const listing = await apiRequest<{ url: string; retailer: string }>(
			fetch,
			base,
			'/products/validate-listing',
			{
				method: 'POST',
				body: JSON.stringify({ url: productUrl })
			}
		);
		if (!listing.ok) return fail(listing.status, { error: listing.error.message });
		const prodRes = await apiRequest<Product>(fetch, base, '/products/tracked', {
			method: 'POST',
			body: JSON.stringify({
				name: productName,
				category: 'Shopping',
				variant,
				offer: {
					variant,
					listingUrl: listing.data.url,
					retailer: listing.data.retailer,
					variantConfirmed: true
				}
			})
		});

		if (!prodRes.ok) {
			return fail(prodRes.status, { error: prodRes.error.message });
		}

		// Update goal with new productId
		const goalRes = await apiRequest<Goal>(fetch, base, `/goals/${goalId}`);
		if (!goalRes.ok) return fail(goalRes.status, { error: goalRes.error.message });
		const updated = await apiRequest(fetch, base, `/goals/${goalId}`, {
			method: 'PUT',
			body: JSON.stringify({ ...goalRes.data, productId: prodRes.data.id })
		});
		if (!updated.ok) return fail(updated.status, { error: updated.error.message });

		return { success: true };
	}
};
