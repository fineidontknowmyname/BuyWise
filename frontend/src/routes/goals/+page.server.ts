import { fail, redirect } from '@sveltejs/kit';
import { resolve } from '$app/paths';
import { apiRequest, type Goal } from '#lib/server/api.js';
import { apiUrl } from '#lib/server/config.js';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ fetch }) => {
	const result = await apiRequest<Goal[]>(fetch, apiUrl(), '/goals');
	if (!result.ok) {
		return { goals: [], error: result.error.message };
	}
	return { goals: result.data, error: null };
};

export const actions: Actions = {
	create: async ({ request, fetch }) => {
		const form = await request.formData();
		const name = String(form.get('name') ?? '');
		const result = await apiRequest<Goal>(fetch, apiUrl(), '/goals', {
			method: 'POST',
			body: JSON.stringify({ name })
		});
		if (!result.ok) {
			return fail(result.status, { name, error: result.error.message });
		}
		redirect(303, resolve('/goals/[id]', { id: result.data.id }));
	}
};
