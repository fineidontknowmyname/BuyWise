import { applyDocumentDetails, documentProductLabels } from '#lib/documentProduct.js';
import { maxPaise } from '#lib/money.js';
import type { Product } from '../api.js';
import type { DocumentProductDetails } from './types.js';

export function parseReviewedDetails(raw: unknown): DocumentProductDetails {
	if (!raw || typeof raw !== 'object' || Array.isArray(raw))
		throw new Error('Review the detected product fields before saving.');
	const result: Record<string, unknown> = {};
	for (const [key, value] of Object.entries(raw)) {
		if (!Object.hasOwn(documentProductLabels, key)) throw new Error('Unsupported product field.');
		if (key.endsWith('Paise')) {
			if (!Number.isSafeInteger(value) || (value as number) < 0 || (value as number) > maxPaise)
				throw new Error('Enter a valid price with up to two decimal places.');
		} else if (key === 'specifications') {
			if (
				!value ||
				typeof value !== 'object' ||
				Array.isArray(value) ||
				Object.keys(value).length > 60 ||
				Object.entries(value).some(
					([name, entry]) =>
						['__proto__', 'constructor', 'prototype'].includes(name) ||
						name.length > 100 ||
						typeof entry !== 'string' ||
						entry.length > 1000
				)
			)
				throw new Error('Specifications must contain named text values.');
		} else {
			if (key === 'name' && (typeof value !== 'string' || !value.trim()))
				throw new Error('Product name is required.');
			if (
				typeof value !== 'string' ||
				value.length >
					(key === 'notes'
						? 10000
						: key.endsWith('Url') || key === 'url'
							? 2000
							: key === 'name'
								? 200
								: 500)
			)
				throw new Error('A product field is invalid or too long.');
			if ((key === 'url' || key === 'imageUrl') && value) {
				let url: URL;
				try {
					url = new URL(value);
				} catch {
					throw new Error('Use a valid HTTPS product or image URL.');
				}
				if (url.protocol !== 'https:' || url.username || url.password)
					throw new Error('Use a valid HTTPS product or image URL.');
			}
			if (key === 'currency' && !/^[A-Z]{3}$/.test(value))
				throw new Error('Use a three-letter currency code.');
			if (
				key === 'availability' &&
				!['unknown', 'in_stock', 'out_of_stock', 'pre_order'].includes(value)
			)
				throw new Error('Select a valid availability status.');
		}
		result[key] = value;
	}
	return result as DocumentProductDetails;
}

export function reviewedProductPayload(product: Product, incoming: DocumentProductDetails) {
	const current: DocumentProductDetails = {
		...product,
		availability: product.availability as DocumentProductDetails['availability']
	};
	const merged = applyDocumentDetails(
		current,
		incoming,
		Object.keys(incoming) as Array<keyof DocumentProductDetails>
	);
	return {
		name: merged.name,
		brand: merged.brand,
		model: merged.model,
		variant: merged.variant,
		category: product.category,
		description: product.description,
		currentPricePaise: merged.currentPricePaise,
		originalPricePaise: merged.originalPricePaise ?? 0,
		targetPricePaise: product.targetPricePaise,
		discountPercent:
			(merged.originalPricePaise ?? 0) > (merged.currentPricePaise ?? 0)
				? ((merged.originalPricePaise! - merged.currentPricePaise!) / merged.originalPricePaise!) *
					100
				: 0,
		currency: merged.currency,
		availability: merged.availability,
		specifications: merged.specifications,
		url: merged.url,
		imageUrl: merged.imageUrl,
		retailer: merged.retailer,
		notes: merged.notes,
		lastRetrievedAt: product.lastRetrievedAt,
		warnings: product.warnings,
		source: 'manual',
		verificationStatus: 'manual_entry'
	};
}
