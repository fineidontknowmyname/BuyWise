import { parseSignedRupees } from './money.js';

export const maxExpenseRows = 200;
export const maxExpenseFileBytes = 400 * 1024;
export type ExpenseMapping = {
	date: string;
	amount: string;
	debit: string;
	credit: string;
	description: string;
	reference: string;
	kind: string;
};
export type SourceExpense = {
	amountPaise: number | null;
	occurredOn: string;
	description: string;
	reference: string;
	kind: string;
	warnings: string[];
};
export type ImportRow = SourceExpense & {
	index: number;
	status: 'pending' | 'confirmed' | 'reversed';
	possibleDuplicate: boolean;
	transactionId: string;
	accountId?: string;
	categoryId?: string;
};
export type ExpenseImport = {
	id: string;
	filename: string;
	format: string;
	status: string;
	revision: number;
	createdAt: string;
	rows: ImportRow[];
};

export function parseExpenseCSV(text: string): {
	headers: string[];
	rows: Record<string, string>[];
} {
	if (text.includes('\0')) throw new Error('This does not look like a text CSV file.');
	text = text.replace(/^\uFEFF/, '');
	const first = text.split(/\r?\n/, 1)[0];
	const delimiter = first.includes('\t')
		? '\t'
		: first.includes(';') && !first.includes(',')
			? ';'
			: ',';
	const records: string[][] = [];
	let row: string[] = [],
		cell = '',
		quoted = false,
		closed = false;
	for (let i = 0; i < text.length; i++) {
		const char = text[i];
		if (quoted) {
			if (char === '"' && text[i + 1] === '"') {
				cell += '"';
				i++;
			} else if (char === '"') {
				quoted = false;
				closed = true;
			} else cell += char;
		} else if (char === '"' && !cell.trim() && !closed) {
			quoted = true;
			cell = '';
		} else if (char === delimiter) {
			row.push(cell.trim());
			cell = '';
			closed = false;
		} else if (char === '\n' || char === '\r') {
			if (char === '\r' && text[i + 1] === '\n') i++;
			row.push(cell.trim());
			if (row.some(Boolean)) records.push(row);
			row = [];
			cell = '';
			closed = false;
			if (records.length > maxExpenseRows + 1)
				throw new Error('Import no more than 200 rows at a time.');
		} else {
			if (closed && char.trim()) throw new Error('The CSV has text outside a quoted field.');
			if (char === '"') throw new Error('The CSV contains an unescaped quote.');
			cell += char;
		}
	}
	if (quoted) throw new Error('The CSV has an unfinished quoted field.');
	row.push(cell.trim());
	if (row.some(Boolean)) records.push(row);
	if (records.length < 2 || records.length > maxExpenseRows + 1)
		throw new Error('Include a header and between 1 and 200 data rows.');
	const headers = records.shift()!;
	if (
		headers.some((header) => !header || header.length > 100) ||
		new Set(headers.map((header) => header.toLowerCase())).size !== headers.length
	)
		throw new Error('Use unique, non-empty column headings (up to 100 characters).');
	return {
		headers,
		rows: records.map((cells, index) => {
			if (cells.length > headers.length)
				throw new Error(
					`Row ${index + 2} has too many columns. Quote amounts that contain commas.`
				);
			return Object.fromEntries(headers.map((header, column) => [header, cells[column] ?? '']));
		})
	};
}

export function suggestExpenseMapping(headers: string[]): ExpenseMapping {
	const find = (names: string[]) =>
		headers.find((header) => names.includes(header.toLowerCase().replace(/[^a-z]/g, ''))) ?? '';
	return {
		date: find(['date', 'transactiondate', 'occurredon', 'valuedate']),
		amount: find(['amount', 'amountinr', 'total', 'transactionamount']),
		debit: find(['debit', 'debitamount', 'withdrawal', 'withdrawalamount']),
		credit: find(['credit', 'creditamount', 'deposit', 'depositamount']),
		description: find(['description', 'merchant', 'narration', 'note', 'particulars', 'details']),
		reference: find(['reference', 'referencenumber', 'transactionid', 'ref', 'utr']),
		kind: find(['type', 'kind', 'transactiontype', 'drcr'])
	};
}

