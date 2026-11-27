<script lang="ts">
	import { resolve } from '$app/paths';
	import { enhance } from '$app/forms';
	import { SvelteSet } from 'svelte/reactivity';
	import { formatRupees } from '#lib/money.js';
	import Modal from '#lib/components/Modal.svelte';
	import ProductListingPreview from '#lib/components/ProductListingPreview.svelte';
	import { fillListingField, listingPrice, type ProductListing } from '#lib/productListing.js';
	import type { ProductStats } from '#lib/server/api.js';
	import {
		Plus,
		Search,
		ExternalLink,
		Sparkles,
		ShoppingBag,
		TrendingDown,
		Tag,
		Building2,
		ArrowRight,
		Globe,
		SlidersHorizontal
	} from '@lucide/svelte';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();

	let searchQuery = $state('');
	let selectedCategory = $state('all');
	let isAddModalOpen = $state(false);
	let activeTab = $state<'url' | 'manual'>('url');

	// URL extractor state
	let extractUrl = $state('');
	let isExtracting = $state(false);
	let extractError = $state('');
	let extractedData = $state<ProductListing | null>(null);
	let formVariant = $state('');
	let variantConfirmed = $state(false);
	$effect(() => {
		void formUrl;
		void formVariant;
		void formName;
		variantConfirmed = false;
	});

	// Form fields
	let formName = $state('');
	let formBrand = $state('');
	let formModel = $state('');
	let formCategory = $state('Electronics');
	let formRetailer = $state('');
	let formCurrentPrice = $state('');
	let formTargetPrice = $state('');
	let formUrl = $state('');
	let formImageUrl = $state('');
	let formNotes = $state('');

	async function handleExtract() {
		if (isExtracting) return;
		const raw = extractUrl.trim();
		const previous = extractedData;
		if (!extractUrl.trim()) {
			extractError = 'Please paste a valid product URL.';
			return;
		}
		isExtracting = true;
		extractError = '';

		try {
			const res = await fetch(resolve('/api/products/extract'), {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ url: extractUrl })
			});

			if (!res.ok) {
				const err = await res.json().catch(() => ({}));
				throw new Error(err.error || 'Failed to extract product details.');
			}

			const data = await res.json();
			if (extractUrl.trim() !== raw || !isAddModalOpen) return;
			formName = fillListingField(formName, previous?.name, data.name);
			formBrand = fillListingField(formBrand, previous?.brand, data.brand);
			formCategory = fillListingField(
				formCategory,
				previous?.category ?? 'Electronics',
				data.category
			);
			formVariant = fillListingField(formVariant, previous?.variant, data.variant);
			formCurrentPrice = fillListingField(
				formCurrentPrice,
				listingPrice(previous),
				listingPrice(data)
			);
			formImageUrl = data.imageUrl ?? '';
			extractedData = data;
			formRetailer = data.retailer;
			formUrl = data.url;
			activeTab = 'manual';
		} catch (err: unknown) {
			extractError = err instanceof Error ? err.message : 'Could not validate the retailer link.';
		} finally {
			isExtracting = false;
		}
	}

	function resetForm() {
		extractUrl = '';
		extractError = '';
		extractedData = null;
		formVariant = '';
		variantConfirmed = false;
		formName = '';
		formBrand = '';
		formModel = '';
		formCategory = 'Electronics';
		formRetailer = '';
		formCurrentPrice = '';
		formTargetPrice = '';
		formUrl = '';
		formImageUrl = '';
		formNotes = '';
		activeTab = 'url';
	}

	function openModal() {
		resetForm();
		isAddModalOpen = true;
	}

	const categories = $derived.by(() => {
		const set = new SvelteSet<string>();
		data.products.forEach((p) => {
			if (p.category) set.add(p.category);
		});
		return Array.from(set);
	});

	const filteredProducts = $derived(
		data.products.filter((p) => {
			const matchesSearch =
				p.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
				p.brand.toLowerCase().includes(searchQuery.toLowerCase()) ||
				p.retailer.toLowerCase().includes(searchQuery.toLowerCase());
			const matchesCategory = selectedCategory === 'all' || p.category === selectedCategory;
			return matchesSearch && matchesCategory;
		})
	);
</script>

