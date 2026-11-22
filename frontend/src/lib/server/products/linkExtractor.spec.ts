import { describe, it, expect, vi } from 'vitest';
import {
	extractProductFromUrl,
	isPrivateOrRestrictedIp,
	normalizeProductUrl
} from './linkExtractor.js';

describe('isPrivateOrRestrictedIp', () => {
	it('detects loopback addresses', () => {
		expect(isPrivateOrRestrictedIp('127.0.0.1')).toBe(true);
		expect(isPrivateOrRestrictedIp('127.0.1.5')).toBe(true);
		expect(isPrivateOrRestrictedIp('::1')).toBe(true);
	});

	it('detects RFC 1918 private IPv4 ranges', () => {
		expect(isPrivateOrRestrictedIp('10.0.0.1')).toBe(true);
		expect(isPrivateOrRestrictedIp('192.168.1.1')).toBe(true);
		expect(isPrivateOrRestrictedIp('172.16.0.1')).toBe(true);
		expect(isPrivateOrRestrictedIp('172.31.255.255')).toBe(true);
		expect(isPrivateOrRestrictedIp('169.254.1.1')).toBe(true);
	});

	it('allows public IP addresses', () => {
		expect(isPrivateOrRestrictedIp('8.8.8.8')).toBe(false);
		expect(isPrivateOrRestrictedIp('104.26.14.240')).toBe(false);
	});
});

describe('normalizeProductUrl', () => {
	it('strips tracking and utm parameters', () => {
		const raw =
			'https://www.amazon.in/dp/B09G9FPG54?utm_source=google&utm_medium=cpc&ref=xyz&tag=affil#details';
		const normalized = normalizeProductUrl(raw);
		expect(normalized).toBe('https://www.amazon.in/dp/B09G9FPG54');
	});
});

describe('extractProductFromUrl', () => {
	it('rejects invalid schemes or protocols', async () => {
		const result = await extractProductFromUrl('javascript:alert(1)');
		expect(result.confidence).toBe('manual_fallback');
		expect(result.error).toContain('Only HTTP and HTTPS');
	});

	it('blocks localhost destinations (SSRF defense)', async () => {
		const result = await extractProductFromUrl('http://localhost:8080/secret');
		expect(result.confidence).toBe('manual_fallback');
		expect(result.error).toContain('internal or local hosts is blocked');
	});

	it('extracts product from JSON-LD schema', async () => {
		const mockHtml = `
			<!DOCTYPE html>
			<html>
			<head>
				<script type="application/ld+json">
				{
					"@context": "https://schema.org",
					"@type": "Product",
					"name": "Sony WH-1000XM5 Wireless Headphones",
					"brand": { "@type": "Brand", "name": "Sony" },
					"image": "https://example.com/sony.jpg",
					"offers": {
						"@type": "Offer",
						"price": 29990,
						"priceCurrency": "INR"
					}
				}
				</script>
			</head>
			<body></body>
			</html>
		`;

		const mockFetch = vi.fn().mockResolvedValue({
			ok: true,
			status: 200,
			text: () => Promise.resolve(mockHtml)
		});

		const result = await extractProductFromUrl(
			'https://example.com/sony-headphones',
			mockFetch as unknown as typeof fetch
		);
		expect(result.name).toBe('Sony WH-1000XM5 Wireless Headphones');
		expect(result.brand).toBe('Sony');
		expect(result.currentPricePaise).toBe(2999000);
		expect(result.confidence).toBe('high');
	});
});
