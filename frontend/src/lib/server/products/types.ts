export type ExtractionConfidence = 'high' | 'medium' | 'partial' | 'unverified' | 'manual_fallback';

export interface ExtractedProductData {
	name: string;
	brand: string;
	model: string;
	variant: string;
	category: string;
	description: string;
	currentPricePaise: number;
	originalPricePaise: number;
	discountPercent: number;
	currency: string;
	availability: 'in_stock' | 'out_of_stock' | 'pre_order' | 'unknown';
	specifications: Record<string, string>;
	retailer: string;
	imageUrl: string;
	url: string;
	confidence: ExtractionConfidence;
	stageReached:
		| 'structured_metadata'
		| 'retailer_parser'
		| 'browser_assisted'
		| 'ai_interpretation'
		| 'manual_fallback';
	unverifiedFields?: string[];
	warnings?: string[];
	error?: string;
}

export interface RetailerParser {
	name: string;
	domains: string[];
	canParse(url: URL): boolean;
	parse(html: string, url: URL): Partial<ExtractedProductData> | null;
}
