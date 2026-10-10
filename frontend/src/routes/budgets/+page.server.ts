import { fail, redirect } from '@sveltejs/kit';
import { resolve } from '$app/paths';
import { apiRequest, type BudgetReport, type Category } from '#lib/server/api.js';
import { apiUrl } from '#lib/server/config.js';
import { parseRupees } from '#lib/money.js';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ fetch }) => {
	const [report, categories] = await Promise.all([
		apiRequest<BudgetReport>(fetch, apiUrl(), '/budgets'),
		apiRequest<Category[]>(fetch, apiUrl(), '/categories')
	]);
	if (!report.ok) return { report: null, categories: [], error: report.error.message };
	if (!categories.ok) return { report: null, categories: [], error: categories.error.message };
	return {
		report: report.data,
		categories: categories.data.filter((c) => c.kind === 'expense' && !c.committed),
		error: null
	};
};

export const actions: Actions = {
	set: async ({ request, fetch }) => {
		const form = await request.formData();
		const categoryId = String(form.get('categoryId') ?? '');
		const limit = String(form.get('limit') ?? '');
		const limitPaise = parseRupees(limit);
		if (limitPaise === null) {
			return fail(400, {
				categoryId,
				error: 'Enter the monthly limit as rupees, like 8000 or 8000.50.'
			});
		}
		const result = await apiRequest(fetch, apiUrl(), `/budgets/${encodeURIComponent(categoryId)}`, {
			method: 'PUT',
			body: JSON.stringify({ limitPaise })
		});
		if (!result.ok) {
			return fail(result.status, { categoryId, error: result.error.message });
		}
		redirect(303, resolve('/budgets'));
	}
};
