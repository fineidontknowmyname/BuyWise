import { fail, redirect } from '@sveltejs/kit';
import { resolve } from '$app/paths';
import { apiRequest, type Goal, type Product } from '#lib/server/api.js';
import { apiUrl } from '#lib/server/config.js';
import { parseRupees } from '#lib/money.js';
import { randomUUID } from 'node:crypto';
import { todayInIndia } from '#lib/dates.js';
import type { MoneyPlan } from '#lib/moneyPlan.js';
import { addSavingMonths } from '#lib/reverseEmi.js';
import type { Action } from '@sveltejs/kit';

export const createPurchaseGoal: Action = async ({ request, fetch }) => {
	const form = await request.formData();
	const field = (key: string) => String(form.get(key) ?? '').trim();
	const name = field('name');
	const productUrl = field('productUrl');
	const productName = field('productName');
	const variant = field('variant');
	const target = field('targetPrice');
	const targetPricePaise = target ? parseRupees(target) : 0;
	let productId = field('productId') || null;
	if (!name || targetPricePaise === null) {
		return fail(400, { name, error: 'Enter a goal name and a valid target price.' });
	}

	const base = apiUrl();
	const monthly = field('monthlyContribution') ? parseRupees(field('monthlyContribution')) : 0;
	const savingsNow = field('savingsNow') ? parseRupees(field('savingsNow')) : 0;
	const firstSavingOn = field('firstSavingOn');
	if (
		monthly === null ||
		savingsNow === null ||
		(savingsNow > 0 && !field('savingsAccount')) ||
		(monthly > 0 && (!addSavingMonths(firstSavingOn, 0) || firstSavingOn < todayInIndia()))
	) {
		return fail(400, {
			name,
			error: 'Check the saving amounts, source account and first saving date.'
		});
	}
	if (savingsNow > (targetPricePaise ?? 0))
		return fail(400, { name, error: 'Existing savings cannot exceed the purchase target.' });
	if (monthly > 0 || savingsNow > 0) {
		const money = await apiRequest<MoneyPlan>(fetch, base, '/money-plan');
		if (!money.ok) return fail(money.status, { name, error: money.error.message });
		if (
			!money.data.configured ||
			money.data.reviewRequired ||
			monthly > money.data.unassignedMonthlyPaise ||
			savingsNow > money.data.savingsNowPaise
		) {
			return fail(400, {
				name,
				error:
					'Review My Money: the selected amounts exceed savings available now or monthly savings left after other goals.'
			});
		}
	}
	if (!productId && (productUrl || productName)) {
		const created = await apiRequest<Product>(fetch, base, '/products/tracked', {
			method: 'POST',
			body: JSON.stringify({
				name: productName || name,
				brand: field('brand'),
				imageUrl: field('imageUrl'),
				variant,
				category: 'Shopping',
				notes: field('notes'),
				offer: productUrl
					? {
							listingUrl: productUrl,
							variant,
							variantConfirmed: field('variantConfirmed') === 'yes'
						}
					: null
			})
		});
		if (!created.ok) return fail(created.status, { name, error: created.error.message });
		productId = created.data.id;
	}
	const result = await apiRequest<Goal>(fetch, base, '/goals', {
		method: 'POST',
		body: JSON.stringify({
			name,
			targetPricePaise,
			targetDate: field('targetDate') || null,
			priority: field('priority') || 'medium',
			productId,
			status: 'active'
		})
	});
	if (!result.ok) return fail(result.status, { name, error: result.error.message });
	const destination = resolve('/goals/[id]/plan', { id: result.data.id });
	if (monthly > 0) {
		const plan = await apiRequest(
			fetch,
			base,
			`/goals/${encodeURIComponent(result.data.id)}/plan`,
			{
				method: 'PUT',
				body: JSON.stringify({
					revision: 0,
					mode: 'simple',
					frequency: 'monthly',
					approvedPaise: monthly,
					firstSavingOn
				})
			}
		);
		if (!plan.ok)
			redirect(303, destination + '?' + new URLSearchParams({ setupError: plan.error.message }));
	}
	if (savingsNow > 0) {
		const funding = await apiRequest(
			fetch,
			base,
			`/goals/${encodeURIComponent(result.data.id)}/contributions`,
			{
				method: 'POST',
				body: JSON.stringify({
					accountId: field('savingsAccount'),
					amountPaise: savingsNow,
					idempotencyKey: randomUUID(),
					kind: 'topup'
				})
			}
		);
		if (!funding.ok)
			redirect(303, destination + '?' + new URLSearchParams({ setupError: funding.error.message }));
	}
	redirect(303, destination);
};
