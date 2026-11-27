export type RefreshJob = { state: string; errorCode: string; attempts: number };

const failures: Record<string, string> = {
	provider_rate_limited: 'SerpAPI is rate limited. A retry will be attempted when allowed.',
	provider_quota_exhausted: 'The lookup allowance is exhausted. Saved prices remain available.',
	provider_price_unconfirmed: 'SerpAPI did not return a matching, recent INR item price.',
	provider_credentials: 'SerpAPI rejected its API key. Check the backend configuration.',
	provider_unavailable: 'The provider could not be reached or returned an unusable response.',
	provider_unsupported: 'This listing or seller is not supported for automatic refresh.',
	listing_unavailable: 'The saved retailer listing is unavailable or unsupported.',
	invalid_provider_data: 'The returned price failed validation.',
	retry_limit: 'Automatic retries were exhausted. Try another refresh or record a price manually.'
};

export function priceRefreshStatus(job: RefreshJob | null | undefined) {
	if (!job)
		return {
			title: 'No price refresh requested',
			detail: 'Refresh the price to check the linked retailer.',
			active: false
		};
	const detail = job.errorCode
		? (failures[job.errorCode] ?? 'The provider could not confirm a price.')
		: '';
	if (job.state === 'succeeded')
		return {
			title: 'Price refresh completed',
			detail: 'The latest saved observation is shown below.',
			active: false
		};
	if (job.state === 'running')
		return {
			title: 'Retrieving price',
			detail: 'The background worker is checking this listing.',
			active: true
		};
	if (job.state === 'queued')
		return {
			title: job.errorCode ? 'Price refresh waiting to retry' : 'Price refresh queued',
			detail: detail || 'Waiting for the background worker. No new price has been confirmed yet.',
			active: true
		};
	return {
		title: 'Price refresh failed',
		detail: detail || 'No new price was saved. Check the server logs or try again.',
		active: false
	};
}
