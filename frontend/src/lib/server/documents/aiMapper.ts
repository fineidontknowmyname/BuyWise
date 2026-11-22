import { callGroq, DEFAULT_MODEL } from '../ai/groqClient.js';
import type { DocumentPurpose, DocumentProductDetails, ProductIdentityMatch } from './types.js';

export interface ColumnMappingResult {
	dateColumn?: string;
	priceColumn?: string;
	originalPriceColumn?: string;
	retailerColumn?: string;
	notesColumn?: string;
	purpose: DocumentPurpose;
	productDetails?: DocumentProductDetails;
}

export async function detectColumnMappings(
	headers: string[],
	sampleRows: Array<Record<string, string>>,
	rawText: string,
	fetchFn: typeof fetch = fetch
): Promise<ColumnMappingResult> {
	// 1. Deterministic heuristic mapping
	let dateCol: string | undefined;
	let priceCol: string | undefined;
	let origPriceCol: string | undefined;
	let retailerCol: string | undefined;
	let notesCol: string | undefined;

	for (const h of headers) {
		const lower = h.toLowerCase().trim();
		if (!dateCol && /(?:date|observed|recorded|time|day|period)/i.test(lower)) {
			dateCol = h;
		} else if (
			!origPriceCol &&
			/(?:mrp|original|list.?price|regular.?price|crossed)/i.test(lower)
		) {
			origPriceCol = h;
		} else if (!priceCol && /(?:price|amount|rate|cost|selling|offer|val)/i.test(lower)) {
			priceCol = h;
		} else if (!retailerCol && /(?:retailer|store|seller|vendor|merchant|source)/i.test(lower)) {
			retailerCol = h;
		} else if (!notesCol && /(?:note|remark|comment|reason|event|memo)/i.test(lower)) {
			notesCol = h;
		}
	}

	// If we found date and price deterministically, purpose is price_history
	if (dateCol && priceCol) {
		return {
			dateColumn: dateCol,
			priceColumn: priceCol,
			originalPriceColumn: origPriceCol,
			retailerColumn: retailerCol,
			notesColumn: notesCol,
			purpose: 'price_history'
		};
	}

	// 2. AI-assisted mapping with openai/gpt-oss-120b when ambiguous
	if (headers.length > 0) {
		const prompt = `Given these table headers: ${JSON.stringify(headers)}
And sample row: ${JSON.stringify(sampleRows[0] || {})}
Text excerpt: """${rawText.slice(0, 500)}"""

Identify:
1. Which column represents the observation date?
2. Which column represents the selling/current price?
3. Which column represents original/list price if any?
4. Which column represents retailer or notes?
5. What is the document purpose? (price_history, specifications, quote_or_invoice, mixed)
6. If any product name or brand is mentioned in the excerpt, extract it.

Output strictly valid JSON with this schema:
{
  "dateColumn": "string",
  "priceColumn": "string",
  "originalPriceColumn": "string",
  "retailerColumn": "string",
  "notesColumn": "string",
  "purpose": "price_history",
  "productName": "string",
  "brand": "string"
}`;

		const res = await callGroq(
			[
				{
					role: 'system',
					content: 'You are an expert tabular data mapping assistant. Output JSON only.'
				},
				{ role: 'user', content: prompt }
			],
			{
				fetchFn,
				model: DEFAULT_MODEL,
				jsonMode: true,
				temperature: 0.1
			}
		);

		if (!res.success) {
			if (!dateCol || !priceCol) {
				throw new Error(
					`AI column interpretation failed: ${res.error || 'AI provider unavailable'}. Please verify document columns.`
				);
			}
		} else {
			const content = res.data?.choices?.[0]?.message?.content?.trim();
			if (content) {
				try {
					const p = JSON.parse(content);
					return {
						dateColumn: p.dateColumn && headers.includes(p.dateColumn) ? p.dateColumn : dateCol,
						priceColumn:
							p.priceColumn && headers.includes(p.priceColumn) ? p.priceColumn : priceCol,
						originalPriceColumn:
							p.originalPriceColumn && headers.includes(p.originalPriceColumn)
								? p.originalPriceColumn
								: origPriceCol,
						retailerColumn:
							p.retailerColumn && headers.includes(p.retailerColumn)
								? p.retailerColumn
								: retailerCol,
						notesColumn:
							p.notesColumn && headers.includes(p.notesColumn) ? p.notesColumn : notesCol,
						purpose: p.purpose || 'price_history',
						productDetails: p.productName ? { name: p.productName, brand: p.brand } : undefined
					};
				} catch (e: unknown) {
					if (!dateCol || !priceCol) {
						throw new Error('AI column interpretation returned invalid JSON.', { cause: e });
					}
				}
			}
		}
	}

	if (!dateCol || !priceCol) {
		throw new Error(
			`Unable to identify Date and Price columns from document headers (${headers.join(', ')}). Please verify document headers.`
		);
	}

	return {
		dateColumn: dateCol,
		priceColumn: priceCol,
		originalPriceColumn: origPriceCol,
		retailerColumn: retailerCol,
		notesColumn: notesCol,
		purpose: 'price_history'
	};
}

export function evaluateProductIdentity(
	goalProductName: string,
	documentProductName?: string
): ProductIdentityMatch {
	if (!goalProductName || !documentProductName) {
		return {
			matches: true,
			similarityScore: 100,
			goalProductName,
			documentProductName: documentProductName || goalProductName
		};
	}

	const norm1 = goalProductName
		.toLowerCase()
		.replace(/[^a-z0-9]/g, ' ')
		.split(/\s+/)
		.filter(Boolean);
	const norm2 = documentProductName
		.toLowerCase()
		.replace(/[^a-z0-9]/g, ' ')
		.split(/\s+/)
		.filter(Boolean);

	// Calculate word overlap (Jaccard similarity)
	const set1 = new Set(norm1);
	const set2 = new Set(norm2);
	let intersection = 0;
	for (const w of set1) {
		if (set2.has(w)) intersection++;
	}
	const union = new Set([...norm1, ...norm2]).size;
	const score = union > 0 ? Math.round((intersection / union) * 100) : 0;

	if (score < 30 && norm2.length > 1) {
		return {
			matches: false,
			similarityScore: score,
			goalProductName,
			documentProductName,
			warning: `Product Identity Mismatch: The uploaded document appears to describe "${documentProductName}", but your purchase goal is for "${goalProductName}". Please verify before importing.`
		};
	}

	return {
		matches: true,
		similarityScore: score,
		goalProductName,
		documentProductName
	};
}
