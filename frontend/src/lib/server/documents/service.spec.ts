import { describe, it, expect } from 'vitest';
import { processDocumentUpload, normalizeDateString, normalizePricePaise } from './service.js';
import { evaluateProductIdentity } from './aiMapper.js';

describe('Document Normalization Utilities', () => {
	it('normalizes various date formats into YYYY-MM-DD', () => {
		expect(normalizeDateString('2026-10-15')).toBe('2026-10-15');
		expect(normalizeDateString('15/10/2026')).toBe('2026-10-15');
		expect(normalizeDateString('2026/10/15')).toBe('2026-10-15');
		expect(normalizeDateString('15-10-2026')).toBe('2026-10-15');
		expect(normalizeDateString('invalid-date')).toBeNull();
	});

	it('normalizes price strings into integer paise', () => {
		expect(normalizePricePaise('29990')).toBe(2999000);
		expect(normalizePricePaise('₹29,990.00')).toBe(2999000);
		expect(normalizePricePaise('$349.99')).toBe(34999);
		expect(normalizePricePaise('Rs. 1,49,999')).toBe(14999900);
		expect(normalizePricePaise('0')).toBeNull();
		expect(normalizePricePaise('N/A')).toBeNull();
	});

	it('detects product identity match and flags substantial mismatches', () => {
		const match = evaluateProductIdentity('Sony WH-1000XM5', 'Sony WH-1000XM5 Wireless');
		expect(match.matches).toBe(true);
		expect(match.warning).toBeUndefined();

		const mismatch = evaluateProductIdentity('Sony WH-1000XM5', 'Apple iPhone 15 Pro');
		expect(mismatch.matches).toBe(false);
		expect(mismatch.warning).toBeDefined();
		expect(mismatch.warning).toContain('Product Identity Mismatch');
	});
});

describe('Document Ingestion Processing', () => {
	it('parses and previews CSV with alternative column headers and deduplicates dates', async () => {
		const csvContent = `Recorded On,Offer Price,Seller,Remarks
2026-10-01,₹29990,Amazon,Great Indian Festival
2026-10-01,₹29490,Amazon,Flash deal (duplicate date)
2026-10-10,₹28990,Croma,Weekend offer
invalid_date,₹25000,Retailer,Should fail
2026-10-15,0,Retailer,Zero price should fail
`;

		const buffer = Buffer.from(csvContent, 'utf-8');
		const preview = await processDocumentUpload(
			buffer,
			'price_tracking.csv',
			'text/csv',
			'goal-123',
			'Sony WH-1000XM5'
		);

		expect(preview.purpose).toBe('price_history');
		expect(preview.validRows.length).toBe(2); // 2026-10-01 and 2026-10-10
		expect(preview.duplicateCount).toBe(1);
		expect(preview.errorRows.length).toBe(2); // invalid_date and 0 price
		expect(preview.validRows[0].pricePaise).toBe(2999000);
		expect(preview.validRows[1].pricePaise).toBe(2899000);
	});

	it('throws an explicit error when AI OCR provider fails on an image upload', async () => {
		// Mock fetch returning Groq error
		const mockFetch = (async () => {
			return new Response(JSON.stringify({ error: { message: 'Rate limit exceeded' } }), {
				status: 429,
				headers: { 'Content-Type': 'application/json' }
			});
		}) as unknown as typeof fetch;

		const dummyImageBuffer = Buffer.from('fake-png-content');

		await expect(
			processDocumentUpload(
				dummyImageBuffer,
				'screenshot.png',
				'image/png',
				'goal-123',
				'Sony WH-1000XM5',
				mockFetch
			)
		).rejects.toThrow(/AI OCR provider failed/i);
	});

	it('uses openai/gpt-oss-120b to interpret ambiguous headers and fails gracefully if AI unavailable', async () => {
		// CSV with totally ambiguous headers that deterministic heuristic cannot resolve
		const ambiguousCsv = `Col_Alpha,Col_Beta,Col_Gamma
val1,val2,val3
`;
		const buffer = Buffer.from(ambiguousCsv, 'utf-8');

		// Mock fetch simulating AI provider failure
		const mockFailingFetch = (async () => {
			return new Response(JSON.stringify({ error: { message: 'Model unavailable' } }), {
				status: 503,
				headers: { 'Content-Type': 'application/json' }
			});
		}) as unknown as typeof fetch;

		await expect(
			processDocumentUpload(
				buffer,
				'unknown.csv',
				'text/csv',
				'goal-123',
				'Sony WH-1000XM5',
				mockFailingFetch
			)
		).rejects.toThrow(/AI column interpretation failed/i);
	});

	it('throws error when document is empty', async () => {
		const emptyBuffer = Buffer.from('', 'utf-8');
		await expect(
			processDocumentUpload(emptyBuffer, 'empty.csv', 'text/csv', 'goal-123', 'Sony WH-1000XM5')
		).rejects.toThrow(/empty or could not be read|no readable text/i);
	});
});
