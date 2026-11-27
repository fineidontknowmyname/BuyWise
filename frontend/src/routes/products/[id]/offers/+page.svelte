<script lang="ts">
	import '#lib/styles/workbench.css';
	import { enhance } from '$app/forms';
	import { invalidateAll } from '$app/navigation';
	import { resolve } from '$app/paths';
	import PriceTimelineChart from '#lib/components/PriceTimelineChart.svelte';
	import ProductDocumentImport from '#lib/components/ProductDocumentImport.svelte';
	import { priceRefreshStatus } from '#lib/priceRefresh.js';
	import { formatRupees } from '#lib/money.js';
	import type { PriceObservation } from '#lib/server/api.js';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();
	let pollStopped = $state(false);
	let detailsLoading = $state(false);
	const refreshStatus = $derived(priceRefreshStatus(data.history?.refresh));
	const latestPrice = $derived(data.history?.observations[0]);
	const refreshing = $derived(
		data.history?.refresh?.state === 'queued' || data.history?.refresh?.state === 'running'
	);
	const refreshJobID = $derived(data.history?.refresh?.id);
	$effect(() => {
		pollStopped = false;
		if (!refreshing || !refreshJobID) return;
		let checks = 0;
		let pending = false;
		const poll = window.setInterval(async () => {
			if (pending) return;
			if (++checks > 12) {
				pollStopped = true;
				window.clearInterval(poll);
				return;
			}
			pending = true;
			try {
				await invalidateAll();
			} catch {
				pollStopped = true;
				window.clearInterval(poll);
			} finally {
				pending = false;
			}
		}, 5000);
		return () => window.clearInterval(poll);
	});
	const prices = $derived<PriceObservation[]>(
		(data.history?.observations ?? []).map((entry) => ({
			id: entry.id,
			productId: data.product.id,
			observedOn: entry.observedOn,
			pricePaise: entry.itemPaise,
			source: entry.sourceLabel || entry.source,
			provenance: entry.source,
			createdAt: entry.recordedAt
		}))
	);
	const availability: Record<string, string> = {
		unknown: 'Availability unknown',
		in_stock: 'In stock',
		out_of_stock: 'Out of stock',
		pre_order: 'Pre-order'
	};
	const amount = (paise: number | null) => (paise === null ? '' : String(paise / 100));
	const money = (paise: number | null) => (paise === null ? 'Unavailable' : formatRupees(paise));
	const stamp = (value: string) =>
		new Intl.DateTimeFormat('en-IN', {
			dateStyle: 'medium',
			timeStyle: 'short',
			timeZone: 'Asia/Kolkata'
		}).format(new Date(value));
</script>

