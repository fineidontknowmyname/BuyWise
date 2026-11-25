import type { RetailerParser } from '../types.js';
import { amazonParser } from './amazon.js';
import { flipkartParser } from './flipkart.js';
import { cromaParser } from './croma.js';
import { relianceParser } from './reliance.js';
import { appleParser } from './apple.js';
import { genericParser } from './generic.js';

export const retailerParsers: RetailerParser[] = [
	amazonParser,
	flipkartParser,
	cromaParser,
	relianceParser,
	appleParser,
	genericParser
];

export function findRetailerParser(url: URL): RetailerParser {
	for (const parser of retailerParsers) {
		if (parser.canParse(url)) {
			return parser;
		}
	}
	return genericParser;
}
