import type { DocumentProductDetails } from './server/documents/types.js';

export const documentProductLabels: Record<keyof DocumentProductDetails, string> = {
	name: 'Product name',
	brand: 'Brand',
	model: 'Model',
	variant: 'Variant',
	currentPricePaise: 'Listed price',
	originalPricePaise: 'Original price / MRP',
	currency: 'Currency',
	retailer: 'Retailer',
	url: 'Listing URL',
	imageUrl: 'Image URL',
	availability: 'Availability',
	specifications: 'Specifications',
	notes: 'Notes and terms'
};

export function applyDocumentDetails(
	current: DocumentProductDetails,
	incoming: DocumentProductDetails,
	selected: Array<keyof DocumentProductDetails>
): DocumentProductDetails {
	let result = { ...current };
	for (const key of selected) {
		if (incoming[key] === undefined) continue;
		result = {
			...result,
			[key]:
				key === 'specifications'
					? { ...current.specifications, ...incoming.specifications }
					: incoming[key]
		};
	}
	return result;
}

export function documentIdentityConflicts(
	current: DocumentProductDetails,
	incoming: DocumentProductDetails
): string[] {
	const normalize = (value: string) => value.toLowerCase().replace(/\s+/g, ' ').trim();
	const conflicts: string[] = [];
	for (const key of ['variant', 'model'] as const) {
		if (current[key] && incoming[key] && normalize(current[key]) !== normalize(incoming[key]))
			conflicts.push(documentProductLabels[key]);
	}
	const asin = (url?: string) =>
		url?.match(/\/(?:dp|gp\/product)\/([a-z0-9]{10})(?:[/?#]|$)/i)?.[1]?.toUpperCase();
	const savedAsin = asin(current.url),
		incomingAsin = asin(incoming.url);
	if (savedAsin && incomingAsin && savedAsin !== incomingAsin) conflicts.push('Listing identifier');
	return conflicts;
}
