import { fail, redirect } from '@sveltejs/kit';
import { resolve } from '$app/paths';
import { apiRequest, type Account } from '#lib/server/api.js';
import { apiUrl } from '#lib/server/config.js';
import { parseSignedRupees, parseRupees } from '#lib/money.js';
import type { MoneyPlan } from '#lib/moneyPlan.js';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ fetch }) => {
	const [result, money] = await Promise.all([
		apiRequest<Account[]>(fetch, apiUrl(), '/accounts'),
		apiRequest<MoneyPlan>(fetch, apiUrl(), '/money-plan')
	]);
	if (!result.ok) {
		return { accounts: [], money: money.ok ? money.data : null, error: result.error.message };
	}
	return {
		accounts: result.data,
		money: money.ok ? money.data : null,
		error: money.ok ? null : money.error.message
	};
};

export const actions: Actions = {
	plan: async ({ request, fetch }) => {
		const form = await request.formData();
		const incomePaise = parseRupees(String(form.get('income') ?? ''));
		const categories = [];
		for (const [name, value] of form.entries()) {
			if (!name.startsWith('category:')) continue;
			const amountPaise = parseRupees(String(value));
			if (amountPaise === null)
				return fail(400, { error: 'Enter a valid amount for every category.' });
			categories.push({ categoryId: name.slice(9), amountPaise });
		}
		if (incomePaise === null) return fail(400, { error: 'Enter your recurring monthly income.' });
		const emergencyTargetPaise = form.has('emergencyTarget')
			? parseRupees(String(form.get('emergencyTarget')))
			: undefined;
		if (emergencyTargetPaise === null)
			return fail(400, { error: 'Enter a valid emergency reserve target.' });
		const result = await apiRequest(fetch, apiUrl(), '/money-plan', {
			method: 'PUT',
			body: JSON.stringify({
				revision: Number(form.get('revision')),
				incomePaise,
				categories,
				emergencyTargetPaise,
				debtStatus: form.has('debtStatus') ? String(form.get('debtStatus')) : undefined
			})
		});
		return result.ok
			? { success: 'Monthly plan saved. Account balances are unchanged.' }
			: fail(result.status, { error: result.error.message });
	},
	move: async ({ request, fetch }) => {
		const form = await request.formData();
		const amountPaise = parseRupees(String(form.get('amount') ?? ''));
		if (!amountPaise || form.get('confirm') !== 'yes')
			return fail(400, { error: 'Enter an amount and confirm moving it to savings.' });
		const result = await apiRequest(fetch, apiUrl(), '/money-plan/move', {
			method: 'POST',
			body: JSON.stringify({
				revision: Number(form.get('revision')),
				categoryId: String(form.get('categoryId')),
				amountPaise
			})
		});
		return result.ok
			? { success: 'Moved to savings. Your account balance stays the same.' }
			: fail(result.status, { error: result.error.message });
	},
	nextMonth: async ({ request, fetch }) => {
		const form = await request.formData();
		const result = await apiRequest(fetch, apiUrl(), '/money-plan/next-month', {
			method: 'POST',
			body: JSON.stringify({
				revision: Number(form.get('revision')),
				flexibleChoice: String(form.get('flexibleChoice'))
			})
		});
		return result.ok
			? { success: 'New month opened. Unpaid fixed obligations remain protected.' }
			: fail(result.status, { error: result.error.message });
	},
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