<div class="products-page">
	<header class="page-header">
		<div>
			<h1 class="page-title">Products & Price History</h1>
			<p class="page-subtitle">
				Track prices across stores, import price timelines, and plan purchase decisions.
			</p>
		</div>
		<button class="btn btn-primary" onclick={openModal}>
			<Plus size={18} />
			<span>Add Product</span>
		</button>
	</header>

	{#if data.error}
		<div class="alert alert-error">
			{data.error}
		</div>
	{/if}

	<!-- Filter / Search Toolbar -->
	<div class="toolbar">
		<div class="search-box">
			<Search size={18} class="search-icon" />
			<input
				type="text"
				placeholder="Search products, brands, or stores..."
				bind:value={searchQuery}
				class="search-input"
			/>
		</div>

		{#if categories.length > 0}
			<div class="category-filter">
				<SlidersHorizontal size={16} />
				<select bind:value={selectedCategory} class="filter-select">
					<option value="all">All Categories</option>
					{#each categories as cat (cat)}
						<option value={cat}>{cat}</option>
					{/each}
				</select>
			</div>
		{/if}
	</div>

	{#if filteredProducts.length === 0}
		<div class="empty-state">
			<div class="empty-icon">
				<ShoppingBag size={48} />
			</div>
			<h3 class="empty-title">
				{searchQuery ? 'No matching products found' : 'No products tracked yet'}
			</h3>
			<p class="empty-description">
				{searchQuery
					? 'Try searching with a different term or clear the filter.'
					: 'Paste any Amazon, Flipkart, or online retailer link to start tracking its price history.'}
			</p>
			{#if !searchQuery}
				<button class="btn btn-primary" onclick={openModal}>
					<Plus size={18} />
					<span>Track Your First Product</span>
				</button>
			{/if}
		</div>
	{:else}
		<div class="products-grid">
			{#each filteredProducts as product (product.id)}
				{@const stats = (data.statsMap as Record<string, ProductStats>)?.[product.id]}
				{@const hasDrop = stats && stats.currentPricePaise < stats.averagePricePaise}
				{@const isAtLowest =
					stats && stats.observationCount > 1 && stats.currentPricePaise <= stats.lowestPricePaise}

				<div class="product-card">
					<div class="card-image-box">
						{#if product.imageUrl}
							<img src={product.imageUrl} alt={product.name} class="product-img" loading="lazy" />
						{:else}
							<div class="img-placeholder">
								<ShoppingBag size={32} />
							</div>
						{/if}
						{#if isAtLowest}
							<span class="badge badge-lowest">
								<TrendingDown size={12} /> Lowest Price
							</span>
						{:else if hasDrop}
							<span class="badge badge-drop">
								<TrendingDown size={12} /> Below Avg
							</span>
						{/if}
					</div>

					<div class="card-content">
						<div class="card-meta">
							{#if product.retailer}
								<span class="retailer-pill">
									<Building2 size={12} />
									{product.retailer}
								</span>
							{/if}
							{#if product.category}
								<span class="category-pill">
									<Tag size={12} />
									{product.category}
								</span>
							{/if}
						</div>

						<h3 class="product-name" title={product.name}>
							<a href={resolve('/products/[id]/offers', { id: product.id })}>
								{product.name}
							</a>
						</h3>

						{#if product.brand || product.model}
							<p class="product-spec">
								{[product.brand, product.model].filter(Boolean).join(' • ')}
							</p>
						{/if}

						<div class="pricing-row">
							<div class="price-block">
								<span class="price-label">Earlier product price</span>
								<span class="price-val"
									>{product.currentPricePaise > 0
										? formatRupees(product.currentPricePaise)
										: 'See offers'}</span
								>
							</div>

							{#if product.targetPricePaise > 0}
								<div class="price-block target">
									<span class="price-label">Target</span>
									<span class="price-val">{formatRupees(product.targetPricePaise)}</span>
								</div>
							{/if}
						</div>

						{#if stats && stats.observationCount > 0}
							<div class="stats-summary">
								<span>Observed {stats.observationCount}×</span>
								<span>Min: {formatRupees(stats.lowestPricePaise)}</span>
							</div>
						{/if}

						<div class="card-footer">
							<a
								href={resolve('/products/[id]/offers', { id: product.id })}
								class="btn btn-secondary btn-sm"
							>
								<span>View Details</span>
								<ArrowRight size={14} />
							</a>
							{#if product.url}
								<a
									href={product.url}
									target="_blank"
									rel="external noopener noreferrer"
									class="btn btn-icon btn-sm"
									title="Open store link"
								>
									<ExternalLink size={15} />
								</a>
							{/if}
						</div>
					</div>
				</div>
			{/each}
		</div>
	{/if}
</div>

<!-- Add Product Modal -->
<Modal isOpen={isAddModalOpen} title="Track New Product" onclose={() => (isAddModalOpen = false)}>
	<div class="modal-tabs">
		<button
			type="button"
			class="tab-btn"
			class:active={activeTab === 'url'}
			onclick={() => (activeTab = 'url')}
		>
			<Sparkles size={16} />
			<span>Use retailer link</span>
		</button>
		<button
			type="button"
			class="tab-btn"
			class:active={activeTab === 'manual'}
			onclick={() => (activeTab = 'manual')}
		>
			<Tag size={16} />
			<span>Manual Details</span>
		</button>
	</div>

	{#if activeTab === 'url'}
		<div class="extractor-view">
			<p class="extractor-hint">
				Paste the retailer listing URL, then enter and confirm the exact product variant.
			</p>

			<div class="form-group">
				<label for="extract-url" class="form-label">Product Page URL</label>
				<div class="url-input-wrap">
					<Globe size={18} class="url-icon" />
					<input
						id="extract-url"
						type="url"
						placeholder="https://www.amazon.in/dp/..."
						bind:value={extractUrl}
						class="form-input with-icon"
						disabled={isExtracting}
					/>
				</div>
			</div>

			{#if extractError}
				<div class="alert alert-warning">
					{extractError}
				</div>
			{/if}

			<div class="extractor-actions">
				<button
					type="button"
					class="btn btn-primary"
					onclick={handleExtract}
					disabled={isExtracting || !extractUrl.trim()}
				>
					{#if isExtracting}
						<span class="spinner"></span>
						<span>Fetching details...</span>
					{:else}
						<Sparkles size={16} />
						<span>Fetch product details</span>
					{/if}
				</button>
			</div>
		</div>
	{/if}

	{#if activeTab === 'manual'}
		<form method="POST" action="?/create" class="product-form" use:enhance>
			{#if form?.error}<p role="alert">{form.error}</p>{/if}
			<input type="hidden" name="url" value={formUrl} />
			<input type="hidden" name="imageUrl" value={formImageUrl} />
			{#if extractedData}<ProductListingPreview listing={extractedData} />{/if}

			<div class="form-group">
				<label for="name" class="form-label">Product Name *</label>
				<input
					id="name"
					name="name"
					type="text"
					required
					placeholder="e.g. Sony WH-1000XM5 Headphones"
					bind:value={formName}
					class="form-input"
				/>
			</div>

			<div class="form-grid-2">
				<div class="form-group">
					<label for="brand" class="form-label">Brand</label>
					<input
						id="brand"
						name="brand"
						type="text"
						placeholder="e.g. Sony"
						bind:value={formBrand}
						class="form-input"
					/>
				</div>
				<div class="form-group">
					<label for="model" class="form-label">Model</label>
					<input
						id="model"
						name="model"
						type="text"
						placeholder="e.g. WH-1000XM5"
						bind:value={formModel}
						class="form-input"
					/>
				</div>
			</div>

			<div class="form-grid-2">
				<label class="form-group"
					>Exact variant <input
						class="form-input"
						name="variant"
						bind:value={formVariant}
						maxlength="100"
						placeholder="e.g. 256 GB, black"
					/></label
				>
				<label class="form-group"
					>Seller <input class="form-input" name="seller" maxlength="150" /></label
				>
			</div>
			<label class="form-group"
				><span
					><input
						type="checkbox"
						name="variantConfirmed"
						value="yes"
						bind:checked={variantConfirmed}
						required={!!formUrl}
					/> If adding a listing, I confirmed it matches this variant.</span
				></label
			>

			<div class="form-grid-2">
				<div class="form-group">
					<label for="category" class="form-label">Category</label>
					<input
						id="category"
						name="category"
						type="text"
						placeholder="Electronics, Appliances, etc."
						bind:value={formCategory}
						class="form-input"
					/>
				</div>
				<div class="form-group">
					<label for="retailer" class="form-label">Store / Retailer</label>
					<input
						id="retailer"
						name="retailer"
						type="text"
						placeholder="Amazon, Flipkart, etc."
						bind:value={formRetailer}
						class="form-input"
					/>
				</div>
			</div>

			<div class="form-grid-2">
				<div class="form-group">
					<label for="currentPrice" class="form-label">Observed item price (INR)</label>
					<input
						id="currentPrice"
						name="currentPrice"
						type="number"
						step="0.01"
						min="0"
						placeholder="e.g. 29990"
						bind:value={formCurrentPrice}
						class="form-input"
					/>
				</div>
				<div class="form-group">
					<label for="targetPrice" class="form-label">Maximum price (INR)</label>
					<input
						id="targetPrice"
						name="targetPrice"
						type="number"
						step="0.01"
						min="0"
						placeholder="e.g. 24990"
						bind:value={formTargetPrice}
						class="form-input"
					/>
				</div>
			</div>

			<div class="form-group">
				<label for="notes" class="form-label">Notes & Specifications</label>
				<textarea
					id="notes"
					name="notes"
					rows="2"
					placeholder="Color preferences, upcoming sale dates, warranty notes..."
					bind:value={formNotes}
					class="form-textarea"></textarea>
			</div>

			<div class="modal-footer">
				<button type="button" class="btn btn-secondary" onclick={() => (isAddModalOpen = false)}>
					Cancel
				</button>
				<button type="submit" class="btn btn-primary"> Save Product </button>
			</div>
		</form>
	{/if}
</Modal>

<style>
	.products-page {
		max-width: 1200px;
		margin: 0 auto;
		display: flex;
		flex-direction: column;
		gap: 24px;
	}

	.page-header {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		gap: 16px;
		flex-wrap: wrap;
	}

	.page-title {
		font-size: 28px;
		font-weight: 700;
		color: var(--text-primary);
		letter-spacing: -0.02em;
		margin-bottom: 4px;
	}

	.page-subtitle {
		font-size: 15px;
		color: var(--text-secondary);
	}

	.toolbar {
		display: flex;
		align-items: center;
		gap: 16px;
		flex-wrap: wrap;
	}

	.search-box {
		flex: 1;
		min-width: 260px;
		position: relative;
		display: flex;
		align-items: center;
	}

	:global(.search-icon) {
		position: absolute;
		left: 14px;
		color: var(--text-muted);
		pointer-events: none;
	}

	.search-input {
		width: 100%;
		padding: 10px 14px 10px 42px;
		border-radius: var(--radius-md);
		border: 1px solid var(--border-color);
		background-color: var(--card-bg);
		color: var(--text-primary);
		font-size: 14px;
		outline: none;
		transition: border-color var(--transition-fast);
	}

	.search-input:focus {
		border-color: var(--primary-blue);
	}

	.category-filter {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 8px 12px;
		border-radius: var(--radius-md);
		border: 1px solid var(--border-color);
		background-color: var(--card-bg);
		color: var(--text-secondary);
		font-size: 14px;
	}

	.filter-select {
		border: none;
		background: transparent;
		color: var(--text-primary);
		font-size: 14px;
		outline: none;
		cursor: pointer;
	}

	.products-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
		gap: 20px;
	}

	.product-card {
		background-color: var(--card-bg);
		border: 1px solid var(--card-border);
		border-radius: var(--radius-lg);
		overflow: hidden;
		display: flex;
		flex-direction: column;
		box-shadow: var(--shadow-sm);
		transition:
			transform var(--transition-fast),
			box-shadow var(--transition-fast);
	}

	.product-card:hover {
		transform: translateY(-2px);
		box-shadow: var(--shadow-md);
	}

	.card-image-box {
		position: relative;
		height: 180px;
		background-color: var(--surface-secondary);
		display: flex;
		align-items: center;
		justify-content: center;
		overflow: hidden;
		border-bottom: 1px solid var(--border-color);
	}

	.product-img {
		width: 100%;
		height: 100%;
		object-fit: contain;
		padding: 12px;
		background-color: #ffffff;
	}

	.img-placeholder {
		color: var(--text-muted);
	}

	.badge {
		position: absolute;
		top: 10px;
		left: 10px;
		padding: 4px 8px;
		border-radius: 999px;
		font-size: 11px;
		font-weight: 600;
		display: flex;
		align-items: center;
		gap: 4px;
	}

	.badge-lowest {
		background-color: var(--success);
		color: #ffffff;
	}

	.badge-drop {
		background-color: var(--primary-blue);
		color: #ffffff;
	}

	.card-content {
		padding: 16px;
		display: flex;
		flex-direction: column;
		flex: 1;
		gap: 10px;
	}

	.card-meta {
		display: flex;
		gap: 8px;
		flex-wrap: wrap;
	}

	.retailer-pill,
	.category-pill {
		display: inline-flex;
		align-items: center;
		gap: 4px;
		padding: 2px 8px;
		border-radius: 4px;
		font-size: 11px;
		font-weight: 500;
		background-color: var(--surface-secondary);
		color: var(--text-secondary);
	}

	.product-name {
		font-size: 16px;
		font-weight: 600;
		color: var(--text-primary);
		line-height: 1.3;
		display: -webkit-box;
		-webkit-line-clamp: 2;
		line-clamp: 2;
		-webkit-box-orient: vertical;
		overflow: hidden;
	}

	.product-name a {
		color: inherit;
		text-decoration: none;
	}

	.product-name a:hover {
		color: var(--primary-blue);
	}

	.product-spec {
		font-size: 13px;
		color: var(--text-secondary);
	}

	.pricing-row {
		display: flex;
		align-items: baseline;
		justify-content: space-between;
		margin-top: auto;
		padding-top: 10px;
		border-top: 1px solid var(--border-color);
	}

	.price-block {
		display: flex;
		flex-direction: column;
	}

	.price-label {
		font-size: 11px;
		text-transform: uppercase;
		letter-spacing: 0.05em;
		color: var(--text-muted);
		font-weight: 600;
	}

	.price-val {
		font-size: 18px;
		font-weight: 700;
		color: var(--text-primary);
		font-variant-numeric: tabular-nums;
	}

	.price-block.target .price-val {
		color: var(--text-secondary);
		font-size: 15px;
	}

	.stats-summary {
		display: flex;
		justify-content: space-between;
		font-size: 12px;
		color: var(--text-muted);
	}

	.card-footer {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 8px;
		padding-top: 6px;
	}

	.empty-state {
		background-color: var(--card-bg);
		border: 1px dashed var(--border-color);
		border-radius: var(--radius-lg);
		padding: 60px 24px;
		text-align: center;
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 16px;
	}

	.empty-icon {
		width: 72px;
		height: 72px;
		border-radius: 50%;
		background-color: var(--surface-secondary);
		color: var(--primary-blue);
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.empty-title {
		font-size: 18px;
		font-weight: 600;
		color: var(--text-primary);
	}

	.empty-description {
		font-size: 14px;
		color: var(--text-secondary);
		max-width: 440px;
	}

	/* Modal Styles */
	.modal-tabs {
		display: flex;
		border-bottom: 1px solid var(--border-color);
		margin-bottom: 18px;
	}

	.tab-btn {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 10px 16px;
		font-size: 14px;
		font-weight: 500;
		color: var(--text-secondary);
		border: none;
		background: none;
		cursor: pointer;
		border-bottom: 2px solid transparent;
		transition: all var(--transition-fast);
	}

	.tab-btn.active {
		color: var(--primary-blue);
		border-bottom-color: var(--primary-blue);
		font-weight: 600;
	}

	.extractor-view {
		display: flex;
		flex-direction: column;
		gap: 16px;
	}

	.extractor-hint {
		font-size: 14px;
		color: var(--text-secondary);
		line-height: 1.5;
	}

	.url-input-wrap {
		position: relative;
		display: flex;
		align-items: center;
	}

	:global(.url-icon) {
		position: absolute;
		left: 12px;
		color: var(--text-muted);
	}

	.form-input.with-icon {
		padding-left: 38px;
	}

	.extractor-actions {
		display: flex;
		justify-content: flex-end;
		margin-top: 8px;
	}

	.product-form {
		display: flex;
		flex-direction: column;
		gap: 14px;
	}

	.form-grid-2 {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 12px;
	}

	.form-group {
		display: flex;
		flex-direction: column;
		gap: 6px;
	}

	.form-label {
		font-size: 13px;
		font-weight: 500;
		color: var(--text-primary);
	}

	.form-input,
	.form-textarea {
		width: 100%;
		padding: 10px 12px;
		border-radius: var(--radius-md);
		border: 1px solid var(--border-color);
		background-color: var(--card-bg);
		color: var(--text-primary);
		font-size: 14px;
		outline: none;
		transition: border-color var(--transition-fast);
	}

	.form-input:focus,
	.form-textarea:focus {
		border-color: var(--primary-blue);
	}

	.modal-footer {
		display: flex;
		justify-content: flex-end;
		gap: 12px;
		margin-top: 12px;
		padding-top: 12px;
		border-top: 1px solid var(--border-color);
	}

	.spinner {
		width: 16px;
		height: 16px;
		border: 2px solid rgba(255, 255, 255, 0.4);
		border-top-color: #ffffff;
		border-radius: 50%;
		animation: spin 0.7s linear infinite;
	}

	@keyframes spin {
		to {
			transform: rotate(360deg);
		}
	}
</style>
