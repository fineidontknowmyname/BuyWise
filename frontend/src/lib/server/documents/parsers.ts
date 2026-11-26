import * as XLSX from 'xlsx';
import mammoth from 'mammoth';
import { callGroq, VISION_MODEL } from '../ai/groqClient.js';

export interface RawExtractedDocument {
	rawText: string;
	headers: string[];
	rows: Array<Record<string, string>>;
}

export interface PDFExtractionResult {
	rawText: string;
	isScanned: boolean;
	headers: string[];
	rows: Array<Record<string, string>>;
}

export function parseCSV(content: string): RawExtractedDocument {
	const lines = content
		.split(/\r?\n/)
		.map((l) => l.trim())
		.filter((l) => l.length > 0);

	if (lines.length === 0) {
		return { rawText: content, headers: [], rows: [] };
	}

	// Detect delimiter: comma, semicolon, tab
	const firstLine = lines[0];
	let delimiter = ',';
	if (firstLine.includes('\t')) delimiter = '\t';
	else if (firstLine.includes(';') && !firstLine.includes(',')) delimiter = ';';

	const splitRow = (row: string) => {
		const out: string[] = [];
		let current = '';
		let inQuotes = false;
		for (let i = 0; i < row.length; i++) {
			const c = row[i];
			if (c === '"') {
				inQuotes = !inQuotes;
			} else if (c === delimiter && !inQuotes) {
				out.push(current.trim().replace(/^["']|["']$/g, ''));
				current = '';
			} else {
				current += c;
			}
		}
		out.push(current.trim().replace(/^["']|["']$/g, ''));
		return out;
	};

	const headers = splitRow(lines[0]);
	const rows: Array<Record<string, string>> = [];

	for (let i = 1; i < lines.length; i++) {
		const vals = splitRow(lines[i]);
		const rowObj: Record<string, string> = {};
		for (let j = 0; j < headers.length; j++) {
			rowObj[headers[j]] = vals[j] || '';
		}
		rows.push(rowObj);
	}

	return {
		rawText: content,
		headers,
		rows
	};
}

export function parseExcel(buffer: Buffer): RawExtractedDocument {
	const workbook = XLSX.read(buffer, { type: 'buffer' });
	const firstSheetName = workbook.SheetNames[0];
	if (!firstSheetName) {
		return { rawText: '', headers: [], rows: [] };
	}

	const worksheet = workbook.Sheets[firstSheetName];
	const sheetData: Array<Record<string, unknown>> = XLSX.utils.sheet_to_json(worksheet, {
		defval: ''
	});

	if (sheetData.length === 0) {
		return { rawText: '', headers: [], rows: [] };
	}

	const headers = Object.keys(sheetData[0]);
	const rows = sheetData.map((row) => {
		const stringRow: Record<string, string> = {};
		for (const [k, v] of Object.entries(row)) {
			stringRow[k] = String(v ?? '').trim();
		}
		return stringRow;
	});

	const rawText = rows
		.map((r) => Object.values(r).join(' '))
		.join('\n')
		.slice(0, 10000);

	return {
		rawText,
		headers,
		rows
	};
}

export async function parsePDF(buffer: Buffer): Promise<PDFExtractionResult> {
	let rawText: string;
	try {
		const { PDFParse } = await import('pdf-parse');
		const parser = new PDFParse({ data: new Uint8Array(buffer) });
		try {
			const data = await parser.getText();
			rawText = (data.text || '').trim();
		} finally {
			await parser.destroy();
		}
	} catch (e: unknown) {
		throw new Error(
			`Failed to parse PDF document: ${e instanceof Error ? e.message : 'Invalid PDF data'}`,
			{ cause: e }
		);
	}

	// Scanned PDF detection: if text is empty or negligible, flag as scanned
	if (rawText.length <= 30) {
		return {
			rawText: '',
			isScanned: true,
			headers: [],
			rows: []
		};
	}

	const table = parseTextLinesToTable(rawText);
	return {
		rawText,
		isScanned: false,
		headers: table.headers,
		rows: table.rows
	};
}

export async function parseDocx(buffer: Buffer): Promise<RawExtractedDocument> {
	try {
		const result = await mammoth.extractRawText({ buffer });
		const rawText = (result.value || '').trim();
		return parseTextLinesToTable(rawText);
	} catch (e: unknown) {
		throw new Error(
			`Failed to parse DOCX document: ${e instanceof Error ? e.message : 'Invalid DOCX data'}`,
			{ cause: e }
		);
	}
}

export async function parseImageWithVision(
	buffer: Buffer,
	mimeType: string,
	fetchFn: typeof fetch = fetch
): Promise<RawExtractedDocument> {
	let normalizedMime = mimeType;
	if (!normalizedMime || normalizedMime === 'application/octet-stream') {
		normalizedMime = 'image/jpeg';
	}

	const base64 = buffer.toString('base64');
	const dataUrl = `data:${normalizedMime};base64,${base64}`;

	const prompt = `Extract all product details, technical specifications, prices, and historical date records from this image.
If there are tabular price observations, extract them with Date, Price, and Notes columns.
Output strictly valid JSON with this schema:
{
  "productName": "string",
  "brand": "string",
  "model": "string",
  "currentPrice": "string",
  "specifications": { "key": "value" },
  "headers": ["Date", "Price", "Retailer", "Notes"],
  "rows": [
    { "Date": "2026-10-01", "Price": "29990", "Retailer": "Amazon", "Notes": "Festival sale" }
  ]
}`;

	const res = await callGroq(
		[
			{
				role: 'user',
				content: [
					{ type: 'text', text: prompt },
					{ type: 'image_url', image_url: { url: dataUrl } }
				]
			}
		],
		{
			fetchFn,
			model: VISION_MODEL,
			jsonMode: true,
			temperature: 0.1
		}
	);

	if (!res.success) {
		throw new Error(`AI OCR provider failed: ${res.error || 'Unable to process image'}`);
	}

	const content = res.data?.choices?.[0]?.message?.content?.trim();
	if (!content) {
		throw new Error('AI OCR provider returned an empty response.');
	}

	let parsed: {
		headers?: string[];
		rows?: Array<Record<string, string>>;
		productName?: string;
		brand?: string;
		model?: string;
		currentPrice?: string;
		specifications?: Record<string, string>;
	};

	try {
		parsed = JSON.parse(content);
	} catch (e: unknown) {
		throw new Error(
			`AI OCR provider returned invalid JSON: ${e instanceof Error ? e.message : 'Parse error'}`,
			{ cause: e }
		);
	}

	const headers =
		Array.isArray(parsed.headers) && parsed.headers.length > 0 ? parsed.headers : ['Date', 'Price'];
	const rows: Array<Record<string, string>> = Array.isArray(parsed.rows) ? parsed.rows : [];

	return {
		rawText: content,
		headers,
		rows
	};
}

export function extractImageFromPdf(buffer: Buffer): { buffer: Buffer; mimeType: string } | null {
	// Look for JPEG SOI (0xFF, 0xD8, 0xFF) and EOI (0xFF, 0xD9)
	const jpegHeader = Buffer.from([0xff, 0xd8, 0xff]);
	const jpegTrailer = Buffer.from([0xff, 0xd9]);

	const startIdx = buffer.indexOf(jpegHeader);
	if (startIdx !== -1) {
		const endIdx = buffer.indexOf(jpegTrailer, startIdx + 3);
		if (endIdx !== -1 && endIdx > startIdx) {
			const jpegBuf = buffer.subarray(startIdx, endIdx + 2);
			if (jpegBuf.length > 500) {
				return { buffer: jpegBuf, mimeType: 'image/jpeg' };
			}
		}
	}

	// Look for PNG signature (0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A)
	const pngHeader = Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]);
	const pngIend = Buffer.from([0x49, 0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82]);

	const pngStart = buffer.indexOf(pngHeader);
	if (pngStart !== -1) {
		const pngEnd = buffer.indexOf(pngIend, pngStart + 8);
		if (pngEnd !== -1) {
			const pngBuf = buffer.subarray(pngStart, pngEnd + 8);
			if (pngBuf.length > 500) {
				return { buffer: pngBuf, mimeType: 'image/png' };
			}
		}
	}

	return null;
}

export async function parseScannedPdfWithOcr(
	buffer: Buffer,
	fetchFn: typeof fetch = fetch
): Promise<RawExtractedDocument> {
	const extracted = extractImageFromPdf(buffer);
	if (!extracted) {
		throw new Error(
			'This PDF appears to be a scanned document without a selectable text layer. Please upload the document directly as an image (PNG or JPEG) or provide a text-based PDF.'
		);
	}

	return parseImageWithVision(extracted.buffer, extracted.mimeType, fetchFn);
}

function parseTextLinesToTable(text: string): RawExtractedDocument {
	const lines = text
		.split(/\r?\n/)
		.map((l) => l.trim())
		.filter((l) => l.length > 0);

	// Detect lines that look like date + price (e.g., "2026-10-01, 29990" or "01/10/2026 ₹29,990")
	const datePriceRegex =
		/(\d{4}[-/.]\d{1,2}[-/.]\d{1,2}|\d{1,2}[-/.]\d{1,2}[-/.]\d{2,4})[^\d]+(?:₹|Rs\.?|INR|\$)?\s*([\d,]+(?:\.\d{2})?)/i;
	const detectedRows: Array<Record<string, string>> = [];

	for (const line of lines) {
		const match = line.match(datePriceRegex);
		if (match) {
			detectedRows.push({
				Date: match[1],
				Price: match[2],
				Raw: line
			});
		}
	}

	const headers = detectedRows.length > 0 ? ['Date', 'Price', 'Raw'] : [];
	return {
		rawText: text,
		headers,
		rows: detectedRows
	};
}
