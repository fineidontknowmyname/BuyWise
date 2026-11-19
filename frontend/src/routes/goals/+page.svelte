<script lang="ts">
	import { resolve } from '$app/paths';
	import Modal from '#lib/components/Modal.svelte';
	import { formatRupees } from '#lib/money.js';
	import type { PageProps } from './$types';
	import {
		Target,
		Plus,
		Search,
		ArrowRight,
		Calendar,
		Sparkles,
		AlertCircle,
		Clock
	} from '@lucide/svelte';

	let { data, form }: PageProps = $props();

	let searchQuery = $state('');
	let showCreateModal = $state(false);
	let selectedPriorityFilter = $state('all');

	// Form bindings
	let formName = $state('');
	let formProductUrl = $state('');
	let formProductName = $state('');
	let formRetailer = $state('');
	let formImageUrl = $state('');
	let formBrand = $state('');
	let formModel = $state('');
	let formVariant = $state('');
	let formSpecs = $state('{}');
	let formTargetPrice = $state('');
	let formPriority = $state('medium');
	let formTargetDate = $state('');
	let formNotes = $state('');
	let formProductId = $state('');

	// Product extraction state
	let isExtracting = $state(false);
	let extractedData = $state<{
		name: string;
		brand?: string;
		model?: string;
		variant?: string;
		currentPricePaise: number;
		originalPricePaise?: number;
		specifications?: Record<string, string>;
		retailer: string;
		imageUrl: string;
		confidence: string;
		error?: string;
	} | null>(null);
	let extractError = $state('');

	async function analyzeProductUrl() {
		const raw = formProductUrl.trim();
		if (!raw) return;

		isExtracting = true;
		extractError = '';
		extractedData = null;

		try {
			const res = await fetch(resolve('/api/products/extract'), {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ url: raw })
			});

			const data = await res.json();
			if (!res.ok) {
				extractError = data.error || 'Could not extract product details.';
				return;
			}

			extractedData = data;
		} catch (err: unknown) {
			extractError =
				err instanceof Error ? err.message : 'Failed to reach product extraction service.';
		} finally {
			isExtracting = false;
		}
	}

	function applyExtractedProduct() {
		if (!extractedData) return;
		if (extractedData.name) {
			formProductName = extractedData.name;
			if (!formName) {
				formName = `Buy ${extractedData.name}`;
			}
		}
		if (extractedData.brand) formBrand = extractedData.brand;
		if (extractedData.model) formModel = extractedData.model;
		if (extractedData.variant) formVariant = extractedData.variant;
		if (extractedData.specifications) formSpecs = JSON.stringify(extractedData.specifications);
		if (extractedData.currentPricePaise > 0) {
			formTargetPrice = (extractedData.currentPricePaise / 100).toString();
		}
		if (extractedData.retailer) {
			formRetailer = extractedData.retailer;
		}
		if (extractedData.imageUrl) {
			formImageUrl = extractedData.imageUrl;
		}
	}

	function handleProductSelect(prodId: string) {
		formProductId = prodId;
		if (prodId) {
			const prod = data.products.find((p) => p.id === prodId);
			if (prod) {
				if (!formName) formName = `Buy ${prod.name}`;
				formProductName = prod.name;
				formRetailer = prod.retailer;
				formProductUrl = prod.url;
				formTargetPrice = (
					prod.targetPricePaise || prod.currentPricePaise
						? (prod.targetPricePaise || prod.currentPricePaise) / 100
						: ''
				).toString();
			}
		}
	}

	const filteredGoals = $derived.by(() => {
		let list = data.goals;
		if (searchQuery.trim()) {
			const q = searchQuery.toLowerCase();
			list = list.filter((g) => g.name.toLowerCase().includes(q));
		}
		if (selectedPriorityFilter !== 'all') {
			list = list.filter((g) => (g.priority || 'medium') === selectedPriorityFilter);
		}
		return list;
	});
</script>

<svelte:head>
	<title>Purchase Goals · BuyWise</title>
</svelte:head>

