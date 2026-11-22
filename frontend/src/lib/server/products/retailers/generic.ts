import * as cheerio from 'cheerio';
import type { ExtractedProductData, RetailerParser } from '../types.js';

export const genericParser: RetailerParser = {
	name: 'Generic',
	domains: [],

	canParse(): boolean {
		return true;
	},

	parse(html: string, url: URL): Partial<ExtractedProductData> | null {
		const $ = cheerio.load(html);

		// 1. Title
		const title =
			$('meta[property="og:title"]').attr('content') ||
			$('meta[name="twitter:title"]').attr('content') ||
			$('h1').first().text().trim() ||
			$('title').text().trim();

		if (!title) return null;

		const cleanTitle = title.replace(/\s*[-|–—]\s*[^|–—]+$/i, '').trim();
		const hostname = url.hostname.replace(/^www\./, '');
		const retailer = hostname.split('.')[0];
		const retailerName = retailer.charAt(0).toUpperCase() + retailer.slice(1);

		// 2. Image
		const imageUrl =
			$('meta[property="og:image"]').attr('content') ||
			$('meta[name="twitter:image"]').attr('content') ||
			$('link[rel="image_src"]').attr('href') ||
			'';

		// 3. Description
		const description =
			$('meta[property="og:description"]').attr('content') ||
			$('meta[name="description"]').attr('content') ||
			'';

		// 4. Price
		let currentPricePaise = 0;
		let currency = 'INR';

		const ogPrice =
			$('meta[property="og:price:amount"]').attr('content') ||
			$('meta[property="product:price:amount"]').attr('content') ||
			$('[itemprop="price"]').attr('content') ||
			$('[itemprop="price"]').text().trim();

		const ogCurrency =
			$('meta[property="og:price:currency"]').attr('content') ||
			$('meta[property="product:price:currency"]').attr('content') ||
			$('[itemprop="priceCurrency"]').attr('content');

		if (ogCurrency) currency = ogCurrency.toUpperCase();

		if (ogPrice) {
			const clean = parseFloat(ogPrice.replace(/[^0-9.]/g, ''));
			if (!isNaN(clean) && clean > 0) {
				currentPricePaise = Math.round(clean * 100);
			}
		}

		// 5. Brand
		const brand =
			$('meta[property="og:brand"]').attr('content') ||
			$('meta[property="product:brand"]').attr('content') ||
			$('[itemprop="brand"]').text().trim() ||
			'';

		// 6. Availability
		let availability: ExtractedProductData['availability'] = 'in_stock';
		const avail =
			$('meta[property="og:availability"]').attr('content') ||
			$('meta[property="product:availability"]').attr('content') ||
			$('[itemprop="availability"]').attr('content') ||
			'';

		if (
			avail.toLowerCase().includes('outofstock') ||
			avail.toLowerCase().includes('out_of_stock')
		) {
			availability = 'out_of_stock';
		}

		return {
			name: cleanTitle,
			brand: brand || undefined,
			category: 'Shopping',
			description: description.slice(0, 1500) || undefined,
			currentPricePaise: currentPricePaise || undefined,
			currency,
			availability,
			retailer: retailerName,
			imageUrl: imageUrl || undefined,
			url: url.toString()
		};
	}
};
