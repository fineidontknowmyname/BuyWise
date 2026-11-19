import * as cheerio from 'cheerio';
import type { ExtractedProductData, RetailerParser } from '../types.js';

export const flipkartParser: RetailerParser = {
	name: 'Flipkart',
	domains: ['flipkart.com'],

	canParse(url: URL): boolean {
		const host = url.hostname.toLowerCase().replace(/^www\./, '');
		return this.domains.some((d) => host === d || host.endsWith('.' + d));
	},

	parse(html: string, url: URL): Partial<ExtractedProductData> | null {
		const $ = cheerio.load(html);

		// Title
		const title =
			$('span.B_NuCI').text().trim() ||
			$('h1._6EBuvT').text().trim() ||
			$('span.VU-ZEz').text().trim() ||
			$('h1').first().text().trim();

		if (!title && !html.includes('_30jeq3') && !html.includes('Nx9bqj')) {
			return null;
		}

		const name = title.replace(/\s+/g, ' ').trim();

		// Brand
		let brand = $('span.G6XhRU').text().trim() || '';
		if (!brand) {
			const m = name.match(
				/^(Apple|Samsung|Realme|Xiaomi|Redmi|Poco|Vivo|Oppo|Motorola|OnePlus|Sony|Dell|HP|Asus|Lenovo)/i
			);
			if (m) brand = m[1];
		}

		// Price
		let currentPricePaise = 0;
		let originalPricePaise = 0;

		const priceText =
			$('div._30jeq3._16Jk6d').first().text().trim() ||
			$('div.Nx9bqj.CxhGGd').first().text().trim() ||
			$('div._30jeq3').first().text().trim();

		if (priceText) {
			const clean = parseFloat(priceText.replace(/[^0-9.]/g, ''));
			if (!isNaN(clean) && clean > 0) {
				currentPricePaise = Math.round(clean * 100);
			}
		}

		// Original Price
		const origText =
			$('div._3I9_wc._2p6Xd0').first().text().trim() ||
			$('div.yRaY8j').first().text().trim() ||
			$('div._3I9_wc').first().text().trim();

		if (origText) {
			const clean = parseFloat(origText.replace(/[^0-9.]/g, ''));
			if (!isNaN(clean) && clean > 0) {
				originalPricePaise = Math.round(clean * 100);
			}
		}

		// Discount
		let discountPercent = 0;
		const discText = $('div._3Ay6Sb span, div.UkUFwK span')
			.first()
			.text()
			.replace(/[^0-9]/g, '');
		if (discText) {
			discountPercent = parseFloat(discText);
		} else if (originalPricePaise > currentPricePaise && currentPricePaise > 0) {
			discountPercent = Math.round(
				((originalPricePaise - currentPricePaise) / originalPricePaise) * 100
			);
		}

		// Availability
		let availability: ExtractedProductData['availability'] = 'in_stock';
		const pageText = $('body').text();
		if (
			pageText.includes('Currently Out of Stock') ||
			pageText.includes('Sold Out') ||
			$('div._16FRp0').length > 0
		) {
			availability = 'out_of_stock';
		}

		// Image
		const imageUrl =
			$('img._396cs4._3exPp9').attr('src') ||
			$('img.DByuf4').attr('src') ||
			$('img._2r_T1I').attr('src') ||
			$('div.q6DClP img').attr('src') ||
			'';

		// Specifications
		const specifications: Record<string, string> = {};
		$('table._14cfVK tr, div._3k-BhJ tr, div.GNDEk- tr').each((_, el) => {
			const key = $(el).find('td._1hKmbr, td:first-child').text().trim();
			const val = $(el).find('td._21lJbe, td:last-child').text().trim().replace(/\s+/g, ' ');
			if (key && val && key.length < 50 && val.length < 200) {
				specifications[key] = val;
			}
		});

		// Variant from selection buttons
		const variant = $('li._3V2wfe._31hAvz, div._2C41yO').text().trim();

		return {
			name: name || undefined,
			brand: brand || undefined,
			model: specifications['Model Name'] || specifications['Model Number'] || undefined,
			variant: variant || undefined,
			category: 'Shopping',
			currentPricePaise: currentPricePaise || undefined,
			originalPricePaise: originalPricePaise || undefined,
			discountPercent: discountPercent || undefined,
			currency: 'INR',
			availability,
			specifications,
			retailer: 'Flipkart',
			imageUrl: imageUrl || undefined,
			url: url.toString()
		};
	}
};
