<script lang="ts">
	import { enhance } from '$app/forms';
	import { invalidateAll } from '$app/navigation';
	import { resolve } from '$app/paths';
	import PriceTimelineChart from '#lib/components/PriceTimelineChart.svelte';
	import { formatRupees } from '#lib/money.js';
	import type { PriceObservation } from '#lib/server/api.js';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();
	const refreshing = $derived(
		data.history?.refresh?.state === 'queued' || data.history?.refresh?.state === 'running'
	);
	const refreshJobID = $derived(data.history?.refresh?.id);
	const refreshMessages: Record<string, string> = {
		provider_rate_limited: 'The provider rate limit was reached.',
		provider_quota_exhausted:
			'The refresh budget has been reached. Saved prices are still available.',
		provider_price_unconfirmed: 'The provider could not verify a current price for this listing.',
		provider_credentials: 'Live price access is unavailable.'
	};
	$effect(() => {
		if (!refreshing || !refreshJobID) return;
		let checks = 0;
		const poll = window.setInterval(() => {
			if (++checks <= 12) void invalidateAll();
			else window.clearInterval(poll);
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

<svelte:head><title>{data.product.name} · Retailer offers | BuyWise</title></svelte:head>
<a href={resolve('/products')}>Back to products</a>
<h1>{data.product.name}</h1>
<p>
	{[data.product.brand, data.product.model, data.product.variant].filter(Boolean).join(' · ') ||
		'Variant details not recorded'}
</p>
{#if form?.error}<p role="alert">{form.error}</p>{/if}
{#if form?.notice}<p role="status">{form.notice}</p>{/if}

<section>
	<h2>Retailer offers</h2>
	{#if data.offers.length}
		<form
			method="GET"
			action={resolve('/products/[id]/offers', { id: data.product.id })}
			class="offers"
		>
			{#each data.offers as offer (offer.id)}
				<button
					class="btn"
					name="offer"
					value={offer.id}
					aria-pressed={data.history?.offer.id === offer.id}
				>
					{offer.retailer}{offer.seller ? ' · ' + offer.seller : ''}
				</button>
			{/each}
		</form>
	{:else}<p>No offers tracked for this variant.</p>{/if}
	<form method="POST" action="?/offer" use:enhance>
		<input type="hidden" name="variant" value={data.product.variant ?? ''} />
		<label
			>Retailer<input
				name="retailer"
				maxlength="100"
				placeholder="Optional for a supported URL"
			/></label
		>
		<label>Seller<input name="seller" maxlength="150" /></label>
		<label
			>Listing URL<input
				name="url"
				type="url"
				maxlength="2000"
				placeholder="Optional for manual offers"
			/></label
		>
		<label class="confirm"
			><input type="checkbox" name="confirmed" value="yes" required /> This offer matches the product
			and variant shown above.</label
		>
		<button class="btn btn-primary">Add offer</button>
	</form>
</section>

<section>
	<h2>Price thresholds</h2>
	<form
		method="POST"
		action="?/thresholds"
		use:enhance={() =>
			async ({ update }) => {
				await update({ reset: false });
			}}
	>
		<label
			>Ideal price (₹)<input
				name="ideal"
				inputmode="decimal"
				value={amount(data.thresholds.idealPaise)}
			/></label
		>
		<label
			>Good price (₹)<input
				name="good"
				inputmode="decimal"
				value={amount(data.thresholds.goodPaise)}
			/></label
		>
		<label
			>Maximum price (₹)<input
				name="maximum"
				inputmode="decimal"
				value={amount(data.thresholds.maximumPaise)}
			/></label
		>
		<button class="btn btn-primary">Save thresholds</button>
	</form>
</section>

{#if data.goals.length}
	<section>
		<h2>Link to a purchase goal</h2>
		<form method="POST" action="?/goal" use:enhance>
			<label
				>Goal<select name="goalId" required>
					<option value="">Choose an unlinked goal</option>
					{#each data.goals as goal (goal.id)}<option value={goal.id}>{goal.name}</option>{/each}
				</select></label
			>
			<button class="btn btn-primary">Link product</button>
		</form>
	</section>
{/if}

{#if data.history}
	{@const history = data.history}
	{@const latest = history.observations[0]}
	<section>
		<h2>{history.offer.retailer} price history</h2>
		{#if history.offer.listingUrl}<a
				href={history.offer.listingUrl}
				target="_blank"
				rel="external noopener noreferrer">Open listing</a
			>{/if}
		{#if latest}
			<p>
				<strong>Last observed item price {formatRupees(latest.itemPaise)}</strong> · {history.freshness ===
				'stale'
					? 'Stale'
					: 'Recent'}
			</p>
			<p>
				{availability[latest.availability]} · {latest.observedAt
					? stamp(latest.observedAt)
					: latest.observedOn}
				· {latest.sourceLabel || latest.source}
			</p>
			<p>
				Checkout total: {latest.landedPaise === null
					? 'Unknown price components'
					: formatRupees(latest.landedPaise)}
			</p>
			<p>
				Extra tax: {money(latest.taxPaise)} · Delivery: {money(latest.deliveryPaise)} · Discount: {money(
					latest.discountPaise
				)}
			</p>
			<dl>
				<dt>Lowest item price</dt>
				<dd>{money(history.stats.minimumPaise)}</dd>
				<dt>Highest item price</dt>
				<dd>{money(history.stats.maximumPaise)}</dd>
				<dt>Average item price</dt>
				<dd>{money(history.stats.averagePaise)}</dd>
				<dt>Observations</dt>
				<dd>{history.stats.count}</dd>
			</dl>
			<PriceTimelineChart {prices} />
			<p>Showing the latest {history.observations.length} observations.</p>
			<div class="table-wrap">
				<table>
					<thead
						><tr
							><th scope="col">Observed (IST)</th><th scope="col">Source</th><th scope="col"
								>Item price</th
							><th scope="col">Checkout total</th><th scope="col">Availability</th></tr
						></thead
					>
					<tbody>
						{#each history.observations as record (record.id)}
							<tr
								><td>{record.observedAt ? stamp(record.observedAt) : record.observedOn}</td>
								<td
									>{record.sourceLabel || record.source}<br />{record.fetchedAt
										? 'Fetched ' + stamp(record.fetchedAt)
										: 'Recorded ' + stamp(record.recordedAt)}</td
								>
								<td>{formatRupees(record.itemPaise)}</td><td>{money(record.landedPaise)}</td><td
									>{availability[record.availability]}</td
								></tr
							>
						{/each}
					</tbody>
				</table>
			</div>
		{:else}<p>No price observations for this offer.</p>{/if}
		{#if history.refresh}
			<p>
				Refresh status: {history.refresh.state}.
				{history.refresh.errorCode
					? (refreshMessages[history.refresh.errorCode] ??
						'The source could not confirm a fresh price.')
					: ''}
			</p>
			{#if history.refreshAvailable && history.refresh.nextRefreshAt}
				<p>Next automatic check: {stamp(history.refresh.nextRefreshAt)}</p>
			{/if}
		{/if}
		{#if history.refreshAvailable}
			<form method="POST" action="?/refresh" use:enhance>
				<input type="hidden" name="offerId" value={history.offer.id} />
				<button class="btn" disabled={refreshing}
					>{refreshing ? 'Refreshing…' : 'Refresh price'}</button
				>
			</form>
			{#if !history.refresh}
				<p>Your first refresh also schedules future price checks for this offer.</p>
			{/if}
		{:else}<p>Automatic refresh is unavailable. Record a price or import CSV history.</p>{/if}
	</section>

	<section>
		<h2>Record a price</h2>
		<form method="POST" action="?/record" use:enhance>
			<input type="hidden" name="offerId" value={history.offer.id} />
			<label
				>Observed date<input
					name="date"
					type="date"
					value={data.today}
					max={data.today}
					required
				/></label
			>
			<label>Observed time (IST)<input name="time" type="time" /></label>
			<label>Item price (₹)<input name="item" inputmode="decimal" required /></label>
			<label>Extra tax (₹)<input name="tax" inputmode="decimal" /></label>
			<label>Delivery (₹)<input name="delivery" inputmode="decimal" /></label>
			<label>Discount (₹)<input name="discount" inputmode="decimal" /></label>
			<label
				>Availability<select name="availability">
					{#each Object.entries(availability) as [value, label] (value)}<option {value}
							>{label}</option
						>{/each}
				</select></label
			>
			<label
				>Source note<input
					name="source"
					maxlength="150"
					placeholder="e.g. Retailer checkout"
				/></label
			>
			<button class="btn btn-primary">Record price</button>
		</form>
		<p>Enter zero for any confirmed zero charge. Leave unknown components blank.</p>
	</section>

	<section>
		<h2>Import offer history</h2>
		<p>
			Upload a CSV with date and item price columns in INR. Confirm that every row is for this
			seller and listing.
		</p>
		<form method="POST" action="?/import" enctype="multipart/form-data" use:enhance>
			<input type="hidden" name="offerId" value={history.offer.id} />
			<label>CSV file<input type="file" name="csv" accept=".csv,text/csv" required /></label>
			<button class="btn btn-primary">Import history</button>
		</form>
	</section>
{/if}

{#if data.product.url || data.product.currentPricePaise > 0}
	<p>
		<a href={resolve('/products/[id]', { id: data.product.id })}
			>View earlier product-level history</a
		>
	</p>
{/if}

<style>
	section {
		margin-block: 1.25rem;
		padding: clamp(1rem, 3vw, 1.5rem);
		border: 1px solid var(--border);
		border-radius: 14px;
		background: var(--surface);
	}
	h2 {
		margin-bottom: 0.75rem;
	}
	form {
		display: flex;
		flex-wrap: wrap;
		align-items: end;
		gap: 0.75rem;
		margin-block: 0.75rem;
	}
	label {
		display: grid;
		flex: 1 1 11rem;
		gap: 0.3rem;
	}
	input:not([type='checkbox'], [type='hidden']),
	select {
		color: var(--foreground);
		background: var(--background);
		border: 1px solid var(--border);
		padding: 0.55rem;
		border-radius: 8px;
		width: 100%;
	}
	.confirm {
		display: flex;
		align-items: center;
		flex: 1 1 100%;
	}
	.offers {
		display: flex;
		flex-wrap: wrap;
		gap: 0.65rem;
		list-style: none;
		padding: 0;
	}
	.offers button {
		display: block;
		padding: 0.4rem 0.65rem;
		border: 1px solid var(--border);
		border-radius: 0.5rem;
	}
	.offers [aria-pressed='true'] {
		font-weight: 700;
		border-color: var(--primary);
	}
	dl {
		display: grid;
		grid-template-columns: minmax(0, 1fr) auto;
		gap: 0.4rem 1rem;
		max-width: 28rem;
	}
	dd {
		margin: 0;
		text-align: right;
	}
	.table-wrap {
		overflow-x: auto;
	}
	table {
		width: 100%;
		border-collapse: collapse;
		text-align: left;
	}
	th,
	td {
		padding: 0.6rem;
		border-bottom: 1px solid var(--border);
		vertical-align: top;
	}
	[role='alert'] {
		color: var(--danger-foreground);
	}
</style>