export function expenseDate(value: string, order: string): string {
	let year: number, month: number, day: number;
	const iso = /^(\d{4})-(\d{2})-(\d{2})$/.exec(value.trim());
	if (iso) [, year, month, day] = iso.map(Number);
	else {
		const parts = /^(\d{1,2})[/.-](\d{1,2})[/.-](\d{4})$/.exec(value.trim());
		if (!parts) return '';
		year = Number(parts[3]);
		month = Number(parts[order === 'MDY' ? 1 : 2]);
		day = Number(parts[order === 'MDY' ? 2 : 1]);
	}
	if (
		year < 1000 ||
		year > 9999 ||
		month < 1 ||
		month > 12 ||
		day < 1 ||
		day > new Date(Date.UTC(year, month, 0)).getUTCDate()
	)
		return '';
	return `${year}-${String(month).padStart(2, '0')}-${String(day).padStart(2, '0')}`;
}

function importedAmount(raw: string): number | null {
	let value = raw
		.trim()
		.replace(/^(?:INR|Rs\.?|₹)\s*/i, '')
		.replace(/\s*(?:INR|Rs\.?)$/i, '');
	if (/^\(.*\)$/.test(value)) value = '-' + value.slice(1, -1);
	const numeric = value.replace(/^[+-]/, '');
	if (!/^(?:\d+|\d{1,3}(?:,\d{3})+|\d{1,2}(?:,\d{2})*,\d{3})(?:\.\d{1,2})?$/.test(numeric))
		return null;
	return parseSignedRupees(value.replace(/^\+/, ''));
}

export function mapExpenseRows(
	rows: Record<string, string>[],
	mapping: ExpenseMapping,
	order: string,
	signed: string
): SourceExpense[] {
	if (!mapping.date || !(mapping.amount || mapping.debit))
		throw new Error('Map a date column and either an amount or debit column.');
	return rows.map((row) => {
		const warnings: string[] = [];
		let amount: number | null = null;
		let kind = 'unknown';
		if (mapping.debit) {
			const debit = importedAmount(row[mapping.debit] ?? '');
			const credit = importedAmount(row[mapping.credit] ?? '');
			if (debit !== null && debit > 0 && (credit === null || credit === 0)) {
				amount = debit;
				kind = 'expense';
			} else if (credit !== null && credit > 0 && (debit === null || debit === 0)) {
				amount = credit;
				kind = 'income';
			} else warnings.push('Check the debit and credit fields.');
		} else {
			amount = importedAmount(row[mapping.amount] ?? '');
			if (amount !== null && amount !== 0) {
				kind =
					signed === 'negative'
						? amount < 0
							? 'expense'
							: 'income'
						: amount > 0
							? 'expense'
							: 'refund';
				amount = Math.abs(amount);
			}
		}
		const type = (row[mapping.kind] ?? '').trim().toLowerCase();
		if (['transfer', 'internal transfer'].includes(type)) kind = 'transfer';
		else if (['refund', 'reversal'].includes(type)) kind = 'refund';
		else if (['income', 'credit', 'cr', 'deposit'].includes(type)) kind = 'income';
		else if (['expense', 'debit', 'dr', 'withdrawal'].includes(type)) kind = 'expense';
		if (amount === null || amount <= 0) {
			amount = null;
			warnings.push('Amount needs review. Only INR amounts are supported.');
		}
		const occurredOn = expenseDate(row[mapping.date] ?? '', order);
		if (!occurredOn)
			warnings.push('Date could not be read. Choose the correct date during review.');
		return {
			amountPaise: amount,
			occurredOn,
			description: (row[mapping.description] ?? '').slice(0, 500),
			reference: (row[mapping.reference] ?? '').slice(0, 200),
			kind,
			warnings
		};
	});
}