<div class="goals-page">
	<!-- Page Header -->
	<div class="page-header">
		<div>
			<h1 class="page-headline">Purchase Goals</h1>
			<p class="page-subheadline">
				Plan your next purchase and evaluate affordability before you commit.
			</p>
		</div>
		<div class="header-actions">
			<button type="button" class="btn btn-primary" onclick={() => (showCreateModal = true)}>
				<Plus size={18} />
				<span>Create Goal</span>
			</button>
		</div>
	</div>

	<!-- Search & Filter Controls -->
	{#if data.goals.length > 0}
		<div class="filter-bar">
			<div class="search-box">
				<Search size={16} class="search-icon" />
				<input
					type="search"
					placeholder="Filter goals by name..."
					class="input search-input-field"
					bind:value={searchQuery}
					aria-label="Filter goals"
				/>
			</div>

			<div class="priority-filters">
				<button
					type="button"
					class="filter-pill"
					class:active={selectedPriorityFilter === 'all'}
					onclick={() => (selectedPriorityFilter = 'all')}
				>
					All
				</button>
				<button
					type="button"
					class="filter-pill"
					class:active={selectedPriorityFilter === 'high'}
					onclick={() => (selectedPriorityFilter = 'high')}
				>
					High Priority
				</button>
				<button
					type="button"
					class="filter-pill"
					class:active={selectedPriorityFilter === 'medium'}
					onclick={() => (selectedPriorityFilter = 'medium')}
				>
					Medium
				</button>
			</div>

			<div class="goal-count-badge">
				<span>{filteredGoals.length} {filteredGoals.length === 1 ? 'goal' : 'goals'}</span>
			</div>
		</div>
	{/if}

	<!-- Error Alert -->
	{#if data.error}
		<div class="alert-box alert-error" role="alert">
			<AlertCircle size={18} />
			<span>{data.error}</span>
		</div>
	{/if}

	<!-- Goals Display -->
	{#if data.goals.length === 0}
		<div class="card empty-state">
			<div class="empty-icon-box">
				<Target size={32} />
			</div>
			<h3 class="empty-title">No purchase goals yet</h3>
			<p class="empty-text">
				Whether it's a new laptop, furniture, or a major purchase, add it here to evaluate your
				savings readiness.
			</p>
			<button type="button" class="btn btn-primary" onclick={() => (showCreateModal = true)}>
				<Plus size={18} />
				<span>Create your first goal</span>
			</button>
		</div>
	{:else if filteredGoals.length === 0}
		<div class="card empty-state">
			<h4 class="empty-title">No goals match "{searchQuery}"</h4>
			<p class="empty-text">Try searching for a different keyword or clear your filter.</p>
			<button
				type="button"
				class="btn btn-secondary btn-sm"
				onclick={() => {
					searchQuery = '';
					selectedPriorityFilter = 'all';
				}}
			>
				Clear filter
			</button>
		</div>
	{:else}
		<div class="goals-grid">
			{#each filteredGoals as goal (goal.id)}
				{@const target = goal.targetPricePaise || 0}
				{@const saved = goal.savedAmountPaise || 0}
				{@const pct = target > 0 ? Math.min(100, Math.round((saved / target) * 100)) : 0}
				{@const priority = goal.priority || 'medium'}

				<a href={resolve('/goals/[id]', { id: goal.id })} class="goal-card">
					<div class="goal-card-top">
						<div class="goal-badge">
							<Target size={18} />
						</div>
						<div class="top-tags">
							<span class="badge-priority {priority}">{priority}</span>
							<span class="badge badge-primary">{goal.status || 'active'}</span>
						</div>
					</div>

					<h3 class="goal-title">{goal.name}</h3>

					{#if target > 0}
						<div class="progress-section">
							<div class="progress-bar-bg">
								<div class="progress-bar-fill" style="width: {pct}%;"></div>
							</div>
							<div class="progress-numbers">
								<span class="saved-amt">{formatRupees(saved)} saved</span>
								<span class="target-amt">Target: {formatRupees(target)} ({pct}%)</span>
							</div>
						</div>
					{/if}

					<div class="goal-meta">
						{#if goal.targetDate}
							<div class="meta-item">
								<Clock size={14} />
								<span>Target: {goal.targetDate}</span>
							</div>
						{:else}
							<div class="meta-item">
								<Calendar size={14} />
								<span>Added {new Date(goal.createdAt).toLocaleDateString()}</span>
							</div>
						{/if}
					</div>

					<div class="goal-card-footer">
						<span class="evaluate-label">
							<Sparkles size={14} />
							<span>Evaluate Decision</span>
						</span>
						<div class="arrow-circle">
							<ArrowRight size={16} />
						</div>
					</div>
				</a>
			{/each}
		</div>
	{/if}
</div>

<!-- Modal Dialog: Create Goal -->
<Modal isOpen={showCreateModal} title="New Purchase Goal" onclose={() => (showCreateModal = false)}>
	<form method="POST" action="?/create" class="modal-form">
		<!-- Goal Name (Required) -->
		<div class="form-group">
			<label for="create-goal-name" class="label">Goal Name *</label>
			<input
				id="create-goal-name"
				name="name"
				required
				maxlength="200"
				placeholder="e.g. Ergonomic Office Chair, Sony Headphones"
				class="input"
				bind:value={formName}
			/>
		</div>

		<!-- Direct Product URL & Analyze Product Action (Section 4.A & 4.B) -->
		<div class="form-group">
			<label for="create-goal-url" class="label">Product URL (Optional)</label>
			<div class="url-input-row">
				<input
					id="create-goal-url"
					name="productUrl"
					type="url"
					placeholder="https://www.amazon.in/dp/... or Flipkart link"
					class="input url-field"
					bind:value={formProductUrl}
				/>
				<button
					type="button"
					class="btn btn-secondary btn-analyze"
					disabled={!formProductUrl.trim() || isExtracting}
					onclick={analyzeProductUrl}
				>
					{#if isExtracting}
						<Clock size={15} class="animate-spin" />
						<span>Analyzing...</span>
					{:else}
						<Sparkles size={15} />
						<span>Analyze Product</span>
					{/if}
				</button>
			</div>
			<span class="field-hint"
				>Paste an e-commerce link to extract verified title and price automatically.</span
			>
		</div>

		<!-- Product Extraction Result Banner / User Confirmation -->
		{#if isExtracting}
			<div class="extract-loading-box">
				<Clock size={16} class="animate-spin" />
				<span>Checking retailer page and validating structured product metadata...</span>
			</div>
		{:else if extractError}
			<div class="extract-error-box">
				<AlertCircle size={16} />
				<span>{extractError} (You can enter product details manually below).</span>
			</div>
		{:else if extractedData}
			<div class="extracted-preview-box">
				<div class="preview-header">
					<div class="preview-badge-row">
						<span
							class="badge"
							class:badge-success={extractedData.confidence === 'high'}
							class:badge-warning={extractedData.confidence === 'medium'}
							class:badge-neutral={extractedData.confidence === 'unverified'}
						>
							{extractedData.confidence === 'high'
								? 'Verified Retailer Price'
								: extractedData.confidence === 'medium'
									? 'Detected Price'
									: 'Unverified Price'}
						</span>
						{#if extractedData.retailer}
							<span class="preview-retailer">{extractedData.retailer}</span>
						{/if}
					</div>
					<button type="button" class="btn btn-sm btn-accent" onclick={applyExtractedProduct}>
						Confirm & Use Details
					</button>
				</div>
				<div class="preview-body">
					{#if extractedData.imageUrl}
						<img src={extractedData.imageUrl} alt={extractedData.name} class="preview-thumb" />
					{/if}
					<div class="preview-info">
						<strong class="preview-name">{extractedData.name}</strong>
						{#if extractedData.currentPricePaise > 0}
							<div class="preview-price tabular-nums">
								{formatRupees(extractedData.currentPricePaise)}
							</div>
						{:else}
							<div class="text-muted text-xs">
								Price could not be verified automatically. Enter below.
							</div>
						{/if}
					</div>
				</div>
			</div>
		{/if}

		<!-- Hidden fields populated from extraction if confirmed -->
		<input type="hidden" name="productName" value={formProductName} />
		<input type="hidden" name="retailer" value={formRetailer} />
		<input type="hidden" name="imageUrl" value={formImageUrl} />
		<input type="hidden" name="brand" value={formBrand} />
		<input type="hidden" name="model" value={formModel} />
		<input type="hidden" name="variant" value={formVariant} />
		<input type="hidden" name="specifications" value={formSpecs} />

		{#if data.products.length > 0}
			<div class="form-group">
				<label for="create-goal-product" class="label">Or Link Existing Tracked Product</label>
				<select
					id="create-goal-product"
					name="productId"
					class="input"
					value={formProductId}
					onchange={(e) => handleProductSelect(e.currentTarget.value)}
				>
					<option value="">-- None (Enter custom details or URL above) --</option>
					{#each data.products as prod (prod.id)}
						<option value={prod.id}>
							{prod.name} ({formatRupees(prod.currentPricePaise)})
						</option>
					{/each}
				</select>
			</div>
		{/if}

		<div class="form-grid-2">
			<div class="form-group">
				<label for="create-goal-target" class="label">Target Purchase Price (₹)</label>
				<input
					id="create-goal-target"
					name="targetPrice"
					type="number"
					step="0.01"
					min="0"
					placeholder="e.g. 55000"
					class="input tabular-nums font-mono"
					bind:value={formTargetPrice}
				/>
			</div>
		</div>

		<div class="form-grid-2">
			<div class="form-group">
				<label for="create-goal-priority" class="label">Priority</label>
				<select id="create-goal-priority" name="priority" class="input" bind:value={formPriority}>
					<option value="high">High Priority</option>
					<option value="medium">Medium Priority</option>
					<option value="low">Low Priority</option>
				</select>
			</div>
			<div class="form-group">
				<label for="create-goal-date" class="label">Desired Purchase Date</label>
				<input
					id="create-goal-date"
					name="targetDate"
					type="date"
					class="input"
					bind:value={formTargetDate}
				/>
			</div>
		</div>

		<div class="form-grid-2">
			<div class="form-group">
				<label for="create-goal-notes" class="label">Notes (Optional)</label>
				<input
					id="create-goal-notes"
					name="notes"
					maxlength="500"
					placeholder="e.g. Wait for Diwali sale discount"
					class="input"
					bind:value={formNotes}
				/>
			</div>
		</div>

		{#if form?.error}
			<p class="form-error" role="alert">{form.error}</p>
		{/if}

		<div class="modal-footer">
			<button type="button" class="btn btn-secondary" onclick={() => (showCreateModal = false)}>
				Cancel
			</button>
			<button type="submit" class="btn btn-primary"> Save Goal </button>
		</div>
	</form>
</Modal>

<style>
	.url-input-row {
		display: flex;
		gap: 8px;
		align-items: center;
	}

	.url-field {
		flex: 1;
	}

	.btn-analyze {
		white-space: nowrap;
		flex-shrink: 0;
	}

	.extract-loading-box {
		display: flex;
		align-items: center;
		gap: 10px;
		padding: 12px 14px;
		border-radius: 8px;
		background-color: var(--surface-secondary);
		color: var(--foreground-secondary);
		font-size: 0.84rem;
	}

	.extract-error-box {
		display: flex;
		align-items: center;
		gap: 10px;
		padding: 12px 14px;
		border-radius: 8px;
		background-color: var(--danger-surface);
		color: var(--danger);
		font-size: 0.84rem;
	}

	.extracted-preview-box {
		padding: 14px;
		border-radius: 10px;
		border: 1px solid var(--border);
		background-color: var(--surface-secondary);
		display: flex;
		flex-direction: column;
		gap: 12px;
	}

	.preview-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 10px;
		flex-wrap: wrap;
	}

	.preview-badge-row {
		display: flex;
		align-items: center;
		gap: 8px;
	}

	.preview-retailer {
		font-size: 0.78rem;
		font-weight: 600;
		color: var(--foreground-secondary);
	}

	.preview-body {
		display: flex;
		align-items: center;
		gap: 12px;
	}

	.preview-thumb {
		width: 48px;
		height: 48px;
		border-radius: 8px;
		object-fit: cover;
		background-color: #ffffff;
		border: 1px solid var(--border);
	}

	.preview-info {
		display: flex;
		flex-direction: column;
		gap: 4px;
		min-width: 0;
	}

	.preview-name {
		font-size: 0.88rem;
		color: var(--foreground);
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.preview-price {
		font-size: 1.05rem;
		font-weight: 700;
		color: var(--foreground);
	}

	.animate-spin {
		animation: spin 1s linear infinite;
	}

	@keyframes spin {
		from {
			transform: rotate(0deg);
		}
		to {
			transform: rotate(360deg);
		}
	}

	.goals-page {
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

	.page-headline {
		font-size: 28px;
		font-weight: 700;
		color: var(--text-primary);
		letter-spacing: -0.02em;
		margin-bottom: 4px;
	}

	.page-subheadline {
		font-size: 15px;
		color: var(--text-secondary);
	}

	.filter-bar {
		display: flex;
		align-items: center;
		gap: 16px;
		flex-wrap: wrap;
	}

	.search-box {
		flex: 1;
		min-width: 240px;
		position: relative;
		display: flex;
		align-items: center;
	}

	:global(.search-icon) {
		position: absolute;
		left: 12px;
		color: var(--text-muted);
		pointer-events: none;
	}

	.search-input-field {
		width: 100%;
		padding-left: 36px;
	}

	.priority-filters {
		display: flex;
		gap: 6px;
	}

	.filter-pill {
		padding: 6px 12px;
		border-radius: 999px;
		border: 1px solid var(--border-color);
		background: none;
		color: var(--text-secondary);
		font-size: 13px;
		cursor: pointer;
		transition: all var(--transition-fast);
	}

	.filter-pill.active {
		background-color: var(--primary-blue);
		color: #ffffff;
		border-color: var(--primary-blue);
	}

	.goal-count-badge {
		font-size: 13px;
		color: var(--text-muted);
		font-weight: 500;
	}

	.goals-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
		gap: 20px;
	}

	.goal-card {
		background-color: var(--card-bg);
		border: 1px solid var(--card-border);
		border-radius: var(--radius-lg);
		padding: 20px;
		display: flex;
		flex-direction: column;
		gap: 14px;
		text-decoration: none;
		box-shadow: var(--shadow-sm);
		transition:
			transform var(--transition-fast),
			box-shadow var(--transition-fast);
	}

	.goal-card:hover {
		transform: translateY(-2px);
		box-shadow: var(--shadow-md);
	}

	.goal-card-top {
		display: flex;
		align-items: center;
		justify-content: space-between;
	}

	.goal-badge {
		width: 36px;
		height: 36px;
		border-radius: var(--radius-md);
		background-color: var(--surface-secondary);
		color: var(--primary-blue);
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.top-tags {
		display: flex;
		align-items: center;
		gap: 6px;
	}

	.badge-priority {
		font-size: 10px;
		text-transform: uppercase;
		font-weight: 700;
		padding: 2px 6px;
		border-radius: 4px;
		letter-spacing: 0.04em;
	}

	.badge-priority.high {
		background-color: rgba(214, 69, 69, 0.15);
		color: var(--danger);
	}

	.badge-priority.medium {
		background-color: rgba(183, 121, 31, 0.15);
		color: var(--warning);
	}

	.badge-priority.low {
		background-color: var(--surface-secondary);
		color: var(--text-secondary);
	}

	.badge {
		font-size: 11px;
		padding: 2px 8px;
		border-radius: 999px;
		font-weight: 600;
		text-transform: capitalize;
	}

	.badge-primary {
		background-color: rgba(22, 139, 219, 0.12);
		color: var(--primary-blue);
	}

	.goal-title {
		font-size: 17px;
		font-weight: 600;
		color: var(--text-primary);
		line-height: 1.3;
	}

	.progress-section {
		display: flex;
		flex-direction: column;
		gap: 6px;
	}

	.progress-bar-bg {
		width: 100%;
		height: 7px;
		background-color: var(--surface-secondary);
		border-radius: 999px;
		overflow: hidden;
	}

	.progress-bar-fill {
		height: 100%;
		background-color: var(--primary-blue);
		border-radius: 999px;
		transition: width 0.3s ease;
	}

	.progress-numbers {
		display: flex;
		justify-content: space-between;
		font-size: 12px;
		font-variant-numeric: tabular-nums;
	}

	.saved-amt {
		font-weight: 700;
		color: var(--text-primary);
	}

	.target-amt {
		color: var(--text-muted);
	}

	.goal-meta {
		display: flex;
		gap: 12px;
		font-size: 12px;
		color: var(--text-secondary);
		margin-top: auto;
	}

	.meta-item {
		display: flex;
		align-items: center;
		gap: 5px;
	}

	.goal-card-footer {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding-top: 12px;
		border-top: 1px solid var(--border-color);
	}

	.evaluate-label {
		display: flex;
		align-items: center;
		gap: 6px;
		font-size: 13px;
		font-weight: 600;
		color: var(--primary-blue);
	}

	.arrow-circle {
		width: 28px;
		height: 28px;
		border-radius: 50%;
		background-color: var(--surface-secondary);
		color: var(--primary-blue);
		display: flex;
		align-items: center;
		justify-content: center;
		transition: transform var(--transition-fast);
	}

	.goal-card:hover .arrow-circle {
		transform: translateX(3px);
	}

	.empty-state {
		padding: 60px 24px;
		text-align: center;
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 16px;
	}

	.empty-icon-box {
		width: 64px;
		height: 64px;
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

	.empty-text {
		font-size: 14px;
		color: var(--text-secondary);
		max-width: 440px;
	}

	.modal-form {
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

	.label {
		font-size: 13px;
		font-weight: 500;
		color: var(--text-primary);
	}

	.input {
		width: 100%;
		padding: 8px 12px;
		border-radius: var(--radius-md);
		border: 1px solid var(--border-color);
		background-color: var(--card-bg);
		color: var(--text-primary);
		font-size: 14px;
		outline: none;
	}

	.input:focus {
		border-color: var(--primary-blue);
	}

	.modal-footer {
		display: flex;
		justify-content: flex-end;
		gap: 12px;
		margin-top: 10px;
		padding-top: 12px;
		border-top: 1px solid var(--border-color);
	}

	.form-error {
		color: var(--danger);
		font-size: 13px;
	}
</style>
