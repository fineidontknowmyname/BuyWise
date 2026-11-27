export type DocumentPurpose = 'price_history' | 'specifications' | 'quote_or_invoice' | 'mixed';

export interface ParsedPriceRow {
	line: number;
	observedOn: string; // YYYY-MM-DD
	pricePaise: number;
	priceFormatted: string;
	currency: string;
	priceType: 'regular' | 'discounted' | 'original';
	retailer: string;
	source: string;
	notes: string;
	isValid: boolean;
	error?: string;
}

export interface DocumentProductDetails {
	name?: string;
	brand?: string;
	model?: string;
	variant?: string;
	currentPricePaise?: number;
	originalPricePaise?: number;
	currency?: string;
	specifications?: Record<string, string>;
	retailer?: string;
	url?: string;
	imageUrl?: string;
	availability?: 'in_stock' | 'out_of_stock' | 'pre_order' | 'unknown';
	notes?: string;
}

export interface ProductIdentityMatch {
	matches: boolean;
	similarityScore: number; // 0 to 100
	goalProductName: string;
	documentProductName: string;
	warning?: string;
}

export interface DocumentPreviewResult {
	id: string;
	goalId: string;
	filename: string;
	fileType: 'csv' | 'xlsx' | 'pdf' | 'docx' | 'image';
	fileSize: number;
	purpose: DocumentPurpose;
	detectedColumns: string[];
	columnMappings: {
		dateColumn?: string;
		priceColumn?: string;
		originalPriceColumn?: string;
		retailerColumn?: string;
		notesColumn?: string;
	};
	productDetails?: DocumentProductDetails;
	warnings?: string[];
	identityMatch?: ProductIdentityMatch;
	validRows: ParsedPriceRow[];
	errorRows: Array<{ line: number; raw: string; error: string }>;
	duplicateCount: number;
	summary: {
		totalRows: number;
		validCount: number;
		errorCount: number;
		detectedCurrency: string;
		dateRange?: { from: string; to: string };
	};
}
