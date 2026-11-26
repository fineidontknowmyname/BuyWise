import { error, fail, redirect } from '@sveltejs/kit';
import { resolve } from '$app/paths';
import { apiRequest, type Product, type Goal } from '#lib/server/api.js';
import { apiUrl } from '#lib/server/config.js';
import { marketDate } from '#lib/dates.js';
import { parseRupees } from '#lib/money.js';
import type { Actions, PageServerLoad } from './$types';

type Offer = {
	id: string;
	productId: string;
	retailer: string;
	seller: string;
	listingUrl: string;
};
type Thresholds = {
	idealPaise: number | null;
	goodPaise: number | null;
	maximumPaise: number | null;
};
type OfferRecord = {
	id: string;
	observedOn: string;
	observedAt: string | null;
	recordedAt: string;
	fetchedAt: string | null;
	source: string;
	sourceLabel: string;
	availability: string;
	itemPaise: number;
	landedPaise: number | null;
	taxPaise: number | null;
	deliveryPaise: number | null;
	discountPaise: number | null;
};
type History = {
	refreshAvailable: boolean;
	refresh: {
		id: string;
		state: string;
		errorCode: string;
		attempts: number;
		nextRefreshAt: string | null;
	} | null;
	offer: Offer;
	observations: OfferRecord[];
	freshness: 'missing' | 'stale' | 'recent';
	stats: {
		count: number;
		minimumPaise: number | null;
		maximumPaise: number | null;
		averagePaise: number | null;
	};
};

const path = (id: string) => '/products/' + encodeURIComponent(id);
const idValid = (id: string) => /^[0-9a-f]{8}(?:-[0-9a-f]{4}){3}-[0-9a-f]{12}$/i.test(id);
const field = (form: FormData, name: string) => String(form.get(name) ?? '').trim();
const amount = (form: FormData, name: string) => {
	const value = field(form, name);
	return value ? (parseRupees(value) ?? -1) : null;
};

export const load: PageServerLoad = async ({ params, fetch, url }) => {
	const base = path(params.id) + '/offers';
	const result = await apiRequest<{
		product: Product;
		offers: Offer[];
		thresholds: Thresholds;
		refreshAvailable: boolean;
	}>(fetch, apiUrl(), base);
	if (!result.ok) error(result.status, result.error.message);
	const selected = url.searchParams.get('offer') ?? result.data.offers[0]?.id;
	if (selected && !result.data.offers.some((offer) => offer.id === selected)) {
		error(404, 'This offer does not belong to the product.');
	}
	const history = selected
		? await apiRequest<History>(fetch, apiUrl(), base + '/' + selected + '/history')
		: null;
	if (history && !history.ok) error(history.status, history.error.message);
	const goals = await apiRequest<Goal[]>(fetch, apiUrl(), '/goals');
	return {
		...result.data,
		history: history?.data ?? null,
		today: marketDate(),
		goals: goals.ok
			? goals.data.filter(
					(goal) => !goal.productId && (goal.status === 'active' || goal.status === 'paused')
				)
			: []
	};
};

