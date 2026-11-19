import { error, fail } from '@sveltejs/kit';
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
import { extractProductFromUrl } from '#lib/server/products/engine.js';
import { processDocumentUpload } from '#lib/server/documents/service.js';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ fetch, params }) => {
	const base = apiUrl();
	const goalId = params.id;

	const [decisionRes, goalRes] = await Promise.all([
		apiRequest<Decision>(fetch, base, `/goals/${encodeURIComponent(goalId)}/decision`),
		apiRequest<Goal>(fetch, base, `/goals/${encodeURIComponent(goalId)}`)
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
		decision: decisionRes.data,
		goal,
		product,
		prices,
		stats,
		documents
	};
};

export const actions: Actions = {
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

	refreshPrice: async ({ request, fetch }) => {
		const form = await request.formData();
		const productId = String(form.get('productId') ?? '');
		const productUrl = String(form.get('productUrl') ?? '');

		if (!productId || !productUrl) {
			return fail(400, { error: 'Product URL is required for live price refresh.' });
		}

		const base = apiUrl();
		const prodRes = await apiRequest<Product>(fetch, base, `/products/${productId}`);
		if (!prodRes.ok) {
			return fail(prodRes.status, { error: 'Could not load linked product.' });
		}

		const existing = prodRes.data;

		// Extract product from URL using hybrid engine
		const extracted = await extractProductFromUrl(productUrl, fetch);

		if (extracted.confidence === 'manual_fallback' || extracted.error) {
			// CRITICAL: A failed refresh MUST NOT overwrite the last valid price with zero or empty
			return fail(422, {
				error: `Retailer refresh failed: ${extracted.error || 'Page could not be verified'}. Retained previous valid price.`
			});
		}

		if (extracted.currentPricePaise > 0) {
			// Update product current price & verified fields
			await apiRequest(fetch, base, `/products/${productId}`, {
				method: 'PUT',
				body: JSON.stringify({
					...existing,
					currentPricePaise: extracted.currentPricePaise,
					originalPricePaise: extracted.originalPricePaise || existing.originalPricePaise || 0,
					discountPercent: extracted.discountPercent || existing.discountPercent || 0,
					name: extracted.name || existing.name,
					brand: extracted.brand || existing.brand,
					model: extracted.model || existing.model,
					variant: extracted.variant || existing.variant,
					retailer: extracted.retailer || existing.retailer,
					specifications: { ...existing.specifications, ...extracted.specifications },
					lastRetrievedAt: new Date().toISOString(),
					verificationStatus: 'verified'
				})
			});

			// Record new price observation
			const today = new Date().toISOString().split('T')[0];
			await apiRequest(fetch, base, `/products/${productId}/prices`, {
				method: 'POST',
				body: JSON.stringify({
					pricePaise: extracted.currentPricePaise,
					observedOn: today,
					retailer: extracted.retailer || existing.retailer,
					source: extracted.retailer || 'live_refresh',
					sourceRef: productUrl,
					provenance: 'url_extraction'
				})
			});

			return { success: true, refreshedPrice: extracted.currentPricePaise / 100 };
		}

		return fail(422, {
			error:
				'Product page was accessed, but no verified price was found. Previous valid price was preserved.'
		});
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

		if (!productUrl) {
			return fail(400, { error: 'Product URL is required.' });
		}

		const base = apiUrl();
		const extracted = await extractProductFromUrl(productUrl, fetch);

		const name = productName || extracted.name || 'Tracked Product';
		const currentPricePaise = extracted.currentPricePaise || 0;

		const prodRes = await apiRequest<Product>(fetch, base, '/products', {
			method: 'POST',
			body: JSON.stringify({
				name,
				brand: extracted.brand || '',
				model: extracted.model || '',
				variant: extracted.variant || '',
				category: 'Shopping',
				currentPricePaise,
				originalPricePaise: extracted.originalPricePaise || 0,
				targetPricePaise: currentPricePaise,
				specifications: extracted.specifications || {},
				url: productUrl,
				imageUrl: extracted.imageUrl || '',
				retailer: extracted.retailer || 'Online Store',
				notes: '',
				source: 'url_extraction',
				verificationStatus: extracted.confidence === 'high' ? 'verified' : 'unverified'
			})
		});

		if (!prodRes.ok) {
			return fail(prodRes.status, { error: prodRes.error.message });
		}

		// Update goal with new productId
		const goalRes = await apiRequest<Goal>(fetch, base, `/goals/${goalId}`);
		if (goalRes.ok) {
			await apiRequest(fetch, base, `/goals/${goalId}`, {
				method: 'PUT',
				body: JSON.stringify({
					...goalRes.data,
					productId: prodRes.data.id,
					targetPricePaise: goalRes.data.targetPricePaise || currentPricePaise
				})
			});
		}

		return { success: true };
	}
};
