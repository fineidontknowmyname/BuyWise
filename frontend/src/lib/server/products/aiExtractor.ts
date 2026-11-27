import { callGroq } from '../ai/groqClient.js';
import type { ExtractedProductData } from './types.js';

export interface AIProductInterpretation {
	name: string;
	brand: string;
	model: string;
	variant: string;
	currentPriceRupees: number;
	originalPriceRupees: number;
	discountPercent: number;
	currency: string;
	availability: 'in_stock' | 'out_of_stock' | 'pre_order' | 'unknown';
	specifications: Record<string, string>;
	retailer: string;
	isEmiOrAccessoryPrice: boolean;
	confidence: 'high' | 'medium' | 'low';
}

export async function interpretProductWithAI(
	pageSnippet: string,
	url: URL,
	fetchFn: typeof fetch = fetch
): Promise<Partial<ExtractedProductData> | null> {
	// Minimize context to essentials (max 4000 characters)
	const cleanSnippet = pageSnippet
		.replace(/<script\b[^<]*(?:(?!<\/script>)<[^<]*)*<\/script>/gi, ' ')
		.replace(/<style\b[^<]*(?:(?!<\/style>)<[^<]*)*<\/style>/gi, ' ')
		.replace(/<svg\b[^<]*(?:(?!<\/svg>)<[^<]*)*<\/svg>/gi, ' ')
		.replace(/\s+/g, ' ')
		.slice(0, 4000);

	const prompt = `You are an expert product information extractor. Analyze the following webpage text from retailer "${url.hostname}" and extract structured product details.
CRITICAL RULES:
1. Distinguish between actual full product purchase price and EMI amounts (e.g. "₹2,500/month", "EMI starts at"). NEVER select an EMI installment or accessory price as the current product price.
2. If original crossed-out price (MRP) is shown, extract it as originalPriceRupees.
3. Extract specific technical specifications (specs table, key features) as a clean key-value dictionary.
4. Extract variant (storage, color, size, configuration) if specified.
5. If price or details are genuinely absent or ambiguous, set currentPriceRupees to 0. Do NOT invent prices.

Output strictly valid JSON with this exact schema:
{
  "name": "string",
  "brand": "string",
  "model": "string",
  "variant": "string",
  "currentPriceRupees": 0,
  "originalPriceRupees": 0,
  "discountPercent": 0,
  "currency": "INR",
  "availability": "in_stock",
  "specifications": { "key": "value" },
  "retailer": "string",
  "isEmiOrAccessoryPrice": false,
  "confidence": "high"
}

Webpage text:
"""${cleanSnippet}"""`;

	try {
		const res = await callGroq(
			[
				{
					role: 'system',
					content:
						'You are a precise, deterministic e-commerce data extraction assistant. Output strict JSON only.'
				},
				{
					role: 'user',
					content: prompt
				}
			],
			{
				fetchFn,
				jsonMode: true,
				temperature: 0.1,
				maxTokens: 800
			}
		);

		const content = res.data?.choices?.[0]?.message?.content?.trim();
		if (!content) return null;

		const parsed: AIProductInterpretation = JSON.parse(content);

		// Deterministic sanity check: verify price appears in page text
		let verifiedPricePaise = 0;
		if (parsed.currentPriceRupees > 0 && !parsed.isEmiOrAccessoryPrice) {
			const priceStr = Math.round(parsed.currentPriceRupees).toString();
			// Check if this numeric sequence appears in the cleaned snippet
			const priceFormatted = new Intl.NumberFormat('en-IN').format(parsed.currentPriceRupees);
			if (
				cleanSnippet.includes(priceStr) ||
				cleanSnippet.includes(priceFormatted) ||
				cleanSnippet.includes(parsed.currentPriceRupees.toString())
			) {
				verifiedPricePaise = Math.round(parsed.currentPriceRupees * 100);
			}
		}

		let origPricePaise = 0;
		if (parsed.originalPriceRupees > 0) {
			origPricePaise = Math.round(parsed.originalPriceRupees * 100);
		}

		let discountPercent = parsed.discountPercent || 0;
		if (origPricePaise > verifiedPricePaise && verifiedPricePaise > 0) {
			discountPercent = Math.round(((origPricePaise - verifiedPricePaise) / origPricePaise) * 100);
		}

		const specs: Record<string, string> = {};
		if (parsed.specifications && typeof parsed.specifications === 'object') {
			for (const [k, v] of Object.entries(parsed.specifications)) {
				if (typeof v === 'string' && k.length < 50 && v.length < 200) {
					specs[k] = v;
				}
			}
		}

		return {
			name: parsed.name?.trim() || undefined,
			brand: parsed.brand?.trim() || undefined,
			model: parsed.model?.trim() || undefined,
			variant: parsed.variant?.trim() || undefined,
			currentPricePaise: verifiedPricePaise || undefined,
			originalPricePaise: origPricePaise || undefined,
			discountPercent: discountPercent || undefined,
			currency: parsed.currency || 'INR',
			availability: parsed.availability || 'in_stock',
			specifications: specs,
			retailer: parsed.retailer || undefined,
			stageReached: 'ai_interpretation'
		};
	} catch {
		return null;
	}
}
