// CSV Parser for Historical Product Prices

export interface ParsedPriceRow {
	pricePaise: number;
	observedOn: string; // YYYY-MM-DD
	source: string;
}

export interface CSVParseResult {
	validRows: ParsedPriceRow[];
	errorRows: Array<{ line: number; raw: string; error: string }>;
	duplicateCount: number;
}

export function parsePriceHistoryCSV(csvText: string): CSVParseResult {
	const lines = csvText
		.split(/\r?\n/)
		.map((l) => l.trim())
		.filter((l) => l.length > 0);
	if (lines.length < 2) {
		return {
			validRows: [],
			errorRows: [
				{ line: 1, raw: '', error: 'CSV must contain a header and at least one data row.' }
			],
			duplicateCount: 0
		};
	}

	const header = lines[0]
		.toLowerCase()
		.split(',')
		.map((h) => h.trim().replace(/^["']|["']$/g, ''));
	const dateIdx = header.findIndex((h) => h.includes('date') || h.includes('time') || h === 'on');
	const priceIdx = header.findIndex(
		(h) => h.includes('price') || h.includes('cost') || h.includes('amount') || h.includes('rate')
	);
	const sourceIdx = header.findIndex(
		(h) => h.includes('source') || h.includes('retailer') || h.includes('store')
	);

	if (dateIdx === -1 || priceIdx === -1) {
		return {
			validRows: [],
			errorRows: [
				{ line: 1, raw: lines[0], error: 'Header must contain "date" and "price" columns.' }
			],
			duplicateCount: 0
		};
	}

	const validRows: ParsedPriceRow[] = [];
	const errorRows: Array<{ line: number; raw: string; error: string }> = [];
	const seenDates = new Set<string>();
	let duplicateCount = 0;

	for (let i = 1; i < lines.length; i++) {
		const rawLine = lines[i];
		const cols = rawLine.split(',').map((c) => c.trim().replace(/^["']|["']$/g, ''));

		if (cols.length <= Math.max(dateIdx, priceIdx)) {
			errorRows.push({ line: i + 1, raw: rawLine, error: 'Missing required columns' });
			continue;
		}

		const rawDate = cols[dateIdx];
		const rawPrice = cols[priceIdx];
		const source = sourceIdx !== -1 && cols[sourceIdx] ? cols[sourceIdx] : 'csv_import';

		const normalizedDate = normalizeDate(rawDate);
		if (!normalizedDate) {
			errorRows.push({
				line: i + 1,
				raw: rawLine,
				error: `Invalid date format: "${rawDate}" (use YYYY-MM-DD or DD/MM/YYYY)`
			});
			continue;
		}

		const pricePaise = parsePriceToPaise(rawPrice);
		if (pricePaise === null || pricePaise <= 0) {
			errorRows.push({ line: i + 1, raw: rawLine, error: `Invalid price amount: "${rawPrice}"` });
			continue;
		}

		if (seenDates.has(normalizedDate)) {
			duplicateCount++;
			continue; // skip duplicate observations on the same date
		}
		seenDates.add(normalizedDate);

		validRows.push({
			pricePaise,
			observedOn: normalizedDate,
			source
		});
	}

	// Sort chronologically ascending
	validRows.sort((a, b) => a.observedOn.localeCompare(b.observedOn));

	return {
		validRows,
		errorRows,
		duplicateCount
	};
}

function normalizeDate(d: string): string | null {
	const trimmed = d.trim();
	// YYYY-MM-DD
	if (/^\d{4}-\d{2}-\d{2}$/.test(trimmed)) {
		return trimmed;
	}
	// DD/MM/YYYY or DD-MM-YYYY
	const dmyMatch = trimmed.match(/^(\d{1,2})[/-](\d{1,2})[/-](\d{4})$/);
	if (dmyMatch) {
		const day = dmyMatch[1].padStart(2, '0');
		const month = dmyMatch[2].padStart(2, '0');
		const year = dmyMatch[3];
		return `${year}-${month}-${day}`;
	}
	// Date parse fallback
	const parsed = new Date(trimmed);
	if (!isNaN(parsed.getTime())) {
		return parsed.toISOString().split('T')[0];
	}
	return null;
}

function parsePriceToPaise(p: string): number | null {
	const cleaned = p.replace(/[₹$,\s]/g, '');
	const num = parseFloat(cleaned);
	if (isNaN(num) || num <= 0) return null;
	return Math.round(num * 100);
}
