import fs from 'node:fs';
import puppeteer, { type Browser } from 'puppeteer-core';
import { isPrivateOrRestrictedIp } from './ssrf.js';

const POSSIBLE_EXECUTABLES = [
	'C:\\Program Files (x86)\\Microsoft\\Edge\\Application\\msedge.exe',
	'C:\\Program Files\\Microsoft\\Edge\\Application\\msedge.exe',
	'C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe',
	'C:\\Program Files (x86)\\Google\\Chrome\\Application\\chrome.exe',
	'/usr/bin/google-chrome',
	'/usr/bin/chromium-browser',
	'/usr/bin/chromium'
];

function findBrowserExecutable(): string | null {
	for (const exe of POSSIBLE_EXECUTABLES) {
		try {
			if (fs.existsSync(exe)) {
				return exe;
			}
		} catch {
			// ignore access error
		}
	}
	return null;
}

export async function retrieveRenderedHtml(targetUrl: URL): Promise<string | null> {
	const executablePath = findBrowserExecutable();
	if (!executablePath) {
		return null;
	}

	let browser: Browser | null = null;
	try {
		browser = await puppeteer.launch({
			executablePath,
			headless: true,
			args: [
				'--no-sandbox',
				'--disable-setuid-sandbox',
				'--disable-dev-shm-usage',
				'--disable-gpu',
				'--disable-audio-output',
				'--no-first-run',
				'--no-default-browser-check'
			]
		});

		const page = await browser.newPage();
		page.setDefaultNavigationTimeout(8000);
		page.setDefaultTimeout(8000);

		// Optimize performance & bandwidth: Block heavy resource types
		await page.setRequestInterception(true);
		page.on('request', (req) => {
			const resourceType = req.resourceType();
			const reqUrl = req.url();

			// SSRF check on subrequests
			try {
				const parsed = new URL(reqUrl);
				if (isPrivateOrRestrictedIp(parsed.hostname)) {
					req.abort();
					return;
				}
			} catch {
				req.abort();
				return;
			}

			if (['image', 'media', 'font', 'stylesheet'].includes(resourceType)) {
				req.abort();
			} else {
				req.continue();
			}
		});

		await page.setUserAgent(
			'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36'
		);

		await page.goto(targetUrl.toString(), {
			waitUntil: 'domcontentloaded',
			timeout: 8000
		});

		// Small delay to let initial client-side hydration run
		await new Promise((resolve) => setTimeout(resolve, 1000));

		const renderedHtml = await page.content();
		return renderedHtml;
	} catch {
		return null;
	} finally {
		if (browser) {
			try {
				await browser.close();
			} catch {
				// ignore close error
			}
		}
	}
}
