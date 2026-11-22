import { describe, it, expect, vi } from 'vitest';
import { extractProductFromUrl } from './engine.js';
import { amazonParser } from './retailers/amazon.js';
import { flipkartParser } from './retailers/flipkart.js';
import { cromaParser } from './retailers/croma.js';
import { relianceParser } from './retailers/reliance.js';
import { appleParser } from './retailers/apple.js';

describe('Modular Retailer Parsers', () => {
	it('Amazon parser extracts product name, price, variant, and specs while ignoring EMI', () => {
		const html = `
			<html>
				<body>
					<span id="productTitle">Sony WH-1000XM5 Wireless Noise Cancelling Headphones</span>
					<div id="bylineInfo">Visit the Sony Store</div>
					<div id="variation_color_name"><span class="selection">Silver</span></div>
					<div id="variation_size_name"><span class="selection">Standard</span></div>
					<div class="priceToPay"><span class="a-price-whole">29,990</span></div>
					<div class="basisPrice"><span class="a-offscreen">₹34,990</span></div>
					<span class="savingsPercentage">-14%</span>
					<div id="inemi_feature_div"><span>EMI starts at ₹1,454/month</span></div>
					<div id="availability"><span>In stock</span></div>
					<table id="productDetails_techSpec_section_1">
						<tr><th>Battery Life</th><td>30 Hours</td></tr>
						<tr><th>Weight</th><td>250g</td></tr>
					</table>
					<img id="landingImage" src="https://m.media-amazon.com/images/I/sony.jpg" />
				</body>
			</html>
		`;
		const res = amazonParser.parse(html, new URL('https://www.amazon.in/dp/B09XS7JWHH'));
		expect(res).not.toBeNull();
		expect(res?.name).toContain('Sony WH-1000XM5');
		expect(res?.brand).toBe('Sony');
		expect(res?.variant).toBe('Silver / Standard');
		expect(res?.currentPricePaise).toBe(2999000); // ₹29,990 in paise, NOT ₹1,454
		expect(res?.originalPricePaise).toBe(3499000);
		expect(res?.discountPercent).toBe(14);
		expect(res?.specifications?.['Battery Life']).toBe('30 Hours');
		expect(res?.specifications?.['Weight']).toBe('250g');
		expect(res?.availability).toBe('in_stock');
	});

	it('Flipkart parser extracts name, price, original price, discount, and specs', () => {
		const html = `
			<html>
				<body>
					<span class="B_NuCI">Apple iPhone 15 (Blue, 128 GB)</span>
					<div class="_30jeq3 _16Jk6d">₹69,999</div>
					<div class="_3I9_wc _2p6Xd0">₹79,900</div>
					<div class="_3Ay6Sb _31daqi"><span>12% off</span></div>
					<table class="_14cfVK">
						<tr><td class="_1hKmbr">Internal Storage</td><td class="_21lJbe">128 GB</td></tr>
						<tr><td class="_1hKmbr">Display Size</td><td class="_21lJbe">6.1 inch</td></tr>
					</table>
					<img class="_396cs4 _3exPp9" src="https://rukminim.flixcart.com/image/iphone15.jpg" />
				</body>
			</html>
		`;
		const res = flipkartParser.parse(
			html,
			new URL('https://www.flipkart.com/apple-iphone-15-blue-128-gb/p/itm123')
		);
		expect(res).not.toBeNull();
		expect(res?.name).toBe('Apple iPhone 15 (Blue, 128 GB)');
		expect(res?.brand).toBe('Apple');
		expect(res?.currentPricePaise).toBe(6999900);
		expect(res?.originalPricePaise).toBe(7990000);
		expect(res?.discountPercent).toBe(12);
		expect(res?.specifications?.['Internal Storage']).toBe('128 GB');
	});

	it('Croma parser extracts electronics details and price', () => {
		const html = `
			<html>
				<body>
					<h1 class="pd-title">LG 55 Inch 4K Smart OLED TV</h1>
					<span class="amount">₹1,19,990</span>
					<span class="old-price">₹1,49,990</span>
					<div class="product-image"><img src="https://media.croma.com/tv.jpg" /></div>
					<table class="cp-specification">
						<tr><td>Resolution</td><td>4K Ultra HD</td></tr>
					</table>
				</body>
			</html>
		`;
		const res = cromaParser.parse(html, new URL('https://www.croma.com/lg-tv/p/234'));
		expect(res).not.toBeNull();
		expect(res?.name).toBe('LG 55 Inch 4K Smart OLED TV');
		expect(res?.currentPricePaise).toBe(11999000);
		expect(res?.originalPricePaise).toBe(14999000);
	});

	it('Reliance Digital parser extracts product title and offer price', () => {
		const html = `
			<html>
				<body>
					<h1 class="pdp__title">Samsung Galaxy Tab S9</h1>
					<span class="pdp__offerPrice">₹72,999.00</span>
					<span class="pdp__mrpPrice">₹83,999.00</span>
				</body>
			</html>
		`;
		const res = relianceParser.parse(
			html,
			new URL('https://www.reliancedigital.in/samsung-tab/p/345')
		);
		expect(res).not.toBeNull();
		expect(res?.name).toBe('Samsung Galaxy Tab S9');
		expect(res?.currentPricePaise).toBe(7299900);
		expect(res?.originalPricePaise).toBe(8399900);
	});

	it('Apple Store parser extracts title, model, and price', () => {
		const html = `
			<html>
				<body>
					<h1 class="typography-headline">MacBook Air with M3 chip</h1>
					<span class="rc-prices-fullprice">₹1,14,900.00</span>
				</body>
			</html>
		`;
		const res = appleParser.parse(
			html,
			new URL('https://www.apple.com/in/shop/buy-mac/macbook-air')
		);
		expect(res).not.toBeNull();
		expect(res?.name).toBe('MacBook Air with M3 chip');
		expect(res?.currentPricePaise).toBe(11490000);
		expect(res?.currency).toBe('INR');
	});
});

describe('Hybrid Engine Extraction Flow', () => {
	it('extracts product via mock fetch with high confidence', async () => {
		const mockHtml = `
			<!DOCTYPE html>
			<html>
			<head>
				<script type="application/ld+json">
				{
					"@context": "https://schema.org",
					"@type": "Product",
					"name": "Bose QuietComfort 45",
					"brand": { "@type": "Brand", "name": "Bose" },
					"image": "https://images.bose.com/qc45.jpg",
					"offers": {
						"@type": "Offer",
						"price": "24900",
						"priceCurrency": "INR",
						"availability": "https://schema.org/InStock"
					}
				}
				</script>
			</head>
			<body><h1>Bose QuietComfort 45</h1></body>
			</html>
		`;

		const mockFetch = vi.fn().mockResolvedValue({
			ok: true,
			text: () => Promise.resolve(mockHtml)
		} as Response);

		const res = await extractProductFromUrl(
			'https://example.com/bose-qc45',
			mockFetch as unknown as typeof fetch
		);
		expect(res.name).toBe('Bose QuietComfort 45');
		expect(res.brand).toBe('Bose');
		expect(res.currentPricePaise).toBe(2490000);
		expect(res.currency).toBe('INR');
		expect(res.confidence).toBe('high');
	});

	it('rejects unsafe SSRF loopback destinations safely', async () => {
		const res = await extractProductFromUrl('http://127.0.0.1:8080/admin');
		expect(res.confidence).toBe('manual_fallback');
		expect(res.error).toBeDefined();
		expect(res.error).toContain('Destination resolves to a restricted internal network');
	});
});
