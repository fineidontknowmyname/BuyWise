<script lang="ts">
	import { formatRupees } from '#lib/money.js';
	import type { ProductListing } from '#lib/productListing.js';
	let { listing }: { listing: ProductListing } = $props();
</script>

<div class="listing-preview" role="status" aria-live="polite">
	{#if listing.name}
		<div class="product-summary">
			{#if listing.imageUrl}
				<img src={listing.imageUrl} alt="" referrerpolicy="no-referrer" />
			{/if}
			<div>
				<strong>{listing.name}</strong>
				<p>{listing.retailer}{listing.variant ? ` · ${listing.variant}` : ''}</p>
				{#if listing.currentPricePaise}
					<p>Observed item price: {formatRupees(listing.currentPricePaise)}</p>
				{/if}
				{#if listing.observedAt}
					<p class="source">
						{listing.source} · {new Intl.DateTimeFormat('en-IN', {
							dateStyle: 'medium',
							timeStyle: 'short',
							timeZone: 'Asia/Kolkata'
						}).format(new Date(listing.observedAt))} IST
					</p>
				{/if}
			</div>
		</div>
	{:else}
		<strong>{listing.retailer} link accepted. Product details need manual entry.</strong>
	{/if}
	{#each listing.warnings as warning (warning)}<p>{warning}</p>{/each}
</div>

<style>
	.listing-preview {
		border: 1px solid var(--border-color, var(--border));
		background: var(--bg-secondary);
		border-radius: 14px;
		padding: 16px;
		color: var(--text-primary);
		overflow-wrap: anywhere;
	}
	.product-summary {
		display: flex;
		align-items: flex-start;
		gap: 14px;
	}
	img {
		width: 72px;
		height: 72px;
		flex-shrink: 0;
		object-fit: contain;
		background: white;
		border-radius: 10px;
	}
	p {
		font-size: 13px;
		line-height: 1.5;
		color: var(--text-secondary);
		margin: 8px 0 0;
	}
	.source {
		font-size: 12px;
	}
	@media (max-width: 420px) {
		img {
			width: 52px;
			height: 52px;
		}
	}
</style>
