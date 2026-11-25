import { randomUUID } from 'node:crypto';
import { error, fail } from '@sveltejs/kit';
import { apiRequest, type Goal } from '#lib/server/api.js';
import { apiUrl } from '#lib/server/config.js';
import { parseRupees } from '#lib/money.js';
import type { Actions, PageServerLoad } from './$types';

type Terms = { revision: number; mode: string; frequency: string; approvedPaise: number };
type Plan = {
	goal: Goal & { legacySavedPaise?: number };
	terms: Terms;
	capacityPaise: number;
	profileReady: boolean;
	fundedPaise: number;
	backingAtRisk: boolean;
	projection: {
		remainingPaise: number;
		monthlyRequestedPaise: number;
		monthlyAllocatedPaise: number;
		recommendedPaise: number;
		estimatedDate: string | null;
		state: string;
		overCapacity: boolean;
	};
	accounts: { id: string; name: string; purpose: string; availablePaise: number }[];
	contributions: { id: string; accountId: string; amountPaise: number; createdAt: string }[];
	history: Terms[];
};

const path = (id: string) => '/goals/' + encodeURIComponent(id);
const field = (form: FormData, name: string) => String(form.get(name) ?? '');

export const load: PageServerLoad = async ({ params, fetch }) => {
	const result = await apiRequest<Plan>(fetch, apiUrl(), path(params.id) + '/plan');
	if (!result.ok) error(result.status, result.error.message);
	return { plan: result.data, id: params.id, fundingKey: randomUUID() };
};

export const actions = {
	terms: async ({ params, request, fetch }) => {
		const form = await request.formData();
		const approvedPaise = parseRupees(field(form, 'approved'));
		if (approvedPaise === null) return fail(400, { error: 'Enter a valid contribution amount.' });
		const preview = field(form, 'intent') === 'preview';
		const result = await apiRequest<Plan['projection']>(
			fetch,
			apiUrl(),
			path(params.id) + (preview ? '/plan/preview' : '/plan'),
			{
				method: preview ? 'POST' : 'PUT',
				body: JSON.stringify({
					revision: Number(field(form, 'revision')),
					mode: field(form, 'mode'),
					frequency: field(form, 'frequency'),
					approvedPaise
				})
			}
		);
		if (!result.ok) return fail(result.status, { error: result.error.message });
		return preview ? { preview: result.data } : { success: 'Plan approved.' };
	},
	edit: async ({ params, request, fetch }) => {
		const form = await request.formData();
		const targetPricePaise = parseRupees(field(form, 'target'));
		if (!targetPricePaise) return fail(400, { error: 'Enter a positive target price.' });
		const item = await apiRequest<Goal>(fetch, apiUrl(), path(params.id));
		if (!item.ok) return fail(item.status, { error: item.error.message });
		const result = await apiRequest(fetch, apiUrl(), path(params.id), {
			method: 'PUT',
			body: JSON.stringify({
				name: field(form, 'name'),
				targetPricePaise,
				targetDate: field(form, 'targetDate') || null,
				priority: field(form, 'priority'),
				productId: item.data.productId ?? null
			})
		});
		return result.ok
			? { success: 'Goal updated.' }
			: fail(result.status, { error: result.error.message });
	},
	fund: async ({ params, request, fetch }) => {
		const form = await request.formData();
		const fundingKey = field(form, 'fundingKey');
		const amountPaise = parseRupees(field(form, 'amount'));
		if (!amountPaise) return fail(400, { error: 'Enter a positive amount.', fundingKey });
		const result = await apiRequest(fetch, apiUrl(), path(params.id) + '/contributions', {
			method: 'POST',
			body: JSON.stringify({
				accountId: field(form, 'accountId'),
				amountPaise,
				idempotencyKey: fundingKey
			})
		});
		return result.ok
			? { success: 'Savings earmarked.' }
			: fail(result.status, { error: result.error.message, fundingKey });
	},
	lifecycle: async ({ params, request, fetch }) => {
		const form = await request.formData();
		const status = field(form, 'status');
		if (status === 'cancelled' && field(form, 'confirm') !== 'yes') {
			return fail(400, {
				error: 'Confirm that you want to cancel and release the goal’s allocations.'
			});
		}
		const result = await apiRequest(fetch, apiUrl(), path(params.id) + '/lifecycle', {
			method: 'PUT',
			body: JSON.stringify({ status })
		});
		return result.ok
			? { success: 'Goal status updated.' }
			: fail(result.status, { error: result.error.message });
	}
} satisfies Actions;
