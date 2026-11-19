import {
	parseCSV,
	parseExcel,
	parsePDF,
	parseDocx,
	parseImageWithVision,
	parseScannedPdfWithOcr
} from './parsers.js';
import { detectColumnMappings, evaluateProductIdentity } from './aiMapper.js';
import type { DocumentPreviewResult, ParsedPriceRow } from './types.js';
import { formatRupees } from '#lib/money.js';

export async function processDocumentUpload(
	fileBuffer: Buffer,
	filename: string,
	mimeType: string,
	goalId: string,
	goalProductName: string,
	fetchFn: typeof fetch = fetch
): Promise<DocumentPreviewResult> {
	const ext = filename.split('.').pop()?.toLowerCase() || '';

	let fileType: DocumentPreviewResult['fileType'] = 'csv';
	if (['xlsx', 'xls'].includes(ext)) fileType = 'xlsx';
	else if (ext === 'pdf') fileType = 'pdf';
	else if (['docx', 'doc'].includes(ext)) fileType = 'docx';
	else if (['png', 'jpg', 'jpeg', 'webp'].includes(ext)) fileType = 'image';

	// 1. Extract raw data according to format
	let rawDoc = { rawText: '', headers: [] as string[], rows: [] as Array<Record<string, string>> };

	if (fileType === 'csv') {
		rawDoc = parseCSV(fileBuffer.toString('utf-8'));
	} else if (fileType === 'xlsx') {
		rawDoc = parseExcel(fileBuffer);
	} else if (fileType === 'pdf') {
		const pdfRes = await parsePDF(fileBuffer);
		if (pdfRes.isScanned) {
			rawDoc = await parseScannedPdfWithOcr(fileBuffer, fetchFn);
		} else {
			rawDoc = {
				rawText: pdfRes.rawText,
				headers: pdfRes.headers,
				rows: pdfRes.rows
			};
		}
	} else if (fileType === 'docx') {
		rawDoc = await parseDocx(fileBuffer);
	} else if (fileType === 'image') {
		rawDoc = await parseImageWithVision(fileBuffer, mimeType, fetchFn);
	}

	if (rawDoc.rows.length === 0 && rawDoc.rawText.trim().length === 0) {
		throw new Error('The uploaded document contains no readable text or table data.');
	}

	// 2. Detect column mappings & purpose
	const mappingResult = await detectColumnMappings(
		rawDoc.headers,
		rawDoc.rows,
		rawDoc.rawText,
		fetchFn
	);

	// 3. Product identity matching
	const identityMatch = evaluateProductIdentity(
		goalProductName,
		mappingResult.productDetails?.name
	);

	// 4. Deterministic row validation & normalization
	const validRows: ParsedPriceRow[] = [];
	const errorRows: Array<{ line: number; raw: string; error: string }> = [];
	const seenDates = new Set<string>();
	let duplicateCount = 0;
	let detectedCurrency = 'INR';

	const dateCol = mappingResult.dateColumn || 'Date';
	const priceCol = mappingResult.priceColumn || 'Price';
	const origPriceCol = mappingResult.originalPriceColumn;
	const retailerCol = mappingResult.retailerColumn;
	const notesCol = mappingResult.notesColumn;

	let lineNum = 1;
	for (const row of rawDoc.rows) {
		lineNum++;
		const rawDate = row[dateCol] || '';
		const rawPrice = row[priceCol] || '';
		const rawOrig = origPriceCol ? row[origPriceCol] : '';
		const rawRetailer = retailerCol ? row[retailerCol] : '';
		const rawNotes = notesCol ? row[notesCol] : '';

		if (!rawDate && !rawPrice) {
			continue;
		}

		// Normalize Date
		const normalizedDate = normalizeDateString(rawDate);
		if (!normalizedDate) {
			errorRows.push({
				line: lineNum,
				raw: JSON.stringify(row),
				error: `Invalid date format: "${rawDate}"`
			});
			continue;
		}

		// Check for duplicate date within the same document
		if (seenDates.has(normalizedDate)) {
			duplicateCount++;
			continue;
		}

		// Normalize Price
		const parsedPrice = normalizePricePaise(rawPrice);
		if (!parsedPrice || parsedPrice <= 0) {
			errorRows.push({
				line: lineNum,
				raw: JSON.stringify(row),
				error: `Invalid or zero price value: "${rawPrice}"`
			});
			continue;
		}

		seenDates.add(normalizedDate);

		// Currency detection
		if (rawPrice.includes('$') || rawPrice.includes('USD')) detectedCurrency = 'USD';
		else if (rawPrice.includes('€') || rawPrice.includes('EUR')) detectedCurrency = 'EUR';
		else if (rawPrice.includes('£') || rawPrice.includes('GBP')) detectedCurrency = 'GBP';

		// Price type
		let priceType: ParsedPriceRow['priceType'] = 'regular';
		if (rawOrig) {
			const parsedOrig = normalizePricePaise(rawOrig);
			if (parsedOrig && parsedOrig > parsedPrice) {
				priceType = 'discounted';
			}
		}

		validRows.push({
			line: lineNum,
			observedOn: normalizedDate,
			pricePaise: parsedPrice,
			priceFormatted: formatRupees(parsedPrice),
			currency: detectedCurrency,
			priceType,
			retailer: rawRetailer.trim(),
			source: rawRetailer.trim() || 'document_upload',
			notes: rawNotes.trim(),
			isValid: true
		});
	}

	// Sort observations chronologically
	validRows.sort((a, b) => a.observedOn.localeCompare(b.observedOn));

	const previewResult: DocumentPreviewResult = {
		id: crypto.randomUUID(),
		goalId,
		filename,
		fileType,
		fileSize: fileBuffer.length,
		purpose: mappingResult.purpose,
		detectedColumns: rawDoc.headers,
		columnMappings: {
			dateColumn: dateCol,
			priceColumn: priceCol,
			originalPriceColumn: origPriceCol,
			retailerColumn: retailerCol,
			notesColumn: notesCol
		},
		productDetails: mappingResult.productDetails,
		identityMatch,
		validRows,
		errorRows,
		duplicateCount,
		summary: {
			totalRows: rawDoc.rows.length,
			validCount: validRows.length,
			errorCount: errorRows.length,
			detectedCurrency,
			dateRange:
				validRows.length > 0
					? { from: validRows[0].observedOn, to: validRows[validRows.length - 1].observedOn }
					: undefined
		}
	};

	return previewResult;
}

