import { fail } from '@sveltejs/kit';
import { apiRequest, type Category } from '#lib/server/api.js';
import { apiUrl } from '#lib/server/config.js';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ fetch }) => {
	const result = await apiRequest<Category[]>(fetch, apiUrl(), '/categories');
	if (!result.ok) {
		return { categories: [], error: result.error.message };
	}
	return { categories: result.data, error: null };
};

export const actions: Actions = {
	create: async ({ request, fetch }) => {
		const form = await request.formData();
		const name = String(form.get('name') ?? '');
		const kind = String(form.get('kind') ?? '');
		const committed = form.get('committed') === 'on';
		const result = await apiRequest<Category>(fetch, apiUrl(), '/categories', {
			method: 'POST',
			body: JSON.stringify({ name, kind, committed })
		});
		if (!result.ok) {
			return fail(result.status, { name, kind, committed, error: result.error.message });
		}
		return { created: result.data.name };
	}
};
