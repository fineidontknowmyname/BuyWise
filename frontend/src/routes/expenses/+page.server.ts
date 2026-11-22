import { randomUUID } from 'node:crypto';
import { fail } from '@sveltejs/kit';
import { todayInIndia } from '#lib/dates.js';
import { parseRupees } from '#lib/money.js';
import { apiRequest, type Account, type Category } from '#lib/server/api.js';
import { apiUrl } from '#lib/server/config.js';
import type { Actions, PageServerLoad } from './$types';

type Candidate = {
	id: string;
	sourceApp: string;
	merchant?: string;
	amountPaise: number;
	occurredOn: string;
	possibleDuplicate: boolean;
	categorySuggestion?: { id: string; name: string; confidence: string };
};

export const load: PageServerLoad = async ({ fetch }) => {
	const base = apiUrl();
	const [accounts, categories, candidates] = await Promise.all([
		apiRequest<Account[]>(fetch, base, '/accounts'),
		apiRequest<Category[]>(fetch, base, '/categories'),
		apiRequest<Candidate[]>(fetch, base, '/expenses/candidates')
	]);
	return {
		accounts: accounts.ok ? accounts.data : null,
		categories: categories.ok
			? categories.data.filter((item) => item.kind === 'expense' && !item.archived)
			: null,
		candidates: candidates.ok ? candidates.data : null,
		today: todayInIndia(),
		requestKey: randomUUID()
	};
};

const field = (form: FormData, name: string) => String(form.get(name) ?? '');

function expenseDraft(form: FormData) {
	return {
		accountId: field(form, 'accountId'),
		categoryId: field(form, 'categoryId'),
		amount: field(form, 'amount'),
		occurredOn: field(form, 'occurredOn'),
		note: field(form, 'note')
	};
}

function expenseBody(draft: ReturnType<typeof expenseDraft>, amountPaise: number) {
	return {
		accountId: draft.accountId,
		categoryId: draft.categoryId,
		occurredOn: draft.occurredOn,
		note: draft.note,
		amountPaise
	};
}

export const actions: Actions = {
	manual: async ({ request, fetch }) => {
		const form = await request.formData();
		const draft = expenseDraft(form);
		const requestKey = field(form, 'key');
		const amount = parseRupees(draft.amount);
		if (!amount)
			return fail(400, {
				manualError: 'Enter a positive amount with up to two decimal places.',
				manualDraft: draft,
				requestKey
			});
		const result = await apiRequest(fetch, apiUrl(), '/expenses', {
			method: 'POST',
			headers: { 'Idempotency-Key': requestKey },
			body: JSON.stringify(expenseBody(draft, amount))
		});
		if (result.ok) return { manualSaved: true };
		return fail(result.status, {
			manualError: result.error.message,
			manualDraft: draft,
			requestKey: result.error.code === 'idempotency_conflict' ? randomUUID() : requestKey
		});
	},
	confirm: async ({ request, fetch }) => {
		const form = await request.formData();
		const id = field(form, 'candidateId');
		const draft = expenseDraft(form);
		const amount = parseRupees(draft.amount);
		if (!amount)
			return fail(400, {
				candidateId: id,
				candidateError: 'Enter a positive amount with up to two decimal places.',
				candidateDraft: draft,
				knownDuplicate: false
			});
		const result = await apiRequest(
			fetch,
			apiUrl(),
			`/expenses/candidates/${encodeURIComponent(id)}/confirm`,
			{
				method: 'POST',
				body: JSON.stringify({
					...expenseBody(draft, amount),
					confirmed: form.get('confirmed') === 'yes',
					allowDuplicate: form.get('allowDuplicate') === 'yes'
				})
			}
		);
		return result.ok
			? { candidateSaved: true }
			: fail(result.status, {
					candidateId: id,
					candidateError: result.error.message,
					candidateDraft: draft,
					knownDuplicate: result.error.code === 'possible_duplicate'
				});
	},
	reject: async ({ request, fetch }) => {
		const id = field(await request.formData(), 'candidateId');
		const result = await apiRequest(
			fetch,
			apiUrl(),
			`/expenses/candidates/${encodeURIComponent(id)}/reject`,
			{ method: 'POST' }
		);
		return result.ok
			? { candidateRejected: true }
			: fail(result.status, { candidateId: id, candidateError: result.error.message });
	}
};
