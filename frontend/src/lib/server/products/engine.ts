import { normalizeProductUrl, validateUrlSecurity } from './ssrf.js';
import { findRetailerParser } from './retailers/index.js';
import { retrieveRenderedHtml } from './browserWorker.js';
import { interpretProductWithAI } from './aiExtractor.js';
import type { ExtractedProductData, ExtractionConfidence } from './types.js';

interface CacheEntry {
	data: ExtractedProductData;
	expiresAt: number;
}

const urlCache = new Map<string, CacheEntry>();
const inFlightRequests = new Map<string, Promise<ExtractedProductData>>();
const CACHE_TTL_MS = 60 * 60 * 1000; // 1 hour

export async function extractProductFromUrl(
	rawUrl: string,
	fetchFn: typeof fetch = fetch
): Promise<ExtractedProductData> {
	const normalizedUrl = normalizeProductUrl(rawUrl);

	let validUrl: URL;
	try {
		validUrl = new URL(normalizedUrl);
	} catch {
		return createFallbackResult(rawUrl, 'Invalid URL format.', 'manual_fallback');
	}

	// 1. Cache lookup
	const cached = urlCache.get(normalizedUrl);
	if (cached && cached.expiresAt > Date.now()) {
		return cached.data;
	}

	// 2. In-flight de-duplication
	const existingPromise = inFlightRequests.get(normalizedUrl);
	if (existingPromise) {
		return existingPromise;
	}

	const extractionPromise = (async (): Promise<ExtractedProductData> => {
		const hostname = validUrl.hostname.replace(/^www\./, '');
		const defaultRetailer = capitalize(hostname.split('.')[0] || 'Store');

		// Stage 1: Deterministic URL validation & SSRF defense
		const security = await validateUrlSecurity(validUrl);
		if (!security.safe) {
			return createFallbackResult(
				validUrl.toString(),
				security.error || 'Blocked by security policy.',
				'manual_fallback',
				defaultRetailer
			);
		}

		// Stage 2: Lightweight HTTP retrieval
		let html: string;
		let stageReached: ExtractedProductData['stageReached'] = 'structured_metadata';

		try {
			const res = await fetchFn(validUrl.toString(), {
				headers: {
					'User-Agent':
						'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36',
					Accept: 'text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8',
					'Accept-Language': 'en-US,en;q=0.9'
				},
				redirect: 'error',
				signal: AbortSignal.timeout(6000)
			});

			if (!res.ok) {
				return createFallbackResult(
					validUrl.toString(),
					`Retailer returned HTTP ${res.status}.`,
					'manual_fallback',
					defaultRetailer
				);
			}

			// Read up to 1MB
			const text = await res.text();
			html = text.slice(0, 1048576);
		} catch (err: unknown) {
			const msg = err instanceof Error ? err.message : 'Network retrieval failed.';
			return createFallbackResult(validUrl.toString(), msg, 'manual_fallback', defaultRetailer);
		}

		// Stage 3: Structured Metadata Extraction (JSON-LD & OpenGraph)
		const structuredData = parseStructuredMetadata(html, validUrl);

		// Stage 4: Retailer-Specific Parser
		const retailerParser = findRetailerParser(validUrl);
		const retailerData = retailerParser.parse(html, validUrl);

		// Merge structured and retailer-specific results
		let merged: ExtractedProductData = mergeProductResults(
			validUrl,
			defaultRetailer,
			structuredData,
			retailerData
		);

		if (merged.currentPricePaise > 0 && merged.name) {
			stageReached = 'retailer_parser';
		}

		// Stage 5: Browser-assisted retrieval if page is an empty single-page app shell
		const isClientSideOnly =
			html.length < 5000 &&
			(html.includes('id="root"') || html.includes('id="__next"') || html.includes('id="app"')) &&
			(!merged.name || merged.currentPricePaise === 0);

		if (isClientSideOnly) {
			const renderedHtml = await retrieveRenderedHtml(validUrl);
			if (renderedHtml && renderedHtml.length > html.length) {
				stageReached = 'browser_assisted';
				const renderedStructured = parseStructuredMetadata(renderedHtml, validUrl);
				const renderedRetailer = retailerParser.parse(renderedHtml, validUrl);
				merged = mergeProductResults(
					validUrl,
					defaultRetailer,
					merged,
					renderedStructured,
					renderedRetailer
				);
				html = renderedHtml;
			}
		}

		// Stage 6: AI Interpretation using Groq if price or key details are still missing
		if (
			(!merged.currentPricePaise ||
				!merged.name ||
				Object.keys(merged.specifications).length === 0) &&
			html.length > 500
		) {
			const aiData = await interpretProductWithAI(html, validUrl, fetchFn);
			if (aiData) {
				stageReached = 'ai_interpretation';
				merged = mergeProductResults(validUrl, defaultRetailer, merged, aiData);
			}
		}

		// Stage 7: Deterministic Schema Validation & Sanity Checking
		merged.stageReached = stageReached;
		const unverifiedFields: string[] = [];

		if (!merged.currentPricePaise || merged.currentPricePaise <= 0) {
			unverifiedFields.push('currentPricePaise');
		}
		if (!merged.name) {
			unverifiedFields.push('name');
			merged.name = `Product from ${merged.retailer}`;
		}

		merged.unverifiedFields = unverifiedFields;

		// Stage 8: Recovery & Confidence Assignment
		let confidence: ExtractionConfidence;
		if (merged.currentPricePaise > 0 && merged.name && !unverifiedFields.length) {
			confidence =
				stageReached === 'structured_metadata' || stageReached === 'retailer_parser'
					? 'high'
					: 'medium';
		} else if (merged.name && unverifiedFields.length > 0) {
			confidence = 'partial';
		} else {
			confidence = 'manual_fallback';
		}

		merged.confidence = confidence;

		// Cache successful or partial results
		if (confidence !== 'manual_fallback') {
			urlCache.set(normalizedUrl, {
				data: merged,
				expiresAt: Date.now() + CACHE_TTL_MS
			});
		}

		return merged;
	})();

	inFlightRequests.set(normalizedUrl, extractionPromise);
	try {
		return await extractionPromise;
	} finally {
		inFlightRequests.delete(normalizedUrl);
	}
}

