import * as cheerio from 'cheerio';
import type { ExtractedProductData, RetailerParser } from '../types.js';

export const relianceParser: RetailerParser = {
	name: 'Reliance Digital',
	domains: ['reliancedigital.in'],

	canParse(url: URL): boolean {
		const host = url.hostname.toLowerCase().replace(/^www\./, '');
		return this.domains.some((d) => host === d || host.endsWith('.' + d));
	},

	parse(html: string, url: URL): Partial<ExtractedProductData> | null {
		const $ = cheerio.load(html);

		const title = $('.pdp__title').first().text().trim() || $('h1').first().text().trim();
		if (!title) return null;

		let currentPricePaise = 0;
		let originalPricePaise = 0;

		const priceText =
			$('.pdp__offerPrice').first().text().trim() ||
			$('.pdp__priceSection__priceList span.text').first().text().trim();
		if (priceText) {
			const clean = parseFloat(priceText.replace(/[^0-9.]/g, ''));
			if (!isNaN(clean) && clean > 0) {
				currentPricePaise = Math.round(clean * 100);
			}
		}

		const mrpText =
			$('.pdp__mrpPrice').first().text().trim() || $('.pdp__mrp').first().text().trim();
		if (mrpText) {
			const clean = parseFloat(mrpText.replace(/[^0-9.]/g, ''));
			if (!isNaN(clean) && clean > 0) {
				originalPricePaise = Math.round(clean * 100);
			}
		}

		let discountPercent = 0;
		if (originalPricePaise > currentPricePaise && currentPricePaise > 0) {
			discountPercent = Math.round(
				((originalPricePaise - currentPricePaise) / originalPricePaise) * 100
			);
		}

		const imageUrl =
			$('.pdp__main-image-container img').attr('src') || $('img.product-image').attr('src') || '';

		const specifications: Record<string, string> = {};
		$('.specifications__list tr, .pdp__tab-info tr').each((_, el) => {
			const k = $(el).find('td:first-child').text().trim();
			const v = $(el).find('td:last-child').text().trim();
			if (k && v && k.length < 50 && v.length < 200) {
				specifications[k] = v;
			}
		});

		return {
			name: title,
			category: 'Electronics',
			currentPricePaise: currentPricePaise || undefined,
			originalPricePaise: originalPricePaise || undefined,
			discountPercent: discountPercent || undefined,
			currency: 'INR',
			availability: 'in_stock',
			specifications,
			retailer: 'Reliance Digital',
			imageUrl: imageUrl || undefined,
			url: url.toString()
		};
	}
};
