<script lang="ts">
	import { resolve } from '$app/paths';
	import { formatRupees } from '#lib/money.js';
	import Modal from '#lib/components/Modal.svelte';
	import PriceTimelineChart from '#lib/components/PriceTimelineChart.svelte';
	import type { Goal } from '#lib/server/api.js';
	import {
		ArrowLeft,
		ExternalLink,
		Plus,
		UploadCloud,
		Sparkles,
		Tag,
		Building2,
		Trash2,
		Edit3,
		Target,
		FileSpreadsheet,
		CheckCircle2,
		AlertCircle
	} from '@lucide/svelte';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();

	// Modals
	let isEditModalOpen = $state(false);
	let isDeleteModalOpen = $state(false);
	let isCreateGoalModalOpen = $state(false);

	// Quick manual price entry
	let quickPrice = $state('');
	let quickDate = $state(new Date().toISOString().split('T')[0]);
	let quickSource = $state('');

	// CSV Importer state
	let csvDragActive = $state(false);
	let csvFile = $state<File | null>(null);
	let csvPasteText = $state('');
	let showPasteBox = $state(false);

	// AI Price Analysis state
	let isAnalyzing = $state(false);
	let aiAnalysisText = $state('');
	let aiAnalysisError = $state('');

	// Edit Product Form state
	let editName = $state('');
	let editBrand = $state('');
	let editModel = $state('');
	let editCategory = $state('');
	let editRetailer = $state('');
	let editCurrentPrice = $state('');
	let editTargetPrice = $state('');
	let editUrl = $state('');
	let editImageUrl = $state('');
	let editNotes = $state('');

	function openEditModal() {
		editName = data.product.name;
		editBrand = data.product.brand;
		editModel = data.product.model;
		editCategory = data.product.category;
		editRetailer = data.product.retailer;
		editCurrentPrice = (data.product.currentPricePaise / 100).toString();
		editTargetPrice = data.product.targetPricePaise
			? (data.product.targetPricePaise / 100).toString()
			: '';
		editUrl = data.product.url;
		editImageUrl = data.product.imageUrl;
		editNotes = data.product.notes;
		isEditModalOpen = true;
	}

	$effect(() => {
		if (data.product.retailer && !quickSource) {
			quickSource = data.product.retailer;
		}
	});

	// Check if this product is linked to any goal
	const linkedGoal = $derived(data.goals.find((g: Goal) => g.productId === data.product.id));

	async function requestPriceAnalysis() {
		if (!data.stats) return;
		isAnalyzing = true;
		aiAnalysisError = '';

		try {
			const res = await fetch(resolve('/api/ai/analyze-price'), {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					productName: data.product.name,
					stats: {
						currentPricePaise: data.product.currentPricePaise,
						lowestPricePaise: data.stats.lowestPricePaise,
						highestPricePaise: data.stats.highestPricePaise,
						averagePricePaise: data.stats.averagePricePaise,
						observationCount: data.stats.observationCount
					}
				})
			});

			if (!res.ok) {
				const err = await res.json().catch(() => ({}));
				throw new Error(err.error || 'Failed to fetch AI price analysis.');
			}

			const result = await res.json();
			aiAnalysisText = result.analysis;
		} catch (err: unknown) {
			aiAnalysisError = err instanceof Error ? err.message : 'AI price analysis unavailable.';
		} finally {
			isAnalyzing = false;
		}
	}

	function handleFileDrop(e: DragEvent) {
		e.preventDefault();
		csvDragActive = false;
		if (e.dataTransfer?.files && e.dataTransfer.files.length > 0) {
			const file = e.dataTransfer.files[0];
			if (file.name.endsWith('.csv') || file.type === 'text/csv') {
				csvFile = file;
			}
		}
	}

	function handleFileInput(e: Event) {
		const target = e.target as HTMLInputElement;
		if (target.files && target.files.length > 0) {
			csvFile = target.files[0];
		}
	}
</script>