function parseStructuredMetadata(html: string, url: URL): Partial<ExtractedProductData> {
	let name = '';
	let brand = '';
	let imageUrl = '';
	let pricePaise = 0;
	const origPricePaise = 0;
	let currency = 'INR';
	let availability: ExtractedProductData['availability'] = 'in_stock';
	const specifications: Record<string, string> = {};

	// A. JSON-LD parsing
	const jsonLdMatches = html.match(
		/<script[^>]+type=["']application\/ld\+json["'][^>]*>([\s\S]*?)<\/script>/gi
	);
	if (jsonLdMatches) {
		for (const tag of jsonLdMatches) {
			try {
				const jsonText = tag.replace(/<script[^>]*>|<\/script>/gi, '').trim();
				const parsed = JSON.parse(jsonText);
				const items = Array.isArray(parsed)
					? parsed
					: parsed['@graph']
						? parsed['@graph']
						: [parsed];

				for (const item of items) {
					if (item && (item['@type'] === 'Product' || item['@type'] === 'ProductModel')) {
						if (item.name && !name) name = String(item.name).trim();
						if (item.image) {
							imageUrl = Array.isArray(item.image)
								? String(item.image[0])
								: typeof item.image === 'object'
									? String(item.image.url || '')
									: String(item.image);
						}
						if (item.brand) {
							brand =
								typeof item.brand === 'object' ? String(item.brand.name || '') : String(item.brand);
						}

						const offers = item.offers;
						if (offers) {
							const offer = Array.isArray(offers) ? offers[0] : offers;
							if (offer && offer.price !== undefined) {
								const clean = parseFloat(String(offer.price).replace(/[^0-9.]/g, ''));
								if (!isNaN(clean) && clean > 0) {
									pricePaise = Math.round(clean * 100);
									if (offer.priceCurrency) currency = offer.priceCurrency;
								}
							}
							if (offer && offer.availability) {
								const av = String(offer.availability).toLowerCase();
								if (av.includes('outofstock')) availability = 'out_of_stock';
								else if (av.includes('preorder')) availability = 'pre_order';
							}
						}
					}
				}
			} catch {
				// skip malformed JSON-LD
			}
		}
	}

	// B. OpenGraph fallbacks
	if (!name) {
		const m = html.match(/<meta\s+property=["']og:title["']\s+content=["'](.*?)["']/i);
		if (m) name = decodeHtml(m[1].trim());
	}
	if (!imageUrl) {
		const m = html.match(/<meta\s+property=["']og:image["']\s+content=["'](.*?)["']/i);
		if (m) imageUrl = m[1].trim();
	}
	if (pricePaise === 0) {
		const m = html.match(
			/<meta\s+property=["'](?:og|product):price:amount["']\s+content=["'](.*?)["']/i
		);
		if (m) {
			const clean = parseFloat(m[1].replace(/[^0-9.]/g, ''));
			if (!isNaN(clean) && clean > 0) pricePaise = Math.round(clean * 100);
		}
	}

	return {
		name: name || undefined,
		brand: brand || undefined,
		currentPricePaise: pricePaise || undefined,
		originalPricePaise: origPricePaise || undefined,
		currency,
		availability,
		imageUrl: imageUrl || undefined,
		specifications,
		url: url.toString()
	};
}

function mergeProductResults(
	url: URL,
	defaultRetailer: string,
	...sources: Array<Partial<ExtractedProductData> | null | undefined>
): ExtractedProductData {
	const result: ExtractedProductData = {
		name: '',
		brand: '',
		model: '',
		variant: '',
		category: 'Shopping',
		description: '',
		currentPricePaise: 0,
		originalPricePaise: 0,
		discountPercent: 0,
		currency: 'INR',
		availability: 'in_stock',
		specifications: {},
		retailer: defaultRetailer,
		imageUrl: '',
		url: url.toString(),
		confidence: 'unverified',
		stageReached: 'structured_metadata'
	};

	for (const s of sources) {
		if (!s) continue;
		if (s.name && !result.name) result.name = s.name;
		if (s.brand && !result.brand) result.brand = s.brand;
		if (s.model && !result.model) result.model = s.model;
		if (s.variant && !result.variant) result.variant = s.variant;
		if (s.category && result.category === 'Shopping') result.category = s.category;
		if (s.description && !result.description) result.description = s.description;
		if (s.currentPricePaise && s.currentPricePaise > 0)
			result.currentPricePaise = s.currentPricePaise;
		if (s.originalPricePaise && s.originalPricePaise > 0)
			result.originalPricePaise = s.originalPricePaise;
		if (s.discountPercent && s.discountPercent > 0) result.discountPercent = s.discountPercent;
		if (s.currency) result.currency = s.currency;
		if (s.availability && s.availability !== 'unknown') result.availability = s.availability;
		if (s.retailer && result.retailer === defaultRetailer) result.retailer = s.retailer;
		if (s.imageUrl && !result.imageUrl) result.imageUrl = s.imageUrl;
		if (s.specifications) {
			result.specifications = { ...result.specifications, ...s.specifications };
		}
	}

	return result;
}

function createFallbackResult(
	url: string,
	errorMessage: string,
	confidence: ExtractionConfidence = 'manual_fallback',
	retailer: string = 'Store'
): ExtractedProductData {
	return {
		name: '',
		brand: '',
		model: '',
		variant: '',
		category: 'Shopping',
		description: '',
		currentPricePaise: 0,
		originalPricePaise: 0,
		discountPercent: 0,
		currency: 'INR',
		availability: 'unknown',
		specifications: {},
		retailer,
		imageUrl: '',
		url,
		confidence,
		stageReached: 'manual_fallback',
		error: `${errorMessage} You can enter product information manually.`
	};
}

function capitalize(s: string): string {
	if (!s) return '';
	return s.charAt(0).toUpperCase() + s.slice(1);
}

function decodeHtml(html: string): string {
	return html
		.replace(/&amp;/g, '&')
		.replace(/&lt;/g, '<')
		.replace(/&gt;/g, '>')
		.replace(/&quot;/g, '"')
		.replace(/&#39;/g, "'");
}
