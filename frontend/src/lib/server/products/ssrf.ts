import dns from 'node:dns/promises';

// SSRF Protection: check if IP is private, loopback, or link-local
export function isPrivateOrRestrictedIp(ip: string): boolean {
	const normalized = ip.trim().toLowerCase();

	// IPv6 loopback & link-local
	if (
		normalized === '::1' ||
		normalized === '::' ||
		normalized.startsWith('fe80:') ||
		normalized.startsWith('fc00:') ||
		normalized.startsWith('fd00:')
	) {
		return true;
	}

	// IPv4 mapped IPv6 (::ffff:127.0.0.1)
	const ipv4Part = normalized.startsWith('::ffff:') ? normalized.slice(7) : normalized;

	const parts = ipv4Part.split('.').map((p) => parseInt(p, 10));
	if (parts.length === 4 && parts.every((p) => !isNaN(p) && p >= 0 && p <= 255)) {
		const [a, b] = parts;
		// 127.0.0.0/8 (Loopback)
		if (a === 127) return true;
		// 0.0.0.0/8 (Current network)
		if (a === 0) return true;
		// 10.0.0.0/8 (Private)
		if (a === 10) return true;
		// 172.16.0.0/12 (Private)
		if (a === 172 && b >= 16 && b <= 31) return true;
		// 192.168.0.0/16 (Private)
		if (a === 192 && b === 168) return true;
		// 169.254.0.0/16 (Link-local)
		if (a === 169 && b === 254) return true;
		// Broadcast or multicast
		if (a >= 224) return true;
	}

	return false;
}

// URL Normalization: Remove tracking params and trailing hash
export function normalizeProductUrl(rawUrl: string): string {
	try {
		const parsed = new URL(rawUrl.trim());
		const trackingParams = [
			'utm_source',
			'utm_medium',
			'utm_campaign',
			'utm_term',
			'utm_content',
			'ref',
			'ref_',
			'tag',
			'gclid',
			'fbclid',
			'igshid',
			'msclkid',
			'mc_cid',
			'mc_eid',
			'_ga',
			'_gl',
			'pd_rd_w',
			'pd_rd_r',
			'pd_rd_wg',
			'pf_rd_r',
			'pf_rd_p',
			'spm',
			'scm'
		];
		for (const param of trackingParams) {
			parsed.searchParams.delete(param);
		}
		parsed.hash = '';
		return parsed.toString();
	} catch {
		return rawUrl.trim();
	}
}

export async function validateUrlSecurity(
	targetUrl: URL
): Promise<{ safe: boolean; error?: string }> {
	if (targetUrl.protocol !== 'http:' && targetUrl.protocol !== 'https:') {
		return { safe: false, error: 'Only HTTP and HTTPS protocols are allowed.' };
	}

	const hostname = targetUrl.hostname.toLowerCase();
	if (
		hostname === 'localhost' ||
		hostname.endsWith('.localhost') ||
		hostname.endsWith('.local') ||
		hostname.endsWith('.internal') ||
		hostname.endsWith('.lan') ||
		hostname.endsWith('.corp')
	) {
		return { safe: false, error: 'Access to internal or local hosts is blocked.' };
	}

	try {
		const lookup = await dns.lookup(hostname);
		if (isPrivateOrRestrictedIp(lookup.address)) {
			return { safe: false, error: 'Destination resolves to a restricted internal network.' };
		}
	} catch {
		return { safe: false, error: 'Could not resolve hostname.' };
	}

	return { safe: true };
}
