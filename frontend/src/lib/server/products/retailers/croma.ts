import * as cheerio from 'cheerio';
import type { ExtractedProductData, RetailerParser } from '../types.js';

export const cromaParser: RetailerParser = {
	name: 'Croma',
	domains: ['croma.com'],

	canParse(url: URL): boolean {
		const host = url.hostname.toLowerCase().replace(/^www\./, '');
		return this.domains.some((d) => host === d || host.endsWith('.' + d));
	},

	parse(html: string, url: URL): Partial<ExtractedProductData> | null {
		const $ = cheerio.load(html);

		const title =
			$('.pd-title').first().text().trim() ||
			$('h1.pdp-title').first().text().trim() ||
			$('h1').first().text().trim();

		if (!title) return null;

		let currentPricePaise = 0;
		let originalPricePaise = 0;

		const priceText =
			$('.amount').first().text().trim() ||
			$('#pdp-product-price').text().trim() ||
			$('.pdp-cp-price').text().trim();

		if (priceText) {
			const clean = parseFloat(priceText.replace(/[^0-9.]/g, ''));
			if (!isNaN(clean) && clean > 0) {
				currentPricePaise = Math.round(clean * 100);
			}
		}

		const oldPriceText =
			$('.old-price').first().text().trim() || $('.pdp-mrp-price').first().text().trim();
		if (oldPriceText) {
			const clean = parseFloat(oldPriceText.replace(/[^0-9.]/g, ''));
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
			$('.product-image img').attr('src') ||
			$('#product-image-0').attr('src') ||
			$('.main-img-slider img').attr('src') ||
			'';

		const specifications: Record<string, string> = {};
		$('.cp-specification tr, .spec-body tr').each((_, el) => {
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
			retailer: 'Croma',
			imageUrl: imageUrl || undefined,
			url: url.toString()
		};
	}
};