export function normalizeDateString(raw: string): string | null {
	if (!raw) return null;
	const clean = raw.trim();

	// YYYY-MM-DD
	if (/^\d{4}-\d{2}-\d{2}$/.test(clean)) {
		return clean;
	}

	// YYYY/MM/DD or YYYY.MM.DD
	const isoMatch = clean.match(/^(\d{4})[-/.](\d{1,2})[-/.](\d{1,2})$/);
	if (isoMatch) {
		const y = isoMatch[1];
		const m = isoMatch[2].padStart(2, '0');
		const d = isoMatch[3].padStart(2, '0');
		return `${y}-${m}-${d}`;
	}

	// DD/MM/YYYY or DD-MM-YYYY
	const dmyMatch = clean.match(/^(\d{1,2})[-/.](\d{1,2})[-/.](\d{4})$/);
	if (dmyMatch) {
		const d = dmyMatch[1].padStart(2, '0');
		const m = dmyMatch[2].padStart(2, '0');
		const y = dmyMatch[3];
		return `${y}-${m}-${d}`;
	}

	// Native Date parse fallback
	const timestamp = Date.parse(clean);
	if (!isNaN(timestamp)) {
		const dt = new Date(timestamp);
		return dt.toISOString().split('T')[0];
	}

	return null;
}

export function normalizePricePaise(raw: string): number | null {
	if (!raw) return null;
	// Cleanly remove currency prefixes (Rs., INR, USD, EUR, etc.), symbols, and commas
	let clean = raw.replace(/^(?:rs\.?|inr|usd|eur|gbp)\s*/i, '').trim();
	clean = clean.replace(/[₹$€£,\s]/g, '').trim();
	const num = parseFloat(clean);
	if (isNaN(num) || num <= 0) {
		return null;
	}
	return Math.round(num * 100);
}