<svelte:head><title>{data.product.name} · Product | BuyWise</title></svelte:head>
<div class="workbench">
	<header class="wb-heading">
		<div>
			<a class="wb-muted" href={resolve('/products')}>← Products & prices</a>
			<h1>{data.product.name}</h1>
			<p class="wb-muted">
				{[data.product.brand, data.product.model, data.product.variant]
					.filter(Boolean)
					.join(' · ') || 'Product details'}
			</p>
		</div>
		{#if data.history?.offer.listingUrl}<a
				class="btn btn-secondary"
				href={data.history.offer.listingUrl}
				target="_blank"
				rel="external noopener noreferrer">Open retailer ↗</a
			>{/if}
	</header>
	{#if form?.error}<p class="wb-notice" role="alert">{form.error}</p>{/if}{#if form?.notice}<p
			class="wb-notice"
			role="status"
		>
			{form.notice}
		</p>{/if}
	<div class="wb-columns">
		<div class="wb-stack">
			<section class="wb-card product-overview" aria-label="Saved product and latest price">
				{#if data.product.imageUrl}<img
						class="product-image"
						src={data.product.imageUrl}
						alt={data.product.name}
						referrerpolicy="no-referrer"
						onerror={(e) => ((e.currentTarget as HTMLImageElement).hidden = true)}
					/>{/if}
				<div class="price-copy">
					<p class="wb-eyebrow">{latestPrice ? 'Latest item price' : 'Saved item price'}</p>
					<p class="wb-amount">
						{latestPrice
							? formatRupees(latestPrice.itemPaise)
							: data.product.currentPricePaise > 0
								? formatRupees(data.product.currentPricePaise)
								: 'Not available'}
					</p>
					<p class="wb-muted">
						{latestPrice
							? (data.history?.offer.retailer ?? '') +
								' · ' +
								(latestPrice.observedAt ? stamp(latestPrice.observedAt) : latestPrice.observedOn)
							: data.product.currentPricePaise > 0
								? 'Saved product details · not a live quote'
								: 'Check the retailer or import a product document.'}
					</p>
					{#if latestPrice}<span class="wb-badge"
							>{data.history?.freshness === 'stale'
								? 'Needs a fresh check'
								: availability[latestPrice.availability]}</span
						>{/if}
				</div>
			</section>
			{#if data.history}
				<div class="retrieval-strip" aria-label="Price retrieval status">
					<div role="status" aria-live="polite">
						<strong
							>{refreshing
								? 'Checking retailer…'
								: data.history.refresh?.state === 'failed'
									? 'Couldn’t update price'
									: data.history.refresh?.state === 'succeeded'
										? 'Price updated'
										: 'Retailer price check'}</strong
						>
						<p class="wb-muted">
							{refreshing
								? 'We’ll update this price when the check finishes.'
								: data.history.refresh?.state === 'failed'
									? 'Your saved information is unchanged. Try again or add a price.'
									: latestPrice
										? 'Last saved price is shown above.'
										: 'No retailer price saved yet.'}
						</p>
					</div>
					<div class="wb-actions">
						{#if data.history.refreshAvailable}<form method="POST" action="?/refresh" use:enhance>
								<input type="hidden" name="offerId" value={data.history.offer.id} /><button
									class="btn btn-secondary"
									disabled={refreshing}>{refreshing ? 'Checking…' : 'Refresh price'}</button
								>
							</form>{/if}{#if refreshing}<button
								class="btn btn-ghost"
								type="button"
								onclick={() => invalidateAll()}>Check again</button
							>{/if}
					</div>
					<details class="retrieval-help">
						<summary>Details & troubleshooting</summary>
						<p class="wb-muted">{refreshStatus.detail}</p>
						{#if pollStopped && refreshing}<p class="wb-muted">
								Automatic checks paused. Use Check again to see the latest status.
							</p>{/if}{#if data.history.refresh?.nextRefreshAt}<p class="wb-muted">
								Next check: {stamp(data.history.refresh.nextRefreshAt)}
							</p>{/if}
						<form
							method="POST"
							action="?/details"
							use:enhance={() => {
								detailsLoading = true;
								return async ({ update }) => {
									try {
										await update({ reset: false });
									} finally {
										detailsLoading = false;
									}
								};
							}}
						>
							<input type="hidden" name="offerId" value={data.history.offer.id} /><label
								class="wb-check"
								><input type="checkbox" name="confirmed" value="yes" required />The listing matches
								this saved variant</label
							><button class="btn btn-secondary" disabled={detailsLoading}
								>{detailsLoading ? 'Updating…' : 'Update product details'}</button
							>
						</form>
					</details>
				</div>
			{/if}
			{#if data.history}{@const history = data.history}
				<section class="wb-card">
					<div class="wb-card-head">
						<h2>Price history</h2>
						<span class="wb-badge">{history.stats.count} observations</span>
					</div>
					{#if prices.length}<PriceTimelineChart {prices} />
						<div class="wb-metrics">
							<div class="wb-metric">
								<span>Lowest</span><strong>{money(history.stats.minimumPaise)}</strong>
							</div>
							<div class="wb-metric">
								<span>Average</span><strong>{money(history.stats.averagePaise)}</strong>
							</div>
							<div class="wb-metric">
								<span>Highest</span><strong>{money(history.stats.maximumPaise)}</strong>
							</div>
						</div>
						<details>
							<summary class="small-summary">View observations</summary>
							<div class="wb-table-wrap">
								<table>
									<thead
										><tr><th>Date</th><th>Item price</th><th>Checkout total</th><th>Source</th></tr
										></thead
									><tbody
										>{#each history.observations as entry (entry.id)}<tr
												><td>{entry.observedAt ? stamp(entry.observedAt) : entry.observedOn}</td><td
													>{formatRupees(entry.itemPaise)}</td
												><td>{money(entry.landedPaise)}</td><td
													>{entry.sourceLabel || entry.source}</td
												></tr
											>{/each}</tbody
									>
								</table>
							</div>
						</details>{:else}<div class="wb-empty">
							Your first price check starts the timeline.
						</div>{/if}
				</section>
				<details class="wb-fold">
					<summary>Add a price manually</summary>
					<div class="wb-fold-body">
						<form method="POST" action="?/record" use:enhance>
							<input type="hidden" name="offerId" value={history.offer.id} />
							<div class="wb-fields">
								<label>Item price (₹)<input name="item" inputmode="decimal" required /></label
								><label
									>Date<input
										name="date"
										type="date"
										value={data.today}
										max={data.today}
										required
									/></label
								>
							</div>
							<details class="record-extra">
								<summary class="small-summary">Checkout details</summary>
								<div class="wb-fields">
									<label>Observed time (IST)<input name="time" type="time" /></label><label
										>Tax (₹)<input name="tax" inputmode="decimal" /></label
									><label>Delivery (₹)<input name="delivery" inputmode="decimal" /></label><label
										>Discount (₹)<input name="discount" inputmode="decimal" /></label
									><label
										>Availability<select name="availability"
											>{#each Object.entries(availability) as [value, label] (value)}<option {value}
													>{label}</option
												>{/each}</select
										></label
									><label>Source note<input name="source" maxlength="150" /></label>
								</div>
								<p class="wb-muted">
									Use 0 for a confirmed zero charge. Leave unknown amounts blank.
								</p>
							</details>
							<button class="btn btn-primary">Save price</button>
						</form>
					</div>
				</details>
				<details class="wb-fold">
					<summary>Import price history</summary>
					<div class="wb-fold-body">
						<form
							class="wb-stack"
							method="POST"
							action="?/import"
							enctype="multipart/form-data"
							use:enhance
						>
							<input type="hidden" name="offerId" value={history.offer.id} /><label
								>CSV for this offer<input
									type="file"
									name="csv"
									accept=".csv,text/csv"
									required
								/></label
							><button class="btn btn-secondary">Import CSV</button>
						</form>
					</div>
				</details>
			{/if}
		</div>
		<aside class="wb-stack">
			<section class="wb-card">
				<div>
					<h2>Your price targets</h2>
					<p class="wb-muted">What you’re comfortable paying.</p>
				</div>
				<form
					method="POST"
					action="?/thresholds"
					use:enhance={() =>
						async ({ update }) =>
							update({ reset: false })}
				>
					<div class="wb-fields">
						<label
							>Ideal (₹)<input
								name="ideal"
								inputmode="decimal"
								value={amount(data.thresholds.idealPaise)}
							/></label
						><label
							>Good (₹)<input
								name="good"
								inputmode="decimal"
								value={amount(data.thresholds.goodPaise)}
							/></label
						><label
							>Maximum (₹)<input
								name="maximum"
								inputmode="decimal"
								value={amount(data.thresholds.maximumPaise)}
							/></label
						>
					</div>
					<button class="btn btn-primary form-save">Save targets</button>
					{#if form?.thresholdError}<p class="wb-notice" role="alert">{form.thresholdError}</p>{/if}
				</form>
				{#if latestPrice}<p class="wb-muted">
						{latestPrice.landedPaise === null
							? 'Checkout total not confirmed.'
							: 'Checkout total: ' + formatRupees(latestPrice.landedPaise)}
					</p>{/if}
			</section>
			<section class="wb-card">
				<h2>Retailer offers</h2>
				{#if data.offers.length}<form
						method="GET"
						action={resolve('/products/[id]/offers', { id: data.product.id })}
						class="offer-list"
					>
						{#each data.offers as offer (offer.id)}<button
								class="offer-choice"
								name="offer"
								value={offer.id}
								aria-pressed={data.history?.offer.id === offer.id}
								><span>{offer.retailer}<small>{offer.seller || 'Saved listing'}</small></span><span
									aria-hidden="true">↗</span
								></button
							>{/each}
					</form>{:else}<p class="wb-muted">No offers added.</p>{/if}
				<details>
					<summary class="small-summary">+ Add another offer</summary>
					<form class="wb-stack form-save" method="POST" action="?/offer" use:enhance>
						<input type="hidden" name="variant" value={data.product.variant ?? ''} /><label
							>Listing URL<input
								name="url"
								type="url"
								maxlength="2000"
								placeholder="https://…"
							/></label
						>
						<div class="wb-fields">
							<label
								>Retailer<input
									name="retailer"
									maxlength="100"
									placeholder="If entering manually"
								/></label
							><label>Seller<input name="seller" maxlength="150" /></label>
						</div>
						<label class="wb-check"
							><input type="checkbox" name="confirmed" value="yes" required />Matches this product
							and variant</label
						><button class="btn btn-secondary">Add offer</button>
					</form>
				</details>
			</section>
			{#if data.goals.length}<section class="wb-card">
					<h2>Link a purchase goal</h2>
					<form class="wb-stack" method="POST" action="?/goal" use:enhance>
						<label
							>Goal<select name="goalId" required
								><option value="">Choose a goal</option>{#each data.goals as goal (goal.id)}<option
										value={goal.id}>{goal.name}</option
									>{/each}</select
							></label
						><button class="btn btn-secondary">Link goal</button>
					</form>
				</section>{/if}
			<section class="wb-card" aria-label="Product information">
				<h2>Product information</h2>
				<ProductDocumentImport product={data.product} />
				<dl class="wb-lines">
					<div>
						<dt>Brand</dt>
						<dd>{data.product.brand || 'Not recorded'}</dd>
					</div>
					<div>
						<dt>Model</dt>
						<dd>{data.product.model || 'Not recorded'}</dd>
					</div>
					<div>
						<dt>Variant</dt>
						<dd>{data.product.variant || 'Not recorded'}</dd>
					</div>
					{#if data.product.currentPricePaise > 0}<div>
							<dt>Saved item price</dt>
							<dd>{formatRupees(data.product.currentPricePaise)}</dd>
						</div>{/if}
				</dl>
				{#if Object.keys(data.product.specifications ?? {}).length}<details class="wb-fold">
						<summary>Specifications</summary>
						<dl class="wb-lines wb-fold-body">
							{#each Object.entries(data.product.specifications ?? {}) as [key, value] (key)}<div>
									<dt>{key}</dt>
									<dd>{value}</dd>
								</div>{/each}
						</dl>
					</details>{/if}
				{#if data.product.notes}<details class="wb-fold">
						<summary>Notes and terms</summary>
						<p class="wb-muted wb-fold-body product-notes">{data.product.notes}</p>
					</details>{/if}
				{#if data.product.description}<details class="wb-fold">
						<summary>Description</summary>
						<p class="wb-muted wb-fold-body">{data.product.description}</p>
					</details>{/if}
				{#if data.product.warnings?.length}<details class="wb-fold">
						<summary>Link lookup notes</summary>
						<div class="wb-fold-body">
							{#each data.product.warnings as warning, index (index)}<p class="wb-muted">
									{warning}
								</p>{/each}
						</div>
					</details>{/if}
			</section>
		</aside>
	</div>
</div>

<style>
	.product-notes {
		white-space: pre-wrap;
		overflow-wrap: anywhere;
	}
	.product-overview {
		display: flex;
		align-items: center;
		flex-wrap: wrap;
		gap: 24px;
	}
	.product-image {
		width: 112px;
		height: 112px;
		object-fit: contain;
		border-radius: 12px;
		background: white;
	}
	.price-copy {
		display: grid;
		gap: 8px;
		flex: 1 1 220px;
		min-width: 0;
	}
	.retrieval-strip {
		padding: 16px 18px;
		border: 1px solid var(--border);
		border-radius: 12px;
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		justify-content: space-between;
		gap: 12px;
		background: var(--surface);
	}
	.retrieval-strip strong {
		font-size: 13px;
		font-weight: 550;
	}
	.retrieval-strip p {
		max-width: 52ch;
	}
	.retrieval-help {
		flex: 1 1 100%;
	}
	.retrieval-help summary,
	.small-summary {
		font-size: 12px;
		color: var(--foreground-secondary);
		cursor: pointer;
	}
	.retrieval-help p,
	.retrieval-help form {
		margin-top: 12px;
	}
	.retrieval-help form {
		display: grid;
		gap: 12px;
	}
	.record-extra {
		margin-block: 16px;
	}
	.record-extra .wb-fields {
		margin-block: 16px;
	}
	.offer-list {
		display: grid;
		gap: 8px;
	}
	.offer-choice {
		display: flex;
		justify-content: space-between;
		align-items: center;
		text-align: left;
		gap: 12px;
		padding: 12px;
		border: 1px solid var(--border);
		border-radius: 10px;
		background: var(--background);
		color: var(--foreground);
		font: inherit;
		cursor: pointer;
	}
	.offer-choice[aria-pressed='true'] {
		border-color: var(--brand-dot);
	}
	.offer-choice small {
		display: block;
		color: var(--foreground-secondary);
		font-size: 11px;
		margin-top: 3px;
	}
	.form-save {
		margin-top: 18px;
	}
</style>
