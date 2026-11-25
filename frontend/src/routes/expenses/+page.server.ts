import { randomUUID, createHash } from 'node:crypto';
import { fail, redirect } from '@sveltejs/kit';
import { todayInIndia } from '#lib/dates.js';
import { parseRupees } from '#lib/money.js';
import { apiRequest, type Account, type Category, type Transaction } from '#lib/server/api.js';
import { apiUrl } from '#lib/server/config.js';
import {
	maxExpenseFileBytes,
	parseExpenseCSV,
	suggestExpenseMapping,
	mapExpenseRows,
	type ExpenseImport,
	type ExpenseMapping
} from '#lib/expenseImport.js';
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
type ExpenseList = { items: Transaction[]; count: number; totalPaise: number; month: string };

export const load: PageServerLoad = async ({ fetch, url }) => {
	const base = apiUrl();
	const filters = {
		month: url.searchParams.get('month') || todayInIndia().slice(0, 7),
		account: url.searchParams.get('account') || '',
		category: url.searchParams.get('category') || '',
		search: (url.searchParams.get('search') || '').slice(0, 200)
	};
	const query = new URLSearchParams(filters);
	const importId = url.searchParams.get('import');
	const [accounts, categories, candidates, expenses, imports, selected] = await Promise.all([
		apiRequest<Account[]>(fetch, base, '/accounts'),
		apiRequest<Category[]>(fetch, base, '/categories'),
		apiRequest<Candidate[]>(fetch, base, '/expenses/candidates'),
		apiRequest<ExpenseList>(fetch, base, '/expenses?' + query),
		apiRequest<ExpenseImport[]>(fetch, base, '/expenses/imports'),
		importId
			? apiRequest<ExpenseImport>(fetch, base, '/expenses/imports/' + encodeURIComponent(importId))
			: Promise.resolve(null)
	]);
	return {
		accounts: accounts.ok ? accounts.data : null,
		categories: categories.ok ? categories.data.filter((item) => item.kind === 'expense') : null,
		candidates: candidates.ok ? candidates.data : null,
		expenses: expenses.ok ? expenses.data : null,
		expensesError: expenses.ok ? null : expenses.error.message,
		imports: imports.ok ? imports.data : null,
		importsError: imports.ok ? null : imports.error.message,
		selectedImport: selected?.ok ? selected.data : null,
		selectedImportError: selected && !selected.ok ? selected.error.message : null,
		today: todayInIndia(),
		requestKey: randomUUID(),
		filters
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
	upload: async ({ request, fetch }) => {
		const form = await request.formData();
		try {
			const file = form.get('file');
			if (!(file instanceof File) || !file.size || file.size > maxExpenseFileBytes)
				return fail(400, { uploadError: 'Choose a CSV, PDF, JPEG or PNG file of up to 400 KB.' });
			const extension = file.name.split('.').pop()?.toLowerCase();
			const buffer = Buffer.from(await file.arrayBuffer());
			let format = 'csv';
			let rows;
			if (extension === 'csv') {
				const parsed = parseExpenseCSV(buffer.toString('utf8'));
				const suggested = suggestExpenseMapping(parsed.headers);
				const mapping = Object.fromEntries(
					Object.entries(suggested).map(([key, value]) => [
						key,
						form.has('map_' + key) ? String(form.get('map_' + key)) : value
					])
				) as ExpenseMapping;
				if (Object.values(mapping).some((value) => value && !parsed.headers.includes(value)))
					throw new Error('A mapped column is missing from this file. Select the file again.');
				rows = mapExpenseRows(
					parsed.rows,
					mapping,
					field(form, 'dateOrder'),
					field(form, 'signedMode')
				);
			} else {
				if (form.get('documentConsent') !== 'yes')
					return fail(400, {
						uploadError:
							'Confirm document processing before uploading a PDF or image. CSV does not require external document processing.'
					});
				if (!['pdf', 'png', 'jpg', 'jpeg'].includes(extension ?? ''))
					throw new Error('Use a CSV, PDF, JPEG or PNG file.');
				format = extension === 'pdf' ? 'pdf' : 'image';
				const { extractExpenseDocument } =
					await import('#lib/server/documents/expenseExtraction.js');
				rows = await extractExpenseDocument(
					buffer,
					extension === 'pdf'
						? 'application/pdf'
						: extension === 'png'
							? 'image/png'
							: 'image/jpeg',
					fetch
				);
			}
			const result = await apiRequest<ExpenseImport>(fetch, apiUrl(), '/expenses/imports', {
				method: 'POST',
				body: JSON.stringify({
					filename: file.name.replace(/[\\/]/g, '_').slice(0, 200),
					format,
					fingerprint: createHash('sha256').update(buffer).digest('hex'),
					rows
				})
			});
			if (!result.ok) return fail(result.status, { uploadError: result.error.message });
			redirect(303, '/expenses?import=' + encodeURIComponent(result.data.id) + '#import-review');
		} catch (error) {
			if (error && typeof error === 'object' && 'status' in error && 'location' in error)
				throw error;
			return fail(400, {
				uploadError:
					error instanceof Error
						? error.message
						: 'The file could not be read. Use CSV or manual entry.'
			});
		}
	},
	recordImport: async ({ request, fetch }) => {
		const form = await request.formData();
		const id = field(form, 'importId');
		const indices = form.getAll('selected').map(Number);
		if (
			!indices.length ||
			new Set(indices).size !== indices.length ||
			form.get('reviewed') !== 'yes'
		)
			return fail(400, { importError: 'Select rows and confirm you have checked them.' });
		const rows = indices.map((index) => ({
			index,
			accountId: field(form, 'account_' + index),
			categoryId: field(form, 'category_' + index),
			amountPaise: parseRupees(field(form, 'amount_' + index)),
			occurredOn: field(form, 'date_' + index),
			note: field(form, 'note_' + index),
			allowDuplicate: form.get('duplicate_' + index) === 'yes'
		}));
		if (
			rows.some(
				(row) =>
					!row.amountPaise || !Number.isInteger(row.index) || row.index < 0 || row.index >= 200
			)
		)
			return fail(400, { importError: 'Selected rows need valid amounts.' });
		const result = await apiRequest(
			fetch,
			apiUrl(),
			'/expenses/imports/' + encodeURIComponent(id) + '/confirm',
			{
				method: 'POST',
				body: JSON.stringify({ revision: Number(field(form, 'revision')), confirmed: true, rows })
			}
		);
		return result.ok
			? { importSaved: true }
			: fail(result.status, { importError: result.error.message });
	},
	finishImport: async ({ request, fetch }) => {
		const form = await request.formData();
		const operation = field(form, 'operation');
		if (!['reverse', 'discard'].includes(operation) || form.get('confirmed') !== 'yes')
			return fail(400, { importError: 'Review and confirm this import action.' });
		const result = await apiRequest(
			fetch,
			apiUrl(),
			'/expenses/imports/' + encodeURIComponent(field(form, 'importId')) + '/' + operation,
			{
				method: 'POST',
				body: JSON.stringify({ confirmed: true, reason: field(form, 'reason') })
			}
		);
		return result.ok
			? { importFinished: true }
			: fail(result.status, { importError: result.error.message });
	},
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
