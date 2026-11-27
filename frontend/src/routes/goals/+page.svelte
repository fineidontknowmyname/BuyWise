<script lang="ts">
	import { resolve } from '$app/paths';
	import { enhance } from '$app/forms';
	import { untrack } from 'svelte';
	import GoalSavingsFields from '#lib/components/GoalSavingsFields.svelte';
	import Modal from '#lib/components/Modal.svelte';
	import ProductListingPreview from '#lib/components/ProductListingPreview.svelte';
	import { fillListingField, listingPrice, type ProductListing } from '#lib/productListing.js';
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
	let showCreateModal = $state(untrack(() => data.draft.open || !!form?.error));
	let selectedPriorityFilter = $state('all');
	let selectedStatus = $state('active');
	const statuses = [
		{ value: 'active', label: 'Active' },
		{ value: 'paused', label: 'Paused' },
		{ value: 'completed', label: 'Completed' },
		{ value: 'all', label: 'All goals' }
	];

	// Form bindings
	let formName = $state('');
	let formProductUrl = $state('');
	let formProductName = $state('');
	let formVariant = $state('');
	let formTargetPrice = $state(untrack(() => data.draft.target));
	let formPriority = $state('medium');
	let formTargetDate = $state('');
	let formNotes = $state('');
	let formProductId = $state(untrack(() => data.draft.productId));

	// Product extraction state
	let isExtracting = $state(false);
	let extractedData = $state<ProductListing | null>(null);
	let extractError = $state('');
	let variantConfirmed = $state(false);
	const currentListing = $derived(extractedData?.url === formProductUrl ? extractedData : null);
	$effect(() => {
		void formProductUrl;
		void formVariant;
		void formProductName;
		variantConfirmed = false;
	});

	async function analyzeProductUrl() {
		const raw = formProductUrl.trim();
		if (!raw || isExtracting) return;
		const previous = extractedData;

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
			if (formProductUrl.trim() !== raw || !showCreateModal) return;
			if (!res.ok) {
				extractError = data.error || 'Could not validate the retailer link.';
				return;
			}

			extractedData = data;
			formProductName = fillListingField(formProductName, previous?.name, data.name);
			formVariant = fillListingField(formVariant, previous?.variant, data.variant);
			if (!formName && data.name) formName = `Buy ${data.name}`.slice(0, 200);
			if (String(formTargetPrice ?? '').trim() === '') formTargetPrice = listingPrice(data);
			formProductUrl = data.url;
		} catch (err: unknown) {
			extractError = err instanceof Error ? err.message : 'Could not validate the retailer link.';
		} finally {
			isExtracting = false;
		}
	}

	function handleProductSelect(prodId: string) {
		formProductId = prodId;
		if (prodId) {
			const prod = data.products.find((p) => p.id === prodId);
			if (prod) {
				if (!formName) formName = `Buy ${prod.name}`;
				formProductName = prod.name;
				formVariant = prod.variant ?? '';
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
		if (selectedStatus !== 'all') {
			list = list.filter((goal) => (goal.status ?? 'active') === selectedStatus);
		}
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
			<a class="btn btn-secondary" href={resolve('/simulator')}>Try a calculation</a>
			<button type="button" class="btn btn-primary" onclick={() => (showCreateModal = true)}>
				<Plus size={18} />
				<span>Create Goal</span>
			</button>
		</div>
	</div>

	<!-- Search & Filter Controls -->
	{#if data.goals.length > 0}
		<div class="status-filters" role="group" aria-label="Filter by goal status">
			{#each statuses as status (status.value)}
				<button
					type="button"
					class:active={selectedStatus === status.value}
					aria-pressed={selectedStatus === status.value}
					onclick={() => (selectedStatus = status.value)}
				>
					{status.label}
					<span
						>{data.goals.filter(
							(goal) => status.value === 'all' || (goal.status ?? 'active') === status.value
						).length}</span
					>
				</button>
			{/each}
		</div>
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
					aria-pressed={selectedPriorityFilter === 'all'}
					onclick={() => (selectedPriorityFilter = 'all')}
				>
					All
				</button>
				<button
					type="button"
					class="filter-pill"
					class:active={selectedPriorityFilter === 'high'}
					aria-pressed={selectedPriorityFilter === 'high'}
					onclick={() => (selectedPriorityFilter = 'high')}
				>
					High Priority
				</button>
				<button
					type="button"
					class="filter-pill"
					class:active={selectedPriorityFilter === 'medium'}
					aria-pressed={selectedPriorityFilter === 'medium'}
					onclick={() => (selectedPriorityFilter = 'medium')}
				>
					Medium
				</button>
				<button
					type="button"
					class="filter-pill"
					class:active={selectedPriorityFilter === 'low'}
					aria-pressed={selectedPriorityFilter === 'low'}
					onclick={() => (selectedPriorityFilter = 'low')}>Low</button
				>
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
	{#if data.error}
		<div class="card empty-state">
			<h2 class="empty-title">Your goals are temporarily unavailable</h2>
			<p class="empty-text">
				Your existing plans have not been changed. Try loading this page again.
			</p>
			<a class="btn btn-secondary" href={resolve('/goals')} data-sveltekit-reload>Try again</a>
		</div>
	{:else if data.goals.length === 0}
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
			<h2 class="empty-title">No goals in this view</h2>
			<p class="empty-text">Try another status, search term or priority, or show all your goals.</p>
			<button
				type="button"
				class="btn btn-secondary btn-sm"
				onclick={() => {
					searchQuery = '';
					selectedPriorityFilter = 'all';
					selectedStatus = 'all';
				}}
			>
				Clear filter
			</button>
		</div>
	{:else}
		<div class="goals-grid">
			{#each filteredGoals as goal (goal.id)}
				{@const target = goal.targetPricePaise || 0}
				{@const saved = goal.savedAmountPaise ?? null}
				{@const pct =
					target > 0 && saved !== null
						? Math.max(0, Math.min(100, Math.round((saved / target) * 100)))
						: null}
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
							{#if pct !== null}
								<div
									class="progress-bar-bg"
									role="progressbar"
									aria-label="Recorded savings for {goal.name}"
									aria-valuemin="0"
									aria-valuemax="100"
									aria-valuenow={pct}
								>
									<div class="progress-bar-fill" style:width={pct + '%'}></div>
								</div>
							{/if}
							<div class="progress-numbers">
								<span class="saved-amt"
									>{saved === null
										? 'Savings unavailable'
										: formatRupees(saved) + ' recorded'}</span
								>
								<span class="target-amt"
									>Target: {formatRupees(target)}{pct === null ? '' : ` (${pct}%)`}</span
								>
							</div>
							<p class="progress-note">Funding coverage is checked inside your goal.</p>
						</div>
					{:else}
						<p class="progress-note">Add a target price to start planning.</p>
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
							<span>View goal & plan</span>
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
	<form
		method="POST"
		action="?/create"
		class="modal-form"
		use:enhance={() =>
			async ({ update }) =>
				update({ reset: false })}
	>
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
						<span>Fetching details...</span>
					{:else}
						<Sparkles size={15} />
						<span>Fetch product details</span>
					{/if}
				</button>
			</div>
			<span class="field-hint">Add a supported retailer link and confirm the exact variant.</span>
		</div>

		<!-- Product Extraction Result Banner / User Confirmation -->
		{#if isExtracting}
			<div class="extract-loading-box">
				<Clock size={16} class="animate-spin" />
				<span>Fetching product details...</span>
			</div>
		{:else if extractError}
			<div class="extract-error-box">
				<AlertCircle size={16} />
				<span>{extractError} (You can enter product details manually below).</span>
			</div>
		{:else if currentListing}
			<ProductListingPreview listing={currentListing} />
		{/if}
		<input type="hidden" name="imageUrl" value={currentListing?.imageUrl ?? ''} />
		<input type="hidden" name="brand" value={currentListing?.brand ?? ''} />

		<div class="form-grid-2">
			<label class="form-group"
				>Product name<input
					class="input"
					name="productName"
					maxlength="200"
					bind:value={formProductName}
				/></label
			>
			<label class="form-group"
				>Exact variant<input
					class="input"
					name="variant"
					maxlength="100"
					bind:value={formVariant}
				/></label
			>
		</div>
		{#if formProductUrl && !formProductId}
			<label
				><input
					type="checkbox"
					name="variantConfirmed"
					value="yes"
					bind:checked={variantConfirmed}
					required
				/> The listing matches this product and variant.</label
			>
		{/if}

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

		<GoalSavingsFields
			target={String(formTargetPrice)}
			money={data.money}
			accounts={data.accounts}
			initial={data.draft}
		/>
		<div class="modal-footer">
			<button type="button" class="btn btn-secondary" onclick={() => (showCreateModal = false)}>
				Cancel
			</button>
			<button type="submit" class="btn btn-primary" disabled={isExtracting}> Save Goal </button>
		</div>
	</form>
</Modal>

<style>
	.header-actions,
	.status-filters {
		display: flex;
		gap: var(--space-2);
		flex-wrap: wrap;
	}
	.status-filters {
		padding-bottom: var(--space-4);
		border-bottom: 1px solid var(--border);
	}
	.status-filters button {
		min-height: var(--control-height);
		padding: var(--space-2) var(--space-3);
		border-radius: var(--radius-control);
		color: var(--foreground-secondary);
		font-size: var(--text-small);
	}
	.status-filters button.active {
		background: var(--primary);
		color: var(--primary-foreground);
	}
	.status-filters span {
		margin-inline-start: var(--space-2);
		font-variant-numeric: tabular-nums;
	}
	.progress-note {
		color: var(--foreground-secondary);
		font-size: var(--text-caption);
	}
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
		min-width: min(240px, 100%);
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
		flex-wrap: wrap;
	}

	.filter-pill {
		min-height: var(--control-height);
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
		color: var(--primary-foreground);
		border-color: var(--primary-blue);
	}

	.goal-count-badge {
		font-size: 13px;
		color: var(--text-muted);
		font-weight: 500;
	}

	.goals-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(min(100%, 320px), 1fr));
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
		gap: var(--space-2);
		flex-wrap: wrap;
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
		flex-wrap: wrap;
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
		overflow-wrap: anywhere;
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
		gap: var(--space-2);
		flex-wrap: wrap;
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
		min-height: var(--control-height);
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
	@media (max-width: 640px) {
		.form-grid-2 {
			grid-template-columns: minmax(0, 1fr);
		}
		.url-input-row {
			flex-wrap: wrap;
		}
		.url-field {
			flex-basis: 100%;
			min-width: 0;
		}
		.goal-card {
			padding: var(--space-4);
		}
	}
</style>