export const actions = {
	offer: async ({ params, request, fetch }) => {
		const form = await request.formData();
		const result = await apiRequest<Offer>(fetch, apiUrl(), path(params.id) + '/offers', {
			method: 'POST',
			body: JSON.stringify({
				retailer: field(form, 'retailer'),
				seller: field(form, 'seller'),
				listingUrl: field(form, 'url'),
				variant: field(form, 'variant'),
				variantConfirmed: field(form, 'confirmed') === 'yes'
			})
		});
		if (!result.ok) return fail(result.status, { error: result.error.message });
		redirect(303, resolve('/products/[id]/offers', { id: params.id }) + '?offer=' + result.data.id);
	},
	goal: async ({ params, request, fetch }) => {
		const id = field(await request.formData(), 'goalId');
		if (!idValid(id)) return fail(400, { error: 'Select a purchase goal.' });
		const selected = await apiRequest<Goal>(fetch, apiUrl(), '/goals/' + id);
		if (!selected.ok) return fail(selected.status, { error: selected.error.message });
		if (selected.data.productId && selected.data.productId !== params.id) {
			return fail(409, { error: 'This goal is already linked to a different product.' });
		}
		const result = await apiRequest(fetch, apiUrl(), '/goals/' + id, {
			method: 'PUT',
			body: JSON.stringify({ ...selected.data, productId: params.id })
		});
		return result.ok
			? { notice: 'Product linked to the goal.' }
			: fail(result.status, { error: result.error.message });
	},
	thresholds: async ({ params, request, fetch }) => {
		const form = await request.formData();
		const values = {
			idealPaise: amount(form, 'ideal'),
			goodPaise: amount(form, 'good'),
			maximumPaise: amount(form, 'maximum')
		};
		const entered = Object.values(values).filter((value) => value !== null);
		if (entered.some((value) => value <= 0)) {
			return fail(400, {
				thresholdError:
					'Enter positive prices with up to two decimal places, or leave a field blank.'
			});
		}
		if (entered.some((value, index) => index > 0 && value < entered[index - 1])) {
			return fail(400, {
				thresholdError:
					'Ideal must be no higher than Good, and Good no higher than Maximum. Blank targets are optional.'
			});
		}
		const result = await apiRequest(fetch, apiUrl(), path(params.id) + '/thresholds', {
			method: 'PUT',
			body: JSON.stringify(values)
		});
		return result.ok
			? { notice: 'Price thresholds saved.' }
			: fail(result.status, { thresholdError: result.error.message });
	},
	record: async ({ params, request, fetch }) => {
		const form = await request.formData();
		const id = field(form, 'offerId');
		if (!idValid(id)) return fail(400, { error: 'Select a retailer offer.' });
		const observedOn = field(form, 'date');
		const time = field(form, 'time');
		const result = await apiRequest<{ imported: number }>(
			fetch,
			apiUrl(),
			path(params.id) + '/offers/' + id + '/observations',
			{
				method: 'POST',
				body: JSON.stringify({
					observedOn,
					observedAt: time ? observedOn + 'T' + time + ':00+05:30' : null,
					sourceLabel: field(form, 'source'),
					availability: field(form, 'availability'),
					itemPaise: amount(form, 'item') ?? -1,
					taxPaise: amount(form, 'tax'),
					deliveryPaise: amount(form, 'delivery'),
					discountPaise: amount(form, 'discount')
				})
			}
		);
		return result.ok
			? {
					notice: result.data.imported
						? 'Price recorded.'
						: 'This observation was already recorded.'
				}
			: fail(result.status, { error: result.error.message });
	},
	import: async ({ params, request, fetch }) => {
		const form = await request.formData();
		const id = field(form, 'offerId');
		if (!idValid(id)) return fail(400, { error: 'Select a retailer offer.' });
		const upload = form.get('csv');
		if (typeof upload === 'string' || !upload || upload.size < 1 || upload.size > 200000) {
			return fail(400, { error: 'Choose a CSV file up to 200 KB.' });
		}
		const result = await apiRequest<{ imported: number; duplicates: number }>(
			fetch,
			apiUrl(),
			path(params.id) + '/offers/' + id + '/import',
			{
				method: 'POST',
				body: JSON.stringify({ csv: await upload.text() })
			}
		);
		return result.ok
			? {
					notice: `${result.data.imported} recorded. ${result.data.duplicates} duplicate rows skipped.`
				}
			: fail(result.status, { error: result.error.message });
	},
	refresh: async ({ params, request, fetch }) => {
		const id = field(await request.formData(), 'offerId');
		if (!idValid(id)) return fail(400, { error: 'Select a retailer offer.' });
		const result = await apiRequest(
			fetch,
			apiUrl(),
			path(params.id) + '/offers/' + id + '/refresh',
			{ method: 'POST' }
		);
		return result.ok
			? { refreshRequested: true }
			: fail(result.status, { error: result.error.message });
	},
	details: async ({ params, request, fetch }) => {
		const form = await request.formData();
		const id = field(form, 'offerId');
		if (!idValid(id)) return fail(400, { error: 'Select a retailer offer.' });
		const result = await apiRequest<Product>(
			fetch,
			apiUrl(),
			path(params.id) + '/offers/' + id + '/details',
			{
				method: 'POST',
				body: JSON.stringify({ confirmed: field(form, 'confirmed') === 'yes' }),
				signal: AbortSignal.timeout(25000)
			}
		);
		return result.ok
			? {
					notice:
						'Available product details updated. Your saved identity and price targets were preserved.'
				}
			: fail(result.status, { error: result.error.message });
	}
} satisfies Actions;
