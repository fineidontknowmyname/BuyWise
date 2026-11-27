export type ProductListing = {
	url: string;
	retailer: string;
	name?: string;
	brand?: string;
	variant?: string;
	category?: string;
	imageUrl?: string;
	currentPricePaise?: number;
	observedAt?: string;
	source?: string;
	manualDetailsRequired: boolean;
	warnings: string[];
};

export function fillListingField(
	current: string | number | null | undefined,
	previous: string | undefined,
	next: string | undefined
) {
	const value = current == null ? '' : String(current);
	return !value.trim() || (previous !== undefined && value === previous) ? (next ?? '') : value;
}

export function listingPrice(listing: ProductListing | null) {
	return listing?.currentPricePaise ? (listing.currentPricePaise / 100).toFixed(2) : '';
}
