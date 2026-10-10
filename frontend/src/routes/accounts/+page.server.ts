import { fail, redirect } from '@sveltejs/kit';
import { resolve } from '$app/paths';
import { apiRequest, type Account } from '#lib/server/api.js';
import { apiUrl } from '#lib/server/config.js';
import { parseSignedRupees } from '#lib/money.js';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ fetch }) => {
	const result = await apiRequest<Account[]>(fetch, apiUrl(), '/accounts');
	if (!result.ok) {
		return { accounts: [], error: result.error.message };
	}
	return { accounts: result.data, error: null };
};

export const actions: Actions = {
	create: async ({ request, fetch }) => {
		const form = await request.formData();
		const name = String(form.get('name') ?? '');
		const opening = String(form.get('opening') ?? '');
		const openingBalancePaise = opening.trim() === '' ? 0 : parseSignedRupees(opening);
		if (openingBalancePaise === null) {
			return fail(400, {
				name,
				opening,
				error: 'Enter the opening balance as rupees, like 2500 or 2500.50.'
			});
		}
		const result = await apiRequest<Account>(fetch, apiUrl(), '/accounts', {
			method: 'POST',
			body: JSON.stringify({ name, openingBalancePaise })
		});
		if (!result.ok) {
			return fail(result.status, { name, opening, error: result.error.message });
		}
		redirect(303, resolve('/accounts/[id]', { id: result.data.id }));
	}
};
