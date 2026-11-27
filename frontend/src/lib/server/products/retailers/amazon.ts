import * as cheerio from 'cheerio';
import type { ExtractedProductData, RetailerParser } from '../types.js';

export const amazonParser: RetailerParser = {
	name: 'Amazon',
	domains: ['amazon.in', 'amazon.com', 'amazon.co.uk', 'amazon.ca', 'amazon.de', 'amazon.fr'],

	canParse(url: URL): boolean {
		const host = url.hostname.toLowerCase().replace(/^www\./, '');
		return this.domains.some((d) => host === d || host.endsWith('.' + d));
	},

	parse(html: string, url: URL): Partial<ExtractedProductData> | null {
		const $ = cheerio.load(html);

		// Title
		const title =
			$('#productTitle').text().trim() ||
			$('#title').text().trim() ||
			$('h1.product-title-word-break').text().trim();

		if (!title && !html.includes('a-price')) {
			return null;
		}

		// Clean up title
		const name = title.replace(/\s+/g, ' ').trim();

		// Brand
		let brand =
			$('#bylineInfo')
				.text()
				.replace(/Visit the\s+|Brand:\s+|Store/gi, '')
				.trim() || $('.po-brand .a-span9').text().trim();
		if (!brand) {
			const brandMatch = name.match(
				/^(Apple|Sony|Samsung|Dell|HP|Lenovo|Asus|Boat|Noise|OnePlus|Google|Logitech|Bose)/i
			);
			if (brandMatch) brand = brandMatch[1];
		}

		// Model
		const model =
			$('.po-model_name .a-span9').text().trim() ||
			$('.po-item_model_number .a-span9').text().trim() ||
			'';

		// Variant (Color, Size, Style)
		const variantParts: string[] = [];
		const color = $('#variation_color_name .selection').text().trim();
		const size =
			$('#variation_size_name .selection').text().trim() ||
			$('#variation_style_name .selection').text().trim();
		const config =
			$('#variation_pattern_name .selection').text().trim() ||
			$('#variation_size_per_pack .selection').text().trim();
		if (color) variantParts.push(color);
		if (size) variantParts.push(size);
		if (config) variantParts.push(config);
		const variant = variantParts.join(' / ');

		// Price (Filter out EMI amounts)
		let currentPricePaise = 0;
		let originalPricePaise = 0;
		const currency = url.hostname.includes('.in') ? 'INR' : 'USD';

		// Primary price selector: .priceToPay or #corePrice_desktop
		const priceWhole =
			$('.priceToPay .a-price-whole').first().text().trim() ||
			$('#corePriceDisplay_desktop_feature_div .a-price-whole').first().text().trim() ||
			$('.apexPriceToPay .a-offscreen').first().text().trim() ||
			$('#priceblock_ourprice').text().trim() ||
			$('#priceblock_dealprice').text().trim();

		if (priceWhole) {
			const cleanPrice = parseFloat(priceWhole.replace(/[^0-9.]/g, ''));
			if (!isNaN(cleanPrice) && cleanPrice > 0) {
				currentPricePaise = Math.round(cleanPrice * 100);
			}
		}

		// List / Original Price (Strikethrough)
		const listPriceText =
			$('.basisPrice .a-offscreen').first().text().trim() ||
			$('#corePriceDisplay_desktop_feature_div .a-text-price .a-offscreen').first().text().trim() ||
			$('#listPrice').text().trim();

		if (listPriceText) {
			const cleanList = parseFloat(listPriceText.replace(/[^0-9.]/g, ''));
			if (!isNaN(cleanList) && cleanList > 0) {
				originalPricePaise = Math.round(cleanList * 100);
			}
		}

		// Discount calculation
		let discountPercent = 0;
		const savingsText = $('.savingsPercentage')
			.first()
			.text()
			.replace(/[^0-9]/g, '');
		if (savingsText) {
			discountPercent = parseFloat(savingsText);
		} else if (originalPricePaise > currentPricePaise && currentPricePaise > 0) {
			discountPercent = Math.round(
				((originalPricePaise - currentPricePaise) / originalPricePaise) * 100
			);
		}

		// Availability
		let availability: ExtractedProductData['availability'] = 'in_stock';
		const availText = $('#availability').text().toLowerCase();
		if (availText.includes('currently unavailable') || availText.includes('out of stock')) {
			availability = 'out_of_stock';
		} else if (availText.includes('pre-order') || availText.includes('available for pre-order')) {
			availability = 'pre_order';
		}

		// Image
		const imageUrl =
			$('#landingImage').attr('data-old-hires') ||
			$('#landingImage').attr('src') ||
			$('#imgBlkFront').attr('src') ||
			$('#main-image').attr('src') ||
			'';

		// Specifications Table
		const specifications: Record<string, string> = {};
		$('#productDetails_techSpec_section_1 tr, #prodDetails tr, .po-row').each((_, el) => {
			const label = $(el).find('th, .po-label, .a-span3').text().trim().replace(/[:\n]/g, '');
			const val = $(el).find('td, .po-value, .a-span9').text().trim().replace(/\s+/g, ' ');
			if (label && val && label.length < 50 && val.length < 200) {
				specifications[label] = val;
			}
		});

		// Description
		const description =
			$('#feature-bullets').text().replace(/\s+/g, ' ').trim() ||
			$('#productDescription').text().replace(/\s+/g, ' ').trim();

		return {
			name: name || undefined,
			brand: brand || undefined,
			model: model || undefined,
			variant: variant || undefined,
			category: 'Shopping',
			description: description.slice(0, 1500) || undefined,
			currentPricePaise: currentPricePaise || undefined,
			originalPricePaise: originalPricePaise || undefined,
			discountPercent: discountPercent || undefined,
			currency,
			availability,
			specifications,
			retailer: 'Amazon',
			imageUrl: imageUrl || undefined,
			url: url.toString()
		};
	}
};