<div class="product-detail-page">
	<!-- Breadcrumbs & Nav -->
	<div class="top-nav">
		<a href={resolve('/products')} class="back-link">
			<ArrowLeft size={16} />
			<span>All Products</span>
		</a>
		<div class="header-actions">
			<button class="btn btn-secondary btn-sm" onclick={openEditModal}>
				<Edit3 size={15} />
				<span>Edit</span>
			</button>
			<button class="btn btn-danger-outline btn-sm" onclick={() => (isDeleteModalOpen = true)}>
				<Trash2 size={15} />
				<span>Delete</span>
			</button>
		</div>
	</div>

	<!-- Alert Messages from actions -->
	{#if form?.error}
		<div class="alert alert-error">
			<AlertCircle size={18} />
			<span>{form.error}</span>
		</div>
	{/if}
	{#if form?.success && form?.importedCount}
		<div class="alert alert-success">
			<CheckCircle2 size={18} />
			<span>
				Successfully imported {form.importedCount} price observations.
				{#if form.duplicatesSkipped}
					({form.duplicatesSkipped} duplicate dates updated/skipped)
				{/if}
			</span>
		</div>
	{/if}

	<!-- Product Header Card -->
	<div class="product-hero-card">
		<div class="hero-image-wrap">
			{#if data.product.imageUrl}
				<img src={data.product.imageUrl} alt={data.product.name} class="hero-img" />
			{:else}
				<div class="hero-img-placeholder">
					<Tag size={48} />
				</div>
			{/if}
		</div>

		<div class="hero-details">
			<div class="hero-badges">
				{#if data.product.retailer}
					<span class="badge-pill">
						<Building2 size={13} />
						{data.product.retailer}
					</span>
				{/if}
				{#if data.product.category}
					<span class="badge-pill">
						<Tag size={13} />
						{data.product.category}
					</span>
				{/if}
			</div>

			<h1 class="hero-title">{data.product.name}</h1>
			<a class="btn btn-primary" href={resolve('/products/[id]/offers', { id: data.product.id })}
				>Retailer offers and price history</a
			>
			{#if data.product.brand || data.product.model}
				<p class="hero-specs">
					{[data.product.brand, data.product.model].filter(Boolean).join(' • ')}
				</p>
			{/if}

			<div class="hero-pricing">
				<div class="hero-price-item">
					<span class="hero-price-label">Earlier product-level price</span>
					<span class="hero-price-val"
						>{data.product.currentPricePaise > 0
							? formatRupees(data.product.currentPricePaise)
							: 'Unavailable'}</span
					>
				</div>

				{#if data.product.targetPricePaise > 0}
					<div class="hero-price-item target">
						<span class="hero-price-label">Target / Alert Price</span>
						<span class="hero-price-val">{formatRupees(data.product.targetPricePaise)}</span>
					</div>
				{/if}

				{#if data.product.url}
					<a
						href={data.product.url}
						target="_blank"
						rel="external noopener noreferrer"
						class="btn btn-secondary btn-sm store-link"
					>
						<span>Visit Online Store</span>
						<ExternalLink size={14} />
					</a>
				{/if}
			</div>

			{#if data.product.notes}
				<p class="hero-notes">{data.product.notes}</p>
			{/if}
		</div>
	</div>

	<!-- Main Grid: Chart & Analysis on Left, Importer & Actions on Right -->
	<div class="details-grid">
		<div class="grid-left">
			<!-- SVG Timeline Chart -->
			<section class="section-card">
				<div class="section-header">
					<div>
						<h2 class="section-title">Price History Timeline</h2>
						<p class="section-subtitle">Visual timeline of recorded price points</p>
					</div>
				</div>

				<PriceTimelineChart prices={data.prices} targetPricePaise={data.product.targetPricePaise} />
			</section>

			<!-- AI Price Analyst Card -->
			<section class="section-card ai-analyst-card">
				<div class="ai-header">
					<div class="ai-badge">
						<Sparkles size={16} />
						<span>AI Price Intelligence</span>
					</div>
					{#if !aiAnalysisText}
						<button
							type="button"
							class="btn btn-primary btn-sm"
							onclick={requestPriceAnalysis}
							disabled={isAnalyzing || !data.stats}
						>
							{#if isAnalyzing}
								<span class="spinner-small"></span>
								<span>Analyzing...</span>
							{:else}
								<Sparkles size={14} />
								<span>Analyze Price Trend</span>
							{/if}
						</button>
					{/if}
				</div>

				{#if aiAnalysisText}
					<div class="ai-analysis-content">
						<p>{aiAnalysisText}</p>
						<button
							type="button"
							class="re-analyze-btn"
							onclick={requestPriceAnalysis}
							disabled={isAnalyzing}
						>
							Re-run Analysis
						</button>
					</div>
				{:else if aiAnalysisError}
					<div class="ai-error-box">
						{aiAnalysisError}
					</div>
				{:else}
					<p class="ai-prompt-hint">
						Get an instant AI assessment on whether current prices represent a historical bargain,
						seasonal low, or if you should wait.
					</p>
				{/if}
			</section>

			<!-- Price Observations Table -->
			<section class="section-card">
				<div class="section-header">
					<div>
						<h2 class="section-title">Recorded Observations ({data.prices.length})</h2>
						<p class="section-subtitle">Individual historical records</p>
					</div>
				</div>

				{#if data.prices.length === 0}
					<p class="empty-text">No price points recorded yet.</p>
				{:else}
					<div class="table-wrap">
						<table class="data-table">
							<thead>
								<tr>
									<th>Date</th>
									<th>Price</th>
									<th>Source</th>
									<th>Provenance</th>
								</tr>
							</thead>
							<tbody>
								{#each data.prices as obs (obs.id)}
									<tr>
										<td class="tabular">{obs.observedOn}</td>
										<td class="tabular font-bold">{formatRupees(obs.pricePaise)}</td>
										<td>{obs.source || '—'}</td>
										<td>
											<span class="prov-tag">{obs.provenance || 'manual'}</span>
										</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
				{/if}
			</section>
		</div>

		<div class="grid-right">
			<!-- Linked Purchase Goal Widget -->
			<section class="section-card goal-widget">
				<div class="section-header">
					<div class="goal-widget-title">
						<Target size={18} class="accent-icon" />
						<h2 class="section-title">Purchase Goal</h2>
					</div>
				</div>

				{#if linkedGoal}
					<div class="goal-linked-box">
						<div class="goal-linked-info">
							<span class="goal-name">{linkedGoal.name}</span>
							<span class="goal-status-badge {linkedGoal.status}">{linkedGoal.status}</span>
						</div>

						<div class="goal-progress-wrap">
							<div class="progress-bar-bg">
								<div
									class="progress-bar-fill"
									style="width: {Math.min(
										100,
										Math.round(
											((linkedGoal.savedAmountPaise || 0) /
												(linkedGoal.targetPricePaise || data.product.currentPricePaise || 1)) *
												100
										)
									)}%;"
								></div>
							</div>
							<div class="progress-labels">
								<span>Saved: {formatRupees(linkedGoal.savedAmountPaise || 0)}</span>
								<span
									>Target: {formatRupees(
										linkedGoal.targetPricePaise || data.product.currentPricePaise
									)}</span
								>
							</div>
						</div>

						<a
							href={resolve('/goals/[id]', { id: linkedGoal.id })}
							class="btn btn-secondary btn-sm"
						>
							<span>View Goal Details</span>
						</a>
					</div>
				{:else}
					<div class="goal-unlinked-box">
						<p class="unlinked-desc">
							Connect this product to a purchase goal to track savings progress and get
							affordability recommendations.
						</p>
						<button
							type="button"
							class="btn btn-primary btn-sm"
							onclick={() => (isCreateGoalModalOpen = true)}
						>
							<Plus size={15} />
							<span>Create Goal for Product</span>
						</button>
					</div>
				{/if}
			</section>

			<!-- Quick Price Entry Form -->
			<section class="section-card">
				<div class="section-header">
					<h2 class="section-title">Record Price Point</h2>
				</div>
				<form method="POST" action="?/addPrice" class="quick-price-form">
					<div class="form-group">
						<label for="quick-price" class="form-label">Price (₹)</label>
						<input
							id="quick-price"
							name="price"
							type="number"
							step="0.01"
							min="0.01"
							required
							placeholder="e.g. 27999"
							bind:value={quickPrice}
							class="form-input"
						/>
					</div>

					<div class="form-group">
						<label for="quick-date" class="form-label">Date</label>
						<input
							id="quick-date"
							name="observedOn"
							type="date"
							required
							bind:value={quickDate}
							class="form-input"
						/>
					</div>

					<div class="form-group">
						<label for="quick-source" class="form-label">Source / Note</label>
						<input
							id="quick-source"
							name="source"
							type="text"
							placeholder="e.g. Amazon Great Republic Sale"
							bind:value={quickSource}
							class="form-input"
						/>
					</div>

					<button type="submit" class="btn btn-primary">
						<Plus size={16} />
						<span>Log Price</span>
					</button>
				</form>
			</section>

			<!-- CSV Importer Card -->
			<section class="section-card">
				<div class="section-header">
					<div class="importer-title">
						<FileSpreadsheet size={18} class="accent-icon" />
						<h2 class="section-title">Import CSV History</h2>
					</div>
				</div>

				<form method="POST" action="?/importCsv" enctype="multipart/form-data" class="csv-form">
					<div
						class="drop-zone"
						class:active={csvDragActive}
						ondragover={(e) => {
							e.preventDefault();
							csvDragActive = true;
						}}
						ondragleave={() => (csvDragActive = false)}
						ondrop={handleFileDrop}
						role="region"
						aria-label="CSV file drop zone"
					>
						<UploadCloud size={32} class="drop-icon" />
						{#if csvFile}
							<div class="file-picked-info">
								<span class="file-name">{csvFile.name}</span>
								<span class="file-size">({(csvFile.size / 1024).toFixed(1)} KB)</span>
							</div>
						{:else}
							<p class="drop-text">Drag & drop CSV file here, or</p>
							<label class="btn btn-secondary btn-sm browse-btn">
								<span>Browse Files</span>
								<input
									type="file"
									name="csvFile"
									accept=".csv,text/csv"
									class="hidden-file-input"
									onchange={handleFileInput}
								/>
							</label>
						{/if}
					</div>

					<div class="paste-toggle">
						<button
							type="button"
							class="text-link-btn"
							onclick={() => (showPasteBox = !showPasteBox)}
						>
							{showPasteBox ? '— Hide CSV Paste Box' : '+ Or Paste Raw CSV Text'}
						</button>
					</div>

					{#if showPasteBox}
						<div class="form-group">
							<textarea
								name="csvText"
								rows="4"
								placeholder="date,price,source&#10;2024-01-15,28990,Amazon&#10;2024-02-01,26990,Flipkart"
								bind:value={csvPasteText}
								class="form-textarea code-font"></textarea>
						</div>
					{/if}

					<div class="csv-format-help">
						<span
							>Format: <code>date, price, source</code> (e.g.
							<code>2024-05-10, 24999, Amazon</code>)</span
						>
					</div>

					<button type="submit" class="btn btn-primary" disabled={!csvFile && !csvPasteText.trim()}>
						<UploadCloud size={16} />
						<span>Upload & Process CSV</span>
					</button>
				</form>
			</section>
		</div>
	</div>
</div>

<!-- Edit Product Modal -->
<Modal isOpen={isEditModalOpen} title="Edit Product" onclose={() => (isEditModalOpen = false)}>
	<form method="POST" action="?/update" class="modal-form">
		<div class="form-group">
			<label for="edit-name" class="form-label">Product Name *</label>
			<input
				id="edit-name"
				name="name"
				type="text"
				required
				bind:value={editName}
				class="form-input"
			/>
		</div>

		<div class="form-grid-2">
			<div class="form-group">
				<label for="edit-brand" class="form-label">Brand</label>
				<input id="edit-brand" name="brand" type="text" bind:value={editBrand} class="form-input" />
			</div>
			<div class="form-group">
				<label for="edit-model" class="form-label">Model</label>
				<input id="edit-model" name="model" type="text" bind:value={editModel} class="form-input" />
			</div>
		</div>

		<div class="form-grid-2">
			<div class="form-group">
				<label for="edit-category" class="form-label">Category</label>
				<input
					id="edit-category"
					name="category"
					type="text"
					bind:value={editCategory}
					class="form-input"
				/>
			</div>
			<div class="form-group">
				<label for="edit-retailer" class="form-label">Retailer</label>
				<input
					id="edit-retailer"
					name="retailer"
					type="text"
					bind:value={editRetailer}
					class="form-input"
				/>
			</div>
		</div>

		<div class="form-grid-2">
			<div class="form-group">
				<label for="edit-current-price" class="form-label">Current Price (₹)</label>
				<input
					id="edit-current-price"
					name="currentPrice"
					type="number"
					step="0.01"
					bind:value={editCurrentPrice}
					class="form-input"
				/>
			</div>
			<div class="form-group">
				<label for="edit-target-price" class="form-label">Target / Alert Price (₹)</label>
				<input
					id="edit-target-price"
					name="targetPrice"
					type="number"
					step="0.01"
					bind:value={editTargetPrice}
					class="form-input"
				/>
			</div>
		</div>

		<div class="form-group">
			<label for="edit-url" class="form-label">Product URL</label>
			<input id="edit-url" name="url" type="url" bind:value={editUrl} class="form-input" />
		</div>

		<div class="form-group">
			<label for="edit-image-url" class="form-label">Image URL</label>
			<input
				id="edit-image-url"
				name="imageUrl"
				type="url"
				bind:value={editImageUrl}
				class="form-input"
			/>
		</div>

		<div class="form-group">
			<label for="edit-notes" class="form-label">Notes</label>
			<textarea id="edit-notes" name="notes" rows="2" bind:value={editNotes} class="form-textarea"
			></textarea>
		</div>

		<div class="modal-footer">
			<button type="button" class="btn btn-secondary" onclick={() => (isEditModalOpen = false)}>
				Cancel
			</button>
			<button type="submit" class="btn btn-primary"> Save Changes </button>
		</div>
	</form>
</Modal>

<!-- Delete Product Confirmation Modal -->
<Modal
	isOpen={isDeleteModalOpen}
	title="Delete Product"
	onclose={() => (isDeleteModalOpen = false)}
>
	<form method="POST" action="?/delete" class="modal-form">
		<p class="delete-warning">
			Are you sure you want to delete <strong>{data.product.name}</strong>? All recorded price
			history observations for this product will also be permanently removed.
		</p>
		<div class="modal-footer">
			<button type="button" class="btn btn-secondary" onclick={() => (isDeleteModalOpen = false)}>
				Cancel
			</button>
			<button type="submit" class="btn btn-danger"> Delete Permanently </button>
		</div>
	</form>
</Modal>

<!-- Create Linked Goal Modal -->
<Modal
	isOpen={isCreateGoalModalOpen}
	title="Create Goal for this Product"
	onclose={() => (isCreateGoalModalOpen = false)}
>
	<form method="POST" action="?/createGoal" class="modal-form">
		<div class="form-group">
			<label for="goal-name" class="form-label">Goal Name</label>
			<input
				id="goal-name"
				name="name"
				type="text"
				required
				value="Buy {data.product.name}"
				class="form-input"
			/>
		</div>

		<div class="form-grid-2">
			<div class="form-group">
				<label for="goal-target-price" class="form-label">Target Amount (₹)</label>
				<input
					id="goal-target-price"
					name="targetPrice"
					type="number"
					step="0.01"
					required
					value={data.product.targetPricePaise
						? data.product.targetPricePaise / 100
						: data.product.currentPricePaise / 100}
					class="form-input"
				/>
			</div>
			<div class="form-group">
				<label for="goal-priority" class="form-label">Priority</label>
				<select id="goal-priority" name="priority" class="form-input">
					<option value="high">High</option>
					<option value="medium" selected>Medium</option>
					<option value="low">Low</option>
				</select>
			</div>
		</div>

		<div class="form-group">
			<label for="goal-target-date" class="form-label">Target Purchase Date</label>
			<input id="goal-target-date" name="targetDate" type="date" class="form-input" />
		</div>

		<div class="modal-footer">
			<button
				type="button"
				class="btn btn-secondary"
				onclick={() => (isCreateGoalModalOpen = false)}
			>
				Cancel
			</button>
			<button type="submit" class="btn btn-primary"> Create Purchase Goal </button>
		</div>
	</form>
</Modal>

<style>
	.product-detail-page {
		max-width: 1200px;
		margin: 0 auto;
		display: flex;
		flex-direction: column;
		gap: 24px;
	}

	.top-nav {
		display: flex;
		align-items: center;
		justify-content: space-between;
	}

	.back-link {
		display: flex;
		align-items: center;
		gap: 6px;
		color: var(--text-secondary);
		text-decoration: none;
		font-size: 14px;
		font-weight: 500;
	}

	.back-link:hover {
		color: var(--primary-blue);
	}

	.header-actions {
		display: flex;
		align-items: center;
		gap: 8px;
	}

	.product-hero-card {
		background-color: var(--card-bg);
		border: 1px solid var(--card-border);
		border-radius: var(--radius-lg);
		padding: 24px;
		display: flex;
		gap: 24px;
		box-shadow: var(--shadow-sm);
	}

	@media (max-width: 768px) {
		.product-hero-card {
			flex-direction: column;
		}
	}

	.hero-image-wrap {
		width: 180px;
		height: 180px;
		flex-shrink: 0;
		background-color: #ffffff;
		border: 1px solid var(--border-color);
		border-radius: var(--radius-md);
		display: flex;
		align-items: center;
		justify-content: center;
		overflow: hidden;
		padding: 12px;
	}

	.hero-img {
		width: 100%;
		height: 100%;
		object-fit: contain;
	}

	.hero-img-placeholder {
		color: var(--text-muted);
	}

	.hero-details {
		display: flex;
		flex-direction: column;
		gap: 10px;
		flex: 1;
	}

	.hero-badges {
		display: flex;
		gap: 8px;
	}

	.badge-pill {
		display: inline-flex;
		align-items: center;
		gap: 4px;
		padding: 2px 8px;
		border-radius: 4px;
		font-size: 12px;
		font-weight: 500;
		background-color: var(--surface-secondary);
		color: var(--text-secondary);
	}

	.hero-title {
		font-size: 24px;
		font-weight: 700;
		color: var(--text-primary);
		line-height: 1.25;
	}

	.hero-specs {
		font-size: 14px;
		color: var(--text-secondary);
	}

	.hero-pricing {
		display: flex;
		align-items: center;
		gap: 24px;
		flex-wrap: wrap;
		margin-top: 6px;
	}

	.hero-price-item {
		display: flex;
		flex-direction: column;
	}

	.hero-price-label {
		font-size: 11px;
		text-transform: uppercase;
		letter-spacing: 0.05em;
		color: var(--text-muted);
		font-weight: 600;
	}

	.hero-price-val {
		font-size: 24px;
		font-weight: 700;
		color: var(--text-primary);
		font-variant-numeric: tabular-nums;
	}

	.hero-price-item.target .hero-price-val {
		color: var(--warning);
	}

	.store-link {
		margin-left: auto;
	}

	.hero-notes {
		font-size: 13px;
		color: var(--text-secondary);
		background-color: var(--surface-secondary);
		padding: 8px 12px;
		border-radius: var(--radius-md);
		margin-top: 4px;
	}

	/* Details Grid */
	.details-grid {
		display: grid;
		grid-template-columns: 2fr 1fr;
		gap: 24px;
	}

	@media (max-width: 900px) {
		.details-grid {
			grid-template-columns: 1fr;
		}
	}

	.grid-left,
	.grid-right {
		display: flex;
		flex-direction: column;
		gap: 24px;
	}

	.section-card {
		background-color: var(--card-bg);
		border: 1px solid var(--card-border);
		border-radius: var(--radius-lg);
		padding: 20px;
		box-shadow: var(--shadow-sm);
		display: flex;
		flex-direction: column;
		gap: 16px;
	}

	.section-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
	}

	.section-title {
		font-size: 16px;
		font-weight: 600;
		color: var(--text-primary);
	}

	.section-subtitle {
		font-size: 13px;
		color: var(--text-secondary);
	}

	/* AI Analyst Card */
	.ai-analyst-card {
		background: linear-gradient(135deg, var(--card-bg) 0%, var(--surface-secondary) 100%);
		border: 1px solid rgba(22, 139, 219, 0.2);
	}

	.ai-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
	}

	.ai-badge {
		display: flex;
		align-items: center;
		gap: 6px;
		font-size: 13px;
		font-weight: 600;
		color: var(--primary-blue);
	}

	.ai-analysis-content {
		font-size: 14px;
		line-height: 1.6;
		color: var(--text-primary);
	}

	.re-analyze-btn {
		background: none;
		border: none;
		color: var(--primary-blue);
		font-size: 12px;
		cursor: pointer;
		margin-top: 8px;
		padding: 0;
		text-decoration: underline;
	}

	.ai-prompt-hint {
		font-size: 13px;
		color: var(--text-secondary);
	}

	.ai-error-box {
		font-size: 13px;
		color: var(--danger);
	}

	/* Table */
	.table-wrap {
		overflow-x: auto;
	}

	.data-table {
		width: 100%;
		border-collapse: collapse;
		font-size: 13px;
	}

	.data-table th,
	.data-table td {
		padding: 10px 12px;
		text-align: left;
		border-bottom: 1px solid var(--border-color);
	}

	.data-table th {
		color: var(--text-muted);
		font-weight: 600;
		font-size: 11px;
		text-transform: uppercase;
		letter-spacing: 0.05em;
	}

	.tabular {
		font-variant-numeric: tabular-nums;
	}

	.prov-tag {
		font-size: 11px;
		padding: 2px 6px;
		border-radius: 4px;
		background-color: var(--surface-secondary);
		color: var(--text-secondary);
		text-transform: uppercase;
		font-weight: 500;
	}

	/* Right column widgets */
	.goal-widget-title,
	.importer-title {
		display: flex;
		align-items: center;
		gap: 8px;
	}

	:global(.accent-icon) {
		color: var(--primary-blue);
	}

	.goal-linked-box {
		display: flex;
		flex-direction: column;
		gap: 12px;
	}

	.goal-linked-info {
		display: flex;
		align-items: center;
		justify-content: space-between;
	}

	.goal-name {
		font-size: 15px;
		font-weight: 600;
		color: var(--text-primary);
	}

	.goal-status-badge {
		font-size: 11px;
		padding: 2px 8px;
		border-radius: 999px;
		font-weight: 600;
		text-transform: capitalize;
	}

	.goal-status-badge.active {
		background-color: rgba(23, 134, 91, 0.15);
		color: var(--success);
	}

	.progress-bar-bg {
		width: 100%;
		height: 8px;
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

	.progress-labels {
		display: flex;
		justify-content: space-between;
		font-size: 12px;
		color: var(--text-secondary);
		margin-top: 4px;
	}

	.goal-unlinked-box {
		display: flex;
		flex-direction: column;
		gap: 12px;
	}

	.unlinked-desc {
		font-size: 13px;
		color: var(--text-secondary);
		line-height: 1.4;
	}

	.quick-price-form {
		display: flex;
		flex-direction: column;
		gap: 12px;
	}

	/* CSV Drop Zone */
	.csv-form {
		display: flex;
		flex-direction: column;
		gap: 12px;
	}

	.drop-zone {
		border: 2px dashed var(--border-color);
		border-radius: var(--radius-md);
		padding: 20px;
		text-align: center;
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 8px;
		background-color: var(--surface-secondary);
		cursor: pointer;
		transition: all var(--transition-fast);
	}

	.drop-zone.active {
		border-color: var(--primary-blue);
		background-color: rgba(22, 139, 219, 0.05);
	}

	:global(.drop-icon) {
		color: var(--text-muted);
	}

	.drop-text {
		font-size: 13px;
		color: var(--text-secondary);
	}

	.browse-btn {
		position: relative;
		overflow: hidden;
		cursor: pointer;
	}

	.hidden-file-input {
		position: absolute;
		inset: 0;
		opacity: 0;
		cursor: pointer;
	}

	.file-picked-info {
		display: flex;
		flex-direction: column;
		gap: 2px;
	}

	.file-name {
		font-weight: 600;
		font-size: 13px;
		color: var(--text-primary);
	}

	.file-size {
		font-size: 11px;
		color: var(--text-muted);
	}

	.paste-toggle {
		display: flex;
		justify-content: flex-end;
	}

	.text-link-btn {
		background: none;
		border: none;
		color: var(--primary-blue);
		font-size: 12px;
		cursor: pointer;
		padding: 0;
	}

	.csv-format-help {
		font-size: 11px;
		color: var(--text-muted);
	}

	.csv-format-help code {
		background-color: var(--surface-secondary);
		padding: 2px 4px;
		border-radius: 3px;
	}

	.code-font {
		font-family: monospace;
		font-size: 12px;
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
		padding: 8px 12px;
		border-radius: var(--radius-md);
		border: 1px solid var(--border-color);
		background-color: var(--card-bg);
		color: var(--text-primary);
		font-size: 14px;
		outline: none;
	}

	.form-input:focus,
	.form-textarea:focus {
		border-color: var(--primary-blue);
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

	.modal-footer {
		display: flex;
		justify-content: flex-end;
		gap: 12px;
		margin-top: 12px;
		padding-top: 12px;
		border-top: 1px solid var(--border-color);
	}

	.delete-warning {
		font-size: 14px;
		line-height: 1.5;
		color: var(--text-primary);
	}

	.btn-danger-outline {
		border: 1px solid var(--danger);
		color: var(--danger);
		background: transparent;
	}

	.btn-danger-outline:hover {
		background-color: rgba(214, 69, 69, 0.1);
	}

	.spinner-small {
		width: 14px;
		height: 14px;
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
