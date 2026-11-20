import { randomUUID } from 'node:crypto';
import { fail } from '@sveltejs/kit';
import { apiRequest, type Account } from '#lib/server/api.js';
import { apiUrl } from '#lib/server/config.js';
import { parseRupees } from '#lib/money.js';
import type { Actions, PageServerLoad } from './$types';

type ProfileItem = {
	name: string;
	kind: string;
	frequency: string;
	amountPaise: number;
};

type Foundation = {
	profile: {
		revision: number;
		debtStatus: string;
		items: ProfileItem[];
		variableIncomePercent: number;
		bufferPercent: number;
		bufferMinimumPaise: number;
		emergencyTargetPaise: number;
	};
	capacity: {
		version: string;
		fixedIncomePaise: number;
		variableIncomePaise: number;
		countedVariablePaise: number;
		committedPaise: number;
		discretionaryPaise: number;
		bufferPaise: number;
		safeSavingsPaise: number;
	};
	accounts: (Account & { purpose: string })[];
	emergencySavingsPaise: number;
	generalSavingsPaise: number;
	emergencyGapPaise: number;
};

type Transfer = {
	id: string;
	fromAccountId: string;
	toAccountId: string;
	amountPaise: number;
	occurredOn: string;
	note: string;
	reversesId: string | null;
	reversed: boolean;
};

export const load: PageServerLoad = async ({ fetch }) => {
	const [profile, transfers] = await Promise.all([
		apiRequest<Foundation>(fetch, apiUrl(), '/financial-profile'),
		apiRequest<Transfer[]>(fetch, apiUrl(), '/transfers')
	]);
	return {
		foundation: profile.ok ? profile.data : null,
		transfers: transfers.ok ? transfers.data : [],
		error: !profile.ok ? profile.error.message : !transfers.ok ? transfers.error.message : null,
		idempotencyKey: randomUUID(),
		today: new Intl.DateTimeFormat('en-CA', {
			timeZone: 'Asia/Kolkata',
			dateStyle: 'short'
		}).format(new Date())
	};
};

const field = (form: FormData, name: string) => String(form.get(name) ?? '');
const amount = (form: FormData, name: string) => parseRupees(field(form, name));

export const actions: Actions = {
	profile: async ({ request, fetch }) => {
		const form = await request.formData();
		const names = form.getAll('name');
		const kinds = form.getAll('kind');
		const frequencies = form.getAll('frequency');
		const amounts = form.getAll('amount');
		const items = names.map((name, i) => ({
			name: String(name),
			kind: String(kinds[i] ?? ''),
			frequency: String(frequencies[i] ?? ''),
			amountPaise: parseRupees(String(amounts[i] ?? ''))
		}));
		const bufferMinimumPaise = amount(form, 'bufferMinimum');
		const emergencyTargetPaise = amount(form, 'emergencyTarget');
		if (
			bufferMinimumPaise === null ||
			emergencyTargetPaise === null ||
			items.some((item) => !item.amountPaise)
		) {
			return fail(400, { error: 'Enter valid rupee amounts for the profile.' });
		}
		const result = await apiRequest(fetch, apiUrl(), '/financial-profile', {
			method: 'PUT',
			body: JSON.stringify({
				revision: Number(field(form, 'revision')),
				debtStatus: field(form, 'debtStatus'),
				items,
				variableIncomePercent: Number(field(form, 'variableIncomePercent')),
				bufferPercent: Number(field(form, 'bufferPercent')),
				bufferMinimumPaise,
				emergencyTargetPaise
			})
		});
		return result.ok
			? { success: 'Financial profile saved.' }
			: fail(result.status, { error: result.error.message });
	},
	purpose: async ({ request, fetch }) => {
		const form = await request.formData();
		const id = encodeURIComponent(field(form, 'accountId'));
		const result = await apiRequest(fetch, apiUrl(), `/accounts/${id}/purpose`, {
			method: 'PUT',
			body: JSON.stringify({ purpose: field(form, 'purpose') })
		});
		return result.ok
			? { success: 'Account purpose saved.' }
			: fail(result.status, { error: result.error.message });
	},
	transfer: async ({ request, fetch }) => {
		const form = await request.formData();
		const idempotencyKey = field(form, 'idempotencyKey');
		const result = await apiRequest(fetch, apiUrl(), '/transfers', {
			method: 'POST',
			body: JSON.stringify({
				fromAccountId: field(form, 'fromAccountId'),
				toAccountId: field(form, 'toAccountId'),
				amountPaise: amount(form, 'amount'),
				occurredOn: field(form, 'occurredOn'),
				note: field(form, 'note'),
				idempotencyKey
			})
		});
		return result.ok
			? { success: 'Transfer recorded.' }
			: fail(result.status, { error: result.error.message, idempotencyKey });
	},
	reverse: async ({ request, fetch }) => {
		const form = await request.formData();
		const id = encodeURIComponent(field(form, 'transferId'));
		const result = await apiRequest(fetch, apiUrl(), `/transfers/${id}/reverse`, {
			method: 'POST'
		});
		return result.ok
			? { success: 'Reversal recorded.' }
			: fail(result.status, { error: result.error.message });
	}
};
