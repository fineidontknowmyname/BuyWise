<script lang="ts">
	import { resolve } from '$app/paths';
	import { enhance } from '$app/forms';
	import PurchaseReadiness from '#lib/components/PurchaseReadiness.svelte';
	import PurchaseLifecycle from '#lib/components/PurchaseLifecycle.svelte';
	import Modal from '#lib/components/Modal.svelte';
	import PriceTimelineChart from '#lib/components/PriceTimelineChart.svelte';
	import { formatRupees } from '#lib/money.js';
	import type { PageProps } from './$types';
	import type { DocumentPreviewResult } from '#lib/server/documents/types.js';
	import {
		ArrowLeft,
		CheckCircle2,
		Clock,
		AlertTriangle,
		RefreshCw,
		Sparkles,
		Tag,
		ExternalLink,
		UploadCloud,
		FileSpreadsheet,
		Edit3,
		TrendingDown,
		TrendingUp,
		AlertCircle,
		Plus,
		Check,
		FileText,
		Layers,
		ShieldCheck
	} from '@lucide/svelte';

	let { data, form }: PageProps = $props();
	let reassessing = $state(false);

	const stateConfig = {
		BUY: {
			label: 'Buy',
			badgeClass: 'badge-success',
			icon: CheckCircle2,
			color: 'var(--success)'
		},
		WAIT: {
			label: 'Wait',
			badgeClass: 'badge-warning',
			icon: Clock,
			color: 'var(--warning)'
		},
		SAVE_MORE: {
			label: 'Save more',
			badgeClass: 'badge-primary',
			icon: RefreshCw,
			color: 'var(--primary)'
		},
		REASSESS: {
			label: 'Reassess',
			badgeClass: 'badge-warning',
			icon: AlertTriangle,
			color: 'var(--warning)'
		}
	};

	const decision = $derived(data.decision);
	const activeState = $derived(stateConfig[decision.state] ?? stateConfig.REASSESS);
	const StateIcon = $derived(activeState.icon);
	const decisionDate = new Intl.DateTimeFormat('en-IN', {
		dateStyle: 'medium',
		timeStyle: 'short',
		timeZone: 'Asia/Kolkata'
	});

	// Modals
	let showTargetPriceModal = $state(false);
	let showEditProductModal = $state(false);
	let showAddPriceModal = $state(false);
	let showDocUploadModal = $state(false);
	let showSpecsDetails = $state(false);

	// Target price edit
	let editTargetPriceVal = $state('');
	$effect(() => {
		if (!editTargetPriceVal && data.goal?.targetPricePaise) {
			editTargetPriceVal = (data.goal.targetPricePaise / 100).toString();
		}
	});

	// Manual Product Edit Form Bindings
	let editName = $state('');
	let editBrand = $state('');
	let editModel = $state('');
	let editVariant = $state('');
	let editRetailer = $state('');
	let editUrl = $state('');
	let editImageUrl = $state('');
	let editAvailability = $state('in_stock');
	let editCurrency = $state('INR');
	let editCurrentPrice = $state('');
	let editOriginalPrice = $state('');
	let editNotes = $state('');
	let editSpecsJson = $state('{}');

	function openEditProductModal() {
		if (data.product) {
			editName = data.product.name;
			editBrand = data.product.brand || '';
			editModel = data.product.model || '';
			editVariant = data.product.variant || '';
			editRetailer = data.product.retailer || '';
			editUrl = data.product.url || '';
			editImageUrl = data.product.imageUrl || '';
			editAvailability = data.product.availability || 'in_stock';
			editCurrency = data.product.currency || 'INR';
			editCurrentPrice =
				data.product.currentPricePaise > 0 ? (data.product.currentPricePaise / 100).toString() : '';
			editOriginalPrice =
				data.product.originalPricePaise && data.product.originalPricePaise > 0
					? (data.product.originalPricePaise / 100).toString()
					: '';
			editNotes = data.product.notes || '';
			editSpecsJson = data.product.specifications
				? JSON.stringify(data.product.specifications, null, 2)
				: '{}';
		} else {
			editName = data.goal?.name || '';
			editBrand = '';
			editModel = '';
			editVariant = '';
			editRetailer = '';
			editUrl = '';
			editImageUrl = '';
			editAvailability = 'in_stock';
			editCurrency = 'INR';
			editCurrentPrice = data.goal?.targetPricePaise
				? (data.goal.targetPricePaise / 100).toString()
				: '';
			editOriginalPrice = '';
			editNotes = '';
			editSpecsJson = '{}';
		}
		showEditProductModal = true;
	}

	// Add Manual Observation Form Bindings
	let manualObsPrice = $state('');
	let manualObsDate = $state(new Date().toISOString().split('T')[0]);
	let manualObsRetailer = $state('');
	let manualObsType = $state('regular');
	let manualObsNotes = $state('');

	// Document Ingestion Workflow State (Section 6)
	let docFile = $state<File | null>(null);
	let isUploadingDoc = $state(false);
	let docUploadError = $state('');
	let docPreview = $state<DocumentPreviewResult | null>(null);
	let isConfirmingImport = $state(false);
	let importCompleted = $state(false);

	async function handleDocUpload() {
		if (!docFile) return;
		isUploadingDoc = true;
		docUploadError = '';
		docPreview = null;

		const formData = new FormData();
		formData.append('docFile', docFile);

		try {
			const res = await fetch('?/uploadDocument', {
				method: 'POST',
				body: formData
			});
			const result = await res.json();
			if (result.type === 'success' && result.data?.preview) {
				docPreview = result.data.preview;
			} else {
				docUploadError = result.data?.error || 'Document upload and parsing failed.';
			}
		} catch (err: unknown) {
			docUploadError = err instanceof Error ? err.message : 'Upload failed.';
		} finally {
			isUploadingDoc = false;
		}
	}

	async function confirmDocImport() {
		if (!docPreview || docPreview.validRows.length === 0) return;
		isConfirmingImport = true;

		const formData = new FormData();
		formData.append('productId', data.product?.id || '');
		formData.append('filename', docPreview.filename);
		formData.append('rowsJson', JSON.stringify(docPreview.validRows));

		try {
			const res = await fetch('?/confirmDocumentImport', {
				method: 'POST',
				body: formData
			});
			if (res.ok) {
				importCompleted = true;
				setTimeout(() => {
					window.location.reload();
				}, 1200);
			}
		} catch (err: unknown) {
			docUploadError = err instanceof Error ? err.message : 'Confirmation failed.';
		} finally {
			isConfirmingImport = false;
		}
	}

	// Link product URL input
	let linkProductUrl = $state('');

	// Computed figures
	const currentPricePaise = $derived(
		data.product?.currentPricePaise ?? data.goal?.targetPricePaise ?? 0
	);
	const targetPricePaise = $derived(
		data.goal?.targetPricePaise ?? data.product?.targetPricePaise ?? 0
	);
	const originalPricePaise = $derived(data.product?.originalPricePaise ?? 0);
	const discountPercent = $derived(data.product?.discountPercent ?? 0);
	const savedAmountPaise = $derived(data.goal?.savedAmountPaise ?? 0);
	const savingsRequiredPaise = $derived(
		Math.max(0, (targetPricePaise || currentPricePaise) - savedAmountPaise)
	);
	const savingsProgressPct = $derived(
		targetPricePaise > 0
			? Math.min(100, Math.round((savedAmountPaise / targetPricePaise) * 100))
			: 0
	);

	// Price Difference vs Average
	const priceDiffPaise = $derived.by(() => {
		if (data.stats && data.stats.averagePricePaise > 0 && currentPricePaise > 0) {
			return currentPricePaise - data.stats.averagePricePaise;
		}
		return 0;
	});

	// Active tab in history section
	let historyTab = $state<'chart' | 'table' | 'documents'>('chart');
