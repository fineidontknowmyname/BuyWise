import * as cheerio from 'cheerio';
import type { ExtractedProductData, RetailerParser } from '../types.js';

export const appleParser: RetailerParser = {
	name: 'Apple',
	domains: ['apple.com'],

	canParse(url: URL): boolean {
		const host = url.hostname.toLowerCase().replace(/^www\./, '');
		return this.domains.some((d) => host === d || host.endsWith('.' + d));
	},

	parse(html: string, url: URL): Partial<ExtractedProductData> | null {
		const $ = cheerio.load(html);

		const title =
			$('h1.rf-flagship-hero-eyebrow').text().trim() ||
			$('h1.typography-headline').first().text().trim() ||
			$('h1.product-title').first().text().trim() ||
			$('title')
				.text()
				.replace(/ - Apple.*/i, '')
				.trim();

		if (!title) return null;

		let currentPricePaise = 0;
		const currency = url.pathname.includes('/in/') ? 'INR' : 'USD';

		const priceText =
			$('.rc-prices-fullprice').first().text().trim() ||
			$('.rc-prices-currentprice').first().text().trim() ||
			$('[data-autom="full-price"]').first().text().trim();

		if (priceText) {
			const clean = parseFloat(priceText.replace(/[^0-9.]/g, ''));
			if (!isNaN(clean) && clean > 0) {
				currentPricePaise = Math.round(clean * 100);
			}
		}

		const imageUrl =
			$('img.as-hero-image').attr('src') || $('img.rf-flagship-hero-image').attr('src') || '';

		const specifications: Record<string, string> = {};
		$('table.tech-specs-table tr, .tech-specs tr').each((_, el) => {
			const k = $(el).find('th, td:first-child').text().trim();
			const v = $(el).find('td:last-child').text().trim();
			if (k && v && k.length < 50 && v.length < 200) {
				specifications[k] = v;
			}
		});

		return {
			name: title,
			brand: 'Apple',
			category: 'Electronics',
			currentPricePaise: currentPricePaise || undefined,
			currency,
			availability: 'in_stock',
			specifications,
			retailer: 'Apple',
			imageUrl: imageUrl || undefined,
			url: url.toString()
		};
	}
};