</script>

<svelte:head>
	<title>{data.goal?.name ?? 'Goal'} — Purchase Decision Engine</title>
</svelte:head>

<div class="goal-decision-page">
	<div class="page-nav">
		<a href={resolve('/goals')} class="back-link">
			<ArrowLeft size={16} />
			<span>Back to Goals</span>
		</a>
		{#if data.goal}
			<a href={resolve('/goals/[id]/plan', { id: data.goal.id })} class="btn btn-primary"
				>{data.purchase?.purchase ? 'View savings history' : 'Open savings plan'}</a
			>
		{/if}
	</div>

	<PurchaseLifecycle
		lifecycle={data.purchase}
		goalId={decision.goalId}
		goalName={decision.goalName}
		today={data.purchaseToday}
		allocationKey={data.allocationKey}
		active={data.goal?.status === 'active' || data.goal?.status === 'paused'}
		feedback={form}
	/>
	{#if data.purchase?.purchase || data.goal?.status === 'completed'}
		{#if data.purchase && !data.purchase.purchase}
			<p>This goal was completed before purchase details were recorded.</p>
		{/if}
		<PurchaseReadiness {decision} history={data.decisionHistory} historyOnly />
	{:else}
		{@render goalTracking()}
	{/if}
</div>

{#snippet goalTracking()}
	<div class="decision-hero card {activeState.badgeClass}">
		<div class="hero-main">
			<div class="hero-left">
				<StateIcon size={36} class="state-icon" />
				<div>
					<span class="card-kicker">Purchase decision</span>
					<h2 class="hero-title">
						Verdict:
						<span class="hero-verdict" data-testid="decision-state">{activeState.label}</span>
					</h2>
					<p class="hero-desc">{decision.explanation}</p>
				</div>
			</div>
			<div class="hero-right">
				<div class="engine-badge">
					<span>Rules {decision.rulesVersion}</span>
				</div>
				<div class="timestamp font-mono text-xs">
					Live assessment {decisionDate.format(new Date(decision.decidedAt))}
				</div>
			</div>
		</div>

		<form
			class="reassess-form"
			method="POST"
			action="?/reassess"
			use:enhance={() => {
				reassessing = true;
				return async ({ update }) => {
					try {
						await update();
					} finally {
						reassessing = false;
					}
				};
			}}
		>
			<button class="btn btn-primary" disabled={reassessing}>
				{reassessing ? 'Reassessing…' : 'Reassess and save'}
			</button>
		</form>
		{#if form?.assessmentSaved}
			<p role="status">Assessment saved to decision history.</p>
		{:else if form?.assessmentError}
			<p role="alert">{form.assessmentError}</p>
		{/if}
		<PurchaseReadiness {decision} history={data.decisionHistory} />
	</div>

	<!-- Section 3 & 4: Shared Product Profile Card -->
	{#if data.product}
		<div class="product-profile-card card">
			<div class="product-profile-top">
				<div class="product-info-left">
					{#if data.product.imageUrl}
						<img src={data.product.imageUrl} alt={data.product.name} class="product-img-thumb" />
					{:else}
						<div class="product-img-placeholder">
							<Tag size={28} />
						</div>
					{/if}

					<div class="product-text-details">
						<div class="badges-row">
							<span class="badge badge-primary">{data.product.retailer || 'Online Store'}</span>
							{#if data.product.variant}
								<span class="badge badge-secondary font-mono text-xs">{data.product.variant}</span>
							{/if}
							{#if data.product.availability === 'in_stock'}
								<span class="badge badge-success text-xs">In Stock</span>
							{:else if data.product.availability === 'out_of_stock'}
								<span class="badge badge-danger text-xs">Out of Stock</span>
							{:else if data.product.availability === 'pre_order'}
								<span class="badge badge-warning text-xs">Pre-Order</span>
							{/if}

							<span class="badge badge-muted text-xs">
								{data.product.verificationStatus === 'verified'
									? 'Verified via URL'
									: data.product.verificationStatus === 'manual_entry'
										? 'Manual Entry'
										: 'Imported Record'}
							</span>

							{#if data.product.url}
								<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
								<a
									href={data.product.url}
									target="_blank"
									rel="external noopener noreferrer"
									class="product-external-link"
								>
									<span>View on Retailer</span>
									<ExternalLink size={12} />
								</a>
							{/if}
						</div>

						<h3 class="product-name">{data.product.name}</h3>

						{#if data.product.brand || data.product.model}
							<div class="product-brand-row text-xs text-muted">
								{#if data.product.brand}<span>Brand: <strong>{data.product.brand}</strong></span
									>{/if}
								{#if data.product.model}<span class="ml-2"
										>Model: <strong>{data.product.model}</strong></span
									>{/if}
							</div>
						{/if}

						<div class="timestamp text-xs mt-1">
							Last verified: {data.product.lastRetrievedAt
								? new Date(data.product.lastRetrievedAt).toLocaleDateString()
								: data.product.updatedAt
									? new Date(data.product.updatedAt).toLocaleDateString()
									: 'Recently'}
						</div>
					</div>
				</div>

				<!-- Action Buttons (Refresh, Edit, Upload Document) -->
				<div class="product-actions-right">
					<a
						class="btn btn-secondary btn-sm"
						href={resolve('/products/[id]/offers', { id: data.product.id })}
					>
						Retailer offers
					</a>

					<button type="button" class="btn btn-secondary btn-sm" onclick={openEditProductModal}>
						<Edit3 size={14} />
						<span>Edit Product & Specs</span>
					</button>

					<button
						type="button"
						class="btn btn-secondary btn-sm"
						onclick={() => {
							docPreview = null;
							docFile = null;
							docUploadError = '';
							showDocUploadModal = true;
						}}
					>
						<UploadCloud size={14} />
						<span>Upload Document</span>
					</button>

					<button
						type="button"
						class="btn btn-accent btn-sm"
						onclick={() => (showAddPriceModal = true)}
					>
						<Plus size={14} />
						<span>Add Price Record</span>
					</button>
				</div>
			</div>

			<!-- Specifications Collapsible -->
			{#if data.product.specifications && Object.keys(data.product.specifications).length > 0}
				<div class="specs-section">
					<button
						type="button"
						class="specs-toggle-btn text-xs font-semibold"
						onclick={() => (showSpecsDetails = !showSpecsDetails)}
					>
						<Layers size={13} />
						<span
							>Technical Specifications ({Object.keys(data.product.specifications).length} attributes)</span
						>
						<span>{showSpecsDetails ? '▲ Hide' : '▼ Show'}</span>
					</button>

					{#if showSpecsDetails}
						<div class="specs-grid">
							{#each Object.entries(data.product.specifications) as [specKey, specVal] (specKey)}
								<div class="spec-item">
									<span class="spec-key">{specKey}:</span>
									<span class="spec-val">{specVal}</span>
								</div>
							{/each}
						</div>
					{/if}
				</div>
			{/if}

			<!-- Price & Progress Metrics Grid -->
			<div class="price-metrics-grid">
				<!-- Current Price -->
				<div class="metric-box">
					<span class="metric-label">Current Verified Price</span>
					<div class="price-row-display">
						<div class="metric-val tabular-nums font-mono">
							{currentPricePaise > 0 ? formatRupees(currentPricePaise) : 'Unavailable'}
						</div>
						{#if originalPricePaise > currentPricePaise && currentPricePaise > 0}
							<div class="original-price-strikethrough tabular-nums font-mono text-xs">
								{formatRupees(originalPricePaise)}
							</div>
							{#if discountPercent > 0}
								<span class="badge badge-accent text-xs font-bold">-{discountPercent}%</span>
							{/if}
						{/if}
					</div>

					{#if priceDiffPaise !== 0}
						<div
							class="metric-sub"
							class:text-success={priceDiffPaise < 0}
							class:text-danger={priceDiffPaise > 0}
						>
							{#if priceDiffPaise < 0}
								<TrendingDown size={14} />
								<span>{formatRupees(Math.abs(priceDiffPaise))} below historical average</span>
							{:else}
								<TrendingUp size={14} />
								<span>{formatRupees(priceDiffPaise)} above historical average</span>
							{/if}
						</div>
					{/if}
				</div>

				<!-- Target Purchase Price -->
				<div class="metric-box">
					<span class="metric-label">Your Target Price</span>
					<div class="metric-val tabular-nums font-mono text-primary">
						{targetPricePaise > 0 ? formatRupees(targetPricePaise) : 'Not set'}
					</div>
					<div class="metric-sub">
						<button
							type="button"
							class="link-action text-xs"
							onclick={() => (showTargetPriceModal = true)}
						>
							Edit target price
						</button>
					</div>
				</div>

				<!-- Savings Progress -->
				<div class="metric-box">
					<span class="metric-label">Savings Progress</span>
					<div class="metric-val tabular-nums font-mono">
						{savingsProgressPct}%
					</div>
					<div class="progress-track-sm">
						<div class="progress-fill-sm" style="width: {savingsProgressPct}%;"></div>
					</div>
					<div class="metric-sub font-mono text-xs">
						<span>{formatRupees(savedAmountPaise)} saved</span>
					</div>
				</div>

				<!-- Savings Required -->
				<div class="metric-box">
					<span class="metric-label">Savings Required</span>
					<div
						class="metric-val tabular-nums font-mono"
						class:text-success={savingsRequiredPaise === 0}
						class:text-danger={savingsRequiredPaise > 0}
					>
						{formatRupees(savingsRequiredPaise)}
					</div>
					<div class="metric-sub">
						<span>{savingsRequiredPaise === 0 ? 'Goal fully funded!' : 'Needed before buying'}</span
						>
					</div>
				</div>
			</div>

			<!-- Price History Analytics & Visualizations -->
			<div class="history-tabs-container">
				<div class="history-tabs-nav" role="tablist">
					<button
						type="button"
						class="history-tab-btn"
						class:active={historyTab === 'chart'}
						onclick={() => (historyTab = 'chart')}
					>
						<TrendingUp size={14} />
						<span>Price Timeline ({data.prices.length})</span>
					</button>

					<button
						type="button"
						class="history-tab-btn"
						class:active={historyTab === 'table'}
						onclick={() => (historyTab = 'table')}
					>
						<FileSpreadsheet size={14} />
						<span>Observations Ledger</span>
					</button>

					<button
						type="button"
						class="history-tab-btn"
						class:active={historyTab === 'documents'}
						onclick={() => (historyTab = 'documents')}
					>
						<FileText size={14} />
						<span>Uploaded Documents ({data.documents.length})</span>
					</button>
				</div>

				{#if historyTab === 'chart'}
					{#if data.prices && data.prices.length > 0}
						<div class="chart-content-box">
							<div class="chart-stats-row">
								<div class="stat-item">
									<span class="stat-lbl">Lowest Recorded:</span>
									<strong class="font-mono tabular-nums text-success">
										{data.stats ? formatRupees(data.stats.lowestPricePaise) : 'N/A'}
									</strong>
								</div>
								<div class="stat-item">
									<span class="stat-lbl">Average Recorded:</span>
									<strong class="font-mono tabular-nums">
										{data.stats ? formatRupees(data.stats.averagePricePaise) : 'N/A'}
									</strong>
								</div>
								<div class="stat-item">
									<span class="stat-lbl">Highest Recorded:</span>
									<strong class="font-mono tabular-nums text-danger">
										{data.stats ? formatRupees(data.stats.highestPricePaise) : 'N/A'}
									</strong>
								</div>
							</div>
							<PriceTimelineChart prices={data.prices} {targetPricePaise} />
						</div>
					{:else}
						<div class="empty-history-box text-center py-6 text-muted">
							<Clock size={28} class="mx-auto mb-2 opacity-50" />
							<p>No historical price observations yet.</p>
							<p class="text-xs">
								Refresh price, upload a document (CSV/PDF), or manually add an observation.
							</p>
						</div>
					{/if}
				{:else if historyTab === 'table'}
					<div class="table-content-box">
						<table class="table text-xs">
							<thead>
								<tr>
									<th>Date</th>
									<th>Price</th>
									<th>Type</th>
									<th>Retailer / Source</th>
									<th>Provenance</th>
									<th>Notes</th>
								</tr>
							</thead>
							<tbody>
								{#each data.prices as obs (obs.id)}
									<tr>
										<td class="font-mono font-semibold">{obs.observedOn}</td>
										<td class="font-mono tabular-nums font-bold text-accent">
											{formatRupees(obs.pricePaise)}
										</td>
										<td>
											<span class="badge badge-secondary text-xs uppercase"
												>{obs.priceType || 'regular'}</span
											>
										</td>
										<td>{obs.retailer || obs.source || 'Store'}</td>
										<td>
											<span class="badge badge-muted text-xs">{obs.provenance || 'manual'}</span>
										</td>
										<td class="text-muted">{obs.notes || '—'}</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
				{:else if historyTab === 'documents'}
					<div class="documents-content-box">
						{#if data.documents.length > 0}
							<div class="docs-list">
								{#each data.documents as doc (doc.id)}
									<div class="doc-item card">
										<div class="doc-icon">
											<FileText size={22} class="text-primary" />
										</div>
										<div class="doc-info">
											<strong class="doc-name">{doc.filename}</strong>
											<div class="doc-meta text-xs text-muted">
												<span>Format: {doc.fileType.toUpperCase()}</span>
												<span class="ml-2"
													>Uploaded: {new Date(doc.createdAt).toLocaleDateString()}</span
												>
												{#if doc.summary && typeof doc.summary === 'object'}
													<span class="ml-2"
														>Imported: {doc.summary.importedObservations ??
															doc.summary.validCount ??
															0} observations</span
													>
												{/if}
											</div>
										</div>
										<div class="doc-status">
											<span class="badge badge-success text-xs">
												<ShieldCheck size={12} />
												<span>{doc.status}</span>
											</span>
										</div>
									</div>
								{/each}
							</div>
						{:else}
							<div class="empty-history-box text-center py-6 text-muted">
								<UploadCloud size={28} class="mx-auto mb-2 opacity-50" />
								<p>No documents uploaded for this purchase goal yet.</p>
								<button
									type="button"
									class="btn btn-secondary btn-sm mt-2"
									onclick={() => (showDocUploadModal = true)}
								>
									Upload Product Document
								</button>
							</div>
						{/if}
					</div>
				{/if}
			</div>
		</div>
	{:else}
		<!-- Goal has no product linked yet: Link Product Card -->
		<div class="link-product-card card">
			<div class="link-header">
				<Tag size={28} class="text-primary" />
				<div>
					<h3 class="card-title">Attach Product to Goal</h3>
					<p class="card-desc">
						Track live prices from retailer URLs, import specification documents or historical
						spreadsheets, or enter details manually.
					</p>
				</div>
			</div>

			<div class="link-actions-grid">
				<!-- Method 1: URL Extraction -->
				<div class="link-option-card">
					<h4>1. Extract from Retailer URL</h4>
					<p class="text-xs text-muted mb-3">
						Paste product URL from Amazon, Flipkart, Croma, Apple, etc.
					</p>
					<form method="POST" action="?/linkProduct" class="link-url-form">
						<input
							type="url"
							name="productUrl"
							placeholder="https://www.amazon.in/dp/..."
							class="input"
							required
							bind:value={linkProductUrl}
						/>
						<label
							>Product name<input
								class="form-input"
								name="productName"
								maxlength="200"
								required
							/></label
						>
						<label>Exact variant<input class="form-input" name="variant" maxlength="100" /></label>
						<label
							><input type="checkbox" name="variantConfirmed" value="yes" required /> The listing matches
							this product and variant.</label
						>
						<button type="submit" class="btn btn-primary btn-sm mt-2">
							<Sparkles size={14} />
							<span>Analyze & Attach</span>
						</button>
					</form>
				</div>

				<!-- Method 2: Document Upload -->
				<div class="link-option-card">
					<h4>2. Upload Product Document</h4>
					<p class="text-xs text-muted mb-3">Upload CSV, XLSX, PDF spec sheet, or invoice.</p>
					<button
						type="button"
						class="btn btn-secondary btn-sm"
						onclick={() => (showDocUploadModal = true)}
					>
						<UploadCloud size={14} />
						<span>Upload File</span>
					</button>
				</div>

				<!-- Method 3: Manual Entry -->
				<div class="link-option-card">
					<h4>3. Manual Product Entry</h4>
					<p class="text-xs text-muted mb-3">
						Manually enter brand, model, specifications, and target price.
					</p>
					<button type="button" class="btn btn-secondary btn-sm" onclick={openEditProductModal}>
						<Edit3 size={14} />
						<span>Enter Details Manually</span>
					</button>
				</div>
			</div>
		</div>
	{/if}
{/snippet}

<!-- =========================================================================
     MODAL 1: Edit Target Price
     ========================================================================= -->
<Modal
	isOpen={showTargetPriceModal}
	title="Edit Target Price"
	onclose={() => (showTargetPriceModal = false)}
>
	<form method="POST" action="?/updateTargetPrice" class="modal-form">
		<input type="hidden" name="productId" value={data.product?.id || ''} />
		<div class="form-group">
			<label for="edit-target-price-input" class="label">Target Purchase Price (₹)</label>
			<input
				id="edit-target-price-input"
				name="targetPrice"
				type="number"
				step="0.01"
				min="1"
				required
				class="input tabular-nums font-mono"
				bind:value={editTargetPriceVal}
			/>
		</div>
		<div class="modal-footer">
			<button
				type="button"
				class="btn btn-secondary"
				onclick={() => (showTargetPriceModal = false)}
			>
				Cancel
			</button>
			<button type="submit" class="btn btn-primary">Save Target Price</button>
		</div>
	</form>
</Modal>

<!-- =========================================================================
     MODAL 2: Manual Product Information Entry & Editing (Section 7)
     ========================================================================= -->
<Modal
	isOpen={showEditProductModal}
	title="Product Details & Specifications"
	onclose={() => (showEditProductModal = false)}
>
	<form method="POST" action="?/updateProductDetails" class="modal-form">
		<input type="hidden" name="productId" value={data.product?.id || ''} />

		<div class="form-group">
			<label for="edit-prod-name" class="label">Product Name *</label>
			<input id="edit-prod-name" name="name" required class="input" bind:value={editName} />
		</div>

		<div class="form-grid-3">
			<div class="form-group">
				<label for="edit-prod-brand" class="label">Brand</label>
				<input
					id="edit-prod-brand"
					name="brand"
					class="input"
					bind:value={editBrand}
					placeholder="e.g. Sony, Apple"
				/>
			</div>
			<div class="form-group">
				<label for="edit-prod-model" class="label">Model Number</label>
				<input
					id="edit-prod-model"
					name="model"
					class="input"
					bind:value={editModel}
					placeholder="e.g. WH-1000XM5"
				/>
			</div>
			<div class="form-group">
				<label for="edit-prod-variant" class="label">Variant / Specs</label>
				<input
					id="edit-prod-variant"
					name="variant"
					class="input"
					bind:value={editVariant}
					placeholder="e.g. Silver, 128GB"
				/>
			</div>
		</div>

		<div class="form-grid-3">
			<div class="form-group">
				<label for="edit-prod-current" class="label">Current Listed Price (₹)</label>
				<input
					id="edit-prod-current"
					name="currentPrice"
					type="number"
					step="0.01"
					min="0"
					class="input font-mono"
					bind:value={editCurrentPrice}
				/>
			</div>
			<div class="form-group">
				<label for="edit-prod-original" class="label">Original Price / MRP (₹)</label>
				<input
					id="edit-prod-original"
					name="originalPrice"
					type="number"
					step="0.01"
					min="0"
					class="input font-mono"
					bind:value={editOriginalPrice}
				/>
			</div>
			<div class="form-group">
				<label for="edit-prod-avail" class="label">Availability Status</label>
				<select
					id="edit-prod-avail"
					name="availability"
					class="input"
					bind:value={editAvailability}
				>
					<option value="in_stock">In Stock</option>
					<option value="out_of_stock">Out of Stock</option>
					<option value="pre_order">Pre-Order</option>
				</select>
			</div>
		</div>

		<div class="form-grid-2">
			<div class="form-group">
				<label for="edit-prod-retailer" class="label">Retailer Name</label>
				<input
					id="edit-prod-retailer"
					name="retailer"
					class="input"
					bind:value={editRetailer}
					placeholder="e.g. Amazon, Croma"
				/>
			</div>
			<div class="form-group">
				<label for="edit-prod-currency" class="label">Currency Code</label>
				<input
					id="edit-prod-currency"
					name="currency"
					class="input font-mono uppercase"
					bind:value={editCurrency}
					placeholder="INR"
				/>
			</div>
		</div>

		<div class="form-group">
			<label for="edit-prod-url" class="label">Product Page URL</label>
			<input
				id="edit-prod-url"
				name="url"
				type="url"
				class="input"
				bind:value={editUrl}
				placeholder="https://..."
			/>
		</div>

		<div class="form-group">
			<label for="edit-prod-img" class="label">Product Image URL</label>
			<input
				id="edit-prod-img"
				name="imageUrl"
				type="url"
				class="input"
				bind:value={editImageUrl}
				placeholder="https://...jpg"
			/>
		</div>

		<div class="form-group">
			<label for="edit-prod-specs" class="label">Technical Specifications (JSON)</label>
			<textarea
				id="edit-prod-specs"
				name="specifications"
				rows="4"
				class="input font-mono text-xs"
				bind:value={editSpecsJson}></textarea>
		</div>

		<div class="form-group">
			<label for="edit-prod-notes" class="label">Notes / Description</label>
			<input
				id="edit-prod-notes"
				name="notes"
				class="input"
				bind:value={editNotes}
				placeholder="Optional product notes or context"
			/>
		</div>

		<div class="modal-footer">
			<button type="button" class="btn btn-secondary" onclick={() => (showEditProductModal = false)}
				>Cancel</button
			>
			<button type="submit" class="btn btn-primary">Save Product Details</button>
		</div>
	</form>
</Modal>

<!-- =========================================================================
     MODAL 3: Add Manual Price Observation
     ========================================================================= -->
<Modal
	isOpen={showAddPriceModal}
	title="Record Price Observation"
	onclose={() => (showAddPriceModal = false)}
>
	<form method="POST" action="?/addManualPrice" class="modal-form">
		<input type="hidden" name="productId" value={data.product?.id || ''} />

		<div class="form-group">
			<label for="manual-obs-price" class="label">Observed Price (₹) *</label>
			<input
				id="manual-obs-price"
				name="price"
				type="number"
				step="0.01"
				min="1"
				required
				class="input font-mono tabular-nums"
				bind:value={manualObsPrice}
				placeholder="e.g. 28990"
			/>
		</div>

		<div class="form-grid-2">
			<div class="form-group">
				<label for="manual-obs-date" class="label">Observation Date *</label>
				<input
					id="manual-obs-date"
					name="observedOn"
					type="date"
					required
					class="input"
					bind:value={manualObsDate}
				/>
			</div>
			<div class="form-group">
				<label for="manual-obs-type" class="label">Price Type</label>
				<select id="manual-obs-type" name="priceType" class="input" bind:value={manualObsType}>
					<option value="regular">Regular Listed Price</option>
					<option value="discounted">Discounted / Sale Price</option>
					<option value="original">MRP / List Price</option>
				</select>
			</div>
		</div>

		<div class="form-grid-2">
			<div class="form-group">
				<label for="manual-obs-ret" class="label">Retailer / Seller</label>
				<input
					id="manual-obs-ret"
					name="retailer"
					class="input"
					bind:value={manualObsRetailer}
					placeholder="e.g. Amazon, Croma"
				/>
			</div>
			<div class="form-group">
				<label for="manual-obs-notes" class="label">Notes / Occasion</label>
				<input
					id="manual-obs-notes"
					name="notes"
					class="input"
					bind:value={manualObsNotes}
					placeholder="e.g. Prime Day sale"
				/>
			</div>
		</div>

		<div class="modal-footer">
			<button type="button" class="btn btn-secondary" onclick={() => (showAddPriceModal = false)}
				>Cancel</button
			>
			<button type="submit" class="btn btn-primary">Save Observation</button>
		</div>
	</form>
</Modal>

<!-- =========================================================================
     MODAL 4: Multi-Format Document Ingestion (Section 6)
     ========================================================================= -->
<Modal
	isOpen={showDocUploadModal}
	title="Import Product Document"
	onclose={() => (showDocUploadModal = false)}
>
	<div class="document-upload-workflow">
		{#if !docPreview}
			<!-- Stage 1: File Selection -->
			<div class="drop-zone">
				<UploadCloud size={40} class="text-accent mb-2" />
				<h4 class="font-bold">Select Document to Ingest</h4>
				<p class="text-xs text-muted mb-3">
					Supports CSV, Excel (.xlsx, .xls), PDF, Word (.docx), and Invoice Images (.png, .jpg).
				</p>

				<input
					type="file"
					id="doc-file-input"
					accept=".csv,.xlsx,.xls,.pdf,.docx,.doc,.png,.jpg,.jpeg,.webp"
					onchange={(e) => {
						const files = e.currentTarget.files;
						if (files && files[0]) docFile = files[0];
					}}
					class="file-input-hidden"
				/>
				<label for="doc-file-input" class="btn btn-secondary btn-sm"> Browse Local Files </label>

				{#if docFile}
					<div class="selected-file-badge mt-3">
						<FileText size={16} />
						<span class="font-mono text-xs"
							>{docFile.name} ({(docFile.size / 1024).toFixed(1)} KB)</span
						>
					</div>
				{/if}
			</div>

			{#if docUploadError}
				<div class="alert alert-danger mt-3">{docUploadError}</div>
			{/if}

			<div class="modal-footer mt-4">
				<button type="button" class="btn btn-secondary" onclick={() => (showDocUploadModal = false)}
					>Cancel</button
				>
				<button
					type="button"
					class="btn btn-primary"
					disabled={!docFile || isUploadingDoc}
					onclick={handleDocUpload}
				>
					{#if isUploadingDoc}
						<Clock size={14} class="animate-spin" />
						<span>Analyzing Document...</span>
					{:else}
						<span>Process & Preview</span>
					{/if}
				</button>
			</div>
		{:else}
			<!-- Stage 2: Preview, Validation & Identity Check -->
			<div class="doc-preview-container">
				<!-- Product Identity Mismatch Warning (Section 6.5) -->
				{#if docPreview.identityMatch && !docPreview.identityMatch.matches}
					<div class="alert alert-warning mb-3">
						<AlertTriangle size={18} />
						<div>{docPreview.identityMatch.warning}</div>
					</div>
				{/if}

				<div class="preview-metrics-row mb-3">
					<div class="stat-pill">
						<CheckCircle2 size={15} class="text-success" />
						<span>{docPreview.validRows.length} Valid Records</span>
					</div>
					{#if docPreview.duplicateCount > 0}
						<div class="stat-pill">
							<AlertTriangle size={15} class="text-warning" />
							<span>{docPreview.duplicateCount} Duplicate Dates Filtered</span>
						</div>
					{/if}
					{#if docPreview.errorRows.length > 0}
						<div class="stat-pill">
							<AlertCircle size={15} class="text-danger" />
							<span>{docPreview.errorRows.length} Invalid Lines Skipped</span>
						</div>
					{/if}
				</div>

				<!-- Preview Table of First 5 Observations -->
				<div class="preview-table-box mb-3">
					<table class="table text-xs">
						<thead>
							<tr>
								<th>Date (YYYY-MM-DD)</th>
								<th>Price</th>
								<th>Type</th>
								<th>Retailer</th>
							</tr>
						</thead>
						<tbody>
							{#each docPreview.validRows.slice(0, 5) as row, idx (idx)}
								<tr>
									<td class="font-mono">{row.observedOn}</td>
									<td class="font-mono tabular-nums text-accent">{formatRupees(row.pricePaise)}</td>
									<td><span class="badge badge-secondary text-xs">{row.priceType}</span></td>
									<td>{row.retailer || 'Document'}</td>
								</tr>
							{/each}
						</tbody>
					</table>
					{#if docPreview.validRows.length > 5}
						<div class="text-xs text-muted text-center py-1">
							+ {docPreview.validRows.length - 5} more observations in this document
						</div>
					{/if}
				</div>

				{#if importCompleted}
					<div class="alert alert-success">
						<Check size={16} />
						<span>Import confirmed! Refreshing price history...</span>
					</div>
				{/if}

				<div class="modal-footer">
					<button
						type="button"
						class="btn btn-secondary"
						disabled={isConfirmingImport}
						onclick={() => (docPreview = null)}
					>
						Back
					</button>
					<button
						type="button"
						class="btn btn-accent"
						disabled={isConfirmingImport || importCompleted || docPreview.validRows.length === 0}
						onclick={confirmDocImport}
					>
						{#if isConfirmingImport}
							<Clock size={14} class="animate-spin" />
							<span>Importing Records...</span>
						{:else}
							<span>Confirm & Import ({docPreview.validRows.length} records)</span>
						{/if}
					</button>
				</div>
			</div>
		{/if}
	</div>
</Modal>

<style>
	.goal-decision-page {
		max-width: 1200px;
		margin: 0 auto;
		display: flex;
		flex-direction: column;
		gap: 20px;
	}

	.page-nav {
		margin-bottom: -8px;
	}

	.back-link {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		color: var(--foreground-secondary);
		text-decoration: none;
		font-size: 0.875rem;
	}

	.back-link:hover {
		color: var(--primary);
	}

	/* Hero Banner */
	.decision-hero {
		padding: 24px;
		border-left: 5px solid var(--primary);
	}

	.hero-main {
		display: flex;
		flex-wrap: wrap;
		justify-content: space-between;
		align-items: flex-start;
		gap: 20px;
	}

	.hero-left {
		display: flex;
		align-items: flex-start;
		gap: 16px;
	}

	.hero-title {
		margin: 4px 0 6px 0;
		font-size: 1.5rem;
	}

	.hero-verdict {
		color: var(--accent);
		font-weight: 800;
	}

	.hero-desc {
		color: var(--foreground-secondary);
		margin: 0;
		font-size: 0.95rem;
	}

	.reassess-form {
		margin-top: 16px;
	}

	/* Product Profile Card */
	.product-profile-card {
		padding: 24px;
	}

	.product-profile-top {
		display: flex;
		justify-content: space-between;
		align-items: flex-start;
		gap: 24px;
		flex-wrap: wrap;
	}

	.product-info-left {
		display: flex;
		align-items: flex-start;
		gap: 18px;
		flex: 1;
		min-width: 300px;
	}

	.product-img-thumb {
		width: 90px;
		height: 90px;
		object-fit: contain;
		background: #ffffff;
		border-radius: var(--radius-md);
		border: 1px solid var(--border-subtle);
		padding: 6px;
	}

	.product-img-placeholder {
		width: 90px;
		height: 90px;
		display: flex;
		align-items: center;
		justify-content: center;
		background: var(--surface-secondary);
		border-radius: var(--radius-md);
		color: var(--foreground-muted);
	}

	.product-name {
		font-size: 1.25rem;
		margin: 6px 0;
		line-height: 1.3;
	}

	.badges-row {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 6px;
	}

	.product-external-link {
		display: inline-flex;
		align-items: center;
		gap: 4px;
		font-size: 0.75rem;
		color: var(--primary);
		text-decoration: none;
		margin-left: 6px;
	}

	.product-external-link:hover {
		text-decoration: underline;
	}

	.product-actions-right {
		display: flex;
		flex-wrap: wrap;
		gap: 8px;
		align-items: center;
	}

	.inline-form {
		display: inline-block;
	}

	/* Specs Section */
	.specs-section {
		margin-top: 16px;
		padding-top: 14px;
		border-top: 1px solid var(--border-subtle);
	}

	.specs-toggle-btn {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		background: none;
		border: none;
		color: var(--primary);
		cursor: pointer;
		padding: 0;
	}

	.specs-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
		gap: 8px;
		margin-top: 12px;
		padding: 12px;
		background: var(--surface-secondary);
		border-radius: var(--radius-md);
		font-size: 0.8rem;
	}

	.spec-key {
		color: var(--foreground-secondary);
		font-weight: 500;
	}

	.spec-val {
		color: var(--foreground);
		margin-left: 4px;
		font-weight: 600;
	}

	/* Metrics Grid */
	.price-metrics-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
		gap: 16px;
		margin-top: 20px;
		padding-top: 20px;
		border-top: 1px solid var(--border-subtle);
	}

	.metric-box {
		background: var(--surface-secondary);
		padding: 16px;
		border-radius: var(--radius-md);
	}

	.metric-label {
		display: block;
		font-size: 0.75rem;
		text-transform: uppercase;
		color: var(--foreground-secondary);
		letter-spacing: 0.05em;
		margin-bottom: 6px;
	}

	.metric-val {
		font-size: 1.5rem;
		font-weight: 800;
		line-height: 1;
	}

	.price-row-display {
		display: flex;
		align-items: baseline;
		gap: 8px;
		flex-wrap: wrap;
	}

	.original-price-strikethrough {
		text-decoration: line-through;
		color: var(--foreground-muted);
	}

	.progress-track-sm {
		height: 6px;
		background: var(--border-subtle);
		border-radius: 999px;
		margin: 8px 0;
		overflow: hidden;
	}

	.progress-fill-sm {
		height: 100%;
		background: var(--accent);
	}

	.metric-sub {
		display: flex;
		align-items: center;
		gap: 4px;
		font-size: 0.75rem;
		margin-top: 6px;
		color: var(--foreground-secondary);
	}

	.link-action {
		background: none;
		border: none;
		color: var(--primary);
		cursor: pointer;
		padding: 0;
		text-decoration: underline;
	}

	/* History Tabs */
	.history-tabs-container {
		margin-top: 24px;
		padding-top: 20px;
		border-top: 1px solid var(--border-subtle);
	}

	.history-tabs-nav {
		display: flex;
		gap: 8px;
		border-bottom: 1px solid var(--border-subtle);
		margin-bottom: 16px;
	}

	.history-tab-btn {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		background: none;
		border: none;
		border-bottom: 2px solid transparent;
		padding: 8px 14px;
		font-size: 0.85rem;
		color: var(--foreground-secondary);
		cursor: pointer;
		margin-bottom: -1px;
	}

	.history-tab-btn.active {
		color: var(--primary);
		border-bottom-color: var(--primary);
		font-weight: 700;
	}

	.chart-stats-row {
		display: flex;
		gap: 20px;
		margin-bottom: 14px;
		font-size: 0.8rem;
		color: var(--foreground-secondary);
	}

	/* Document List */
	.docs-list {
		display: flex;
		flex-direction: column;
		gap: 10px;
	}

	.doc-item {
		display: flex;
		align-items: center;
		gap: 14px;
		padding: 12px 16px;
		background: var(--surface-secondary);
	}

	.doc-info {
		flex: 1;
	}

	/* Link Product Card */
	.link-product-card {
		padding: 24px;
	}

	.link-header {
		display: flex;
		align-items: center;
		gap: 16px;
		margin-bottom: 20px;
	}

	.link-actions-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
		gap: 16px;
	}

	.link-option-card {
		background: var(--surface-secondary);
		padding: 18px;
		border-radius: var(--radius-md);
		border: 1px solid var(--border-subtle);
	}

	/* Modal Form Layouts */
	.form-grid-2 {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 12px;
	}

	.form-grid-3 {
		display: grid;
		grid-template-columns: 1fr 1fr 1fr;
		gap: 12px;
	}

	@media (max-width: 768px) {
		.form-grid-2,
		.form-grid-3 {
			grid-template-columns: 1fr;
		}
	}

	/* Dropzone */
	.drop-zone {
		border: 2px dashed var(--border-subtle);
		border-radius: var(--radius-md);
		padding: 30px;
		text-align: center;
		background: var(--surface-secondary);
	}

	.file-input-hidden {
		display: none;
	}

	.selected-file-badge {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		background: var(--surface);
		padding: 4px 10px;
		border-radius: var(--radius-sm);
	}

	.preview-metrics-row {
		display: flex;
		gap: 10px;
		flex-wrap: wrap;
	}

	.stat-pill {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		background: var(--surface-secondary);
		padding: 4px 10px;
		border-radius: var(--radius-sm);
		font-size: 0.75rem;
	}

	.preview-table-box {
		max-height: 220px;
		overflow-y: auto;
		border: 1px solid var(--border-subtle);
		border-radius: var(--radius-sm);
	}

	.alert {
		padding: 10px 14px;
		border-radius: var(--radius-sm);
		font-size: 0.85rem;
		display: flex;
		align-items: center;
		gap: 10px;
	}

	.alert-warning {
		background: #fef3c7;
		color: #92400e;
		border: 1px solid #fde68a;
	}

	.alert-danger {
		background: #fee2e2;
		color: #991b1b;
		border: 1px solid #fecaca;
	}

	.alert-success {
		background: #dcfce7;
		color: #166534;
		border: 1px solid #bbf7d0;
	}
</style>
