<script lang="ts">
	import { enhance } from '$app/forms';
	import { resolve } from '$app/paths';
	import { formatRupees } from '#lib/money.js';
	import { SvelteMap } from 'svelte/reactivity';
	import Modal from '#lib/components/Modal.svelte';
	import type { PageProps } from './$types';
	import type { Category, BudgetLine } from '#lib/server/api.js';
	import {
		PieChart,
		Plus,
		Search,
		Layers,
		AlertTriangle,
		Archive,
		RotateCcw,
		Edit3,
		Trash2
	} from '@lucide/svelte';

	let { data, form }: PageProps = $props();

	// Search & Filter state
	let searchQuery = $state('');
	let activeTab = $state<'active' | 'archived' | 'all'>('active');
	let typeFilter = $state<'all' | 'discretionary' | 'committed'>('all');

	// Modals state
	let showCreateModal = $state(false);
	let showEditModal = $state(false);
	let showArchiveModal = $state(false);
	let showDeleteModal = $state(false);

	// Create Form bindings
	let newName = $state('');
	let newType = $state<'discretionary' | 'committed'>('discretionary');
	let newMonthlyLimit = $state('');

	// Edit / Action targets
	let targetCategory = $state<Category | null>(null);
	let editName = $state('');
	let editType = $state<'discretionary' | 'committed'>('discretionary');

	let editLimit = $state('');
	let editSaving = $state(false);
	let editError = $state('');

	// Quick helper to categorize a Category
	function getCategoryType(cat: Category): 'discretionary' | 'committed' {
		if (cat.committed) return 'committed';
		return 'discretionary';
	}

	// Budget lines lookup map: categoryId -> BudgetLine
	const budgetLineMap = $derived.by(() => {
		const map = new SvelteMap<string, BudgetLine>();
		if (data.budgetReport?.lines) {
			for (const line of data.budgetReport.lines) {
				map.set(line.categoryId, line);
			}
		}
		return map;
	});

	// Metrics derived from data
	const budgetStats = $derived.by(() => {
		const report = data.budgetReport;
		const totalBudget = report?.totalLimitPaise ?? 0;
		const totalSpent = report?.totalSpentPaise ?? 0;
		const totalRemaining = report?.totalRemainingPaise ?? 0;
		const unbudgeted = report?.unbudgetedDiscretionaryPaise ?? 0;

		let withinCount = 0;
		let approachingCount = 0;
		let exceedingCount = 0;

		if (report?.lines) {
			for (const l of report.lines) {
				if (l.limitPaise <= 0) continue;
				const pct = (l.spentPaise / l.limitPaise) * 100;
				if (pct < 75) {
					withinCount++;
				} else if (pct <= 95) {
					approachingCount++;
				} else {
					exceedingCount++;
				}
			}
		}

		return {
			totalBudget,
			totalSpent,
			totalRemaining,
			unbudgeted,
			withinCount,
			approachingCount,
			exceedingCount,
			overallPct: totalBudget > 0 ? Math.min(100, Math.round((totalSpent / totalBudget) * 100)) : 0
		};
	});

	// Filtered categories
	const displayedCategories = $derived.by(() => {
		let list = data.categories || [];

		// Filter active / archived
		if (activeTab === 'active') {
			list = list.filter((c) => !c.archived);
		} else if (activeTab === 'archived') {
			list = list.filter((c) => !!c.archived);
		}

		// Filter by kind
		if (typeFilter !== 'all') {
			list = list.filter((c) => getCategoryType(c) === typeFilter);
		}

		// Search query
		if (searchQuery.trim()) {
			const q = searchQuery.toLowerCase();
			list = list.filter((c) => c.name.toLowerCase().includes(q));
		}

		return list;
	});

	function openEdit(cat: Category) {
		targetCategory = cat;
		editName = cat.name;
		editType = getCategoryType(cat);
		const limit = budgetLineMap.get(cat.id)?.limitPaise;
		editLimit = limit ? String(limit / 100) : '';
		editError = '';
		showEditModal = true;
	}

	function openArchiveConfirm(cat: Category) {
		targetCategory = cat;
		showArchiveModal = true;
	}

	function openDeleteConfirm(cat: Category) {
		targetCategory = cat;
		showDeleteModal = true;
	}
</script>

<svelte:head>
	<title>Budget & Categories · BuyWise</title>
</svelte:head>

<div class="budget-workspace">
	<!-- Top Page Header -->
	<div class="page-header">
		<div>
			<div class="workspace-pill">
				<PieChart size={14} />
				<span>Financial Management</span>
			</div>
			<h1 class="page-headline">Budget & Categories</h1>
			<p class="page-subheadline">
				Configure your spending categories, monthly limits, and real-time budget utilization.
			</p>
		</div>
		<div class="header-actions">
			<button type="button" class="btn btn-primary" onclick={() => (showCreateModal = true)}>
				<Plus size={18} />
				<span>Add Category</span>
			</button>
		</div>
	</div>

	<!-- Top Metric Overview Cards (Section 3.D) -->
	<div class="budget-overview-grid">
		<!-- Metric 1: Monthly Budget -->
		<div class="overview-card">
			<div class="card-kicker">Total Monthly Budget</div>
			<div class="card-stat tabular-nums">{formatRupees(budgetStats.totalBudget)}</div>
			<div class="stat-progress-track">
				<div
					class="stat-progress-fill"
					class:warning={budgetStats.overallPct >= 75 && budgetStats.overallPct <= 95}
					class:danger={budgetStats.overallPct > 95}
					style="width: {budgetStats.overallPct}%;"
				></div>
			</div>
			<div class="card-caption">
				<span>{budgetStats.overallPct}% allocated used</span>
			</div>
		</div>

		<!-- Metric 2: Total Spent -->
		<div class="overview-card">
			<div class="card-kicker">Total Spent This Month</div>
			<div class="card-stat tabular-nums text-danger">{formatRupees(budgetStats.totalSpent)}</div>
			<div class="card-caption">
				<span>Across {data.budgetReport?.lines?.length || 0} budgeted categories</span>
			</div>
		</div>

		<!-- Metric 3: Total Remaining -->
		<div class="overview-card">
			<div class="card-kicker">Discretionary Remaining</div>
			<div
				class="card-stat tabular-nums"
				class:text-success={budgetStats.totalRemaining >= 0}
				class:text-danger={budgetStats.totalRemaining < 0}
			>
				{formatRupees(budgetStats.totalRemaining)}
			</div>
			<div class="card-caption">
				<span
					>{budgetStats.totalRemaining >= 0
						? 'Safe buffer for planned goals'
						: 'Over-budget deficit'}</span
				>
			</div>
		</div>

		<!-- Metric 4: Limit Health & Thresholds -->
		<div class="overview-card health-card">
			<div class="card-kicker">Category Limits Health</div>
			<div class="threshold-pill-group">
				<div class="threshold-pill normal">
					<span class="threshold-dot"></span>
					<span class="threshold-val">{budgetStats.withinCount}</span>
					<span class="threshold-lbl">Normal (&lt;75%)</span>
				</div>
				<div class="threshold-pill warning">
					<span class="threshold-dot"></span>
					<span class="threshold-val">{budgetStats.approachingCount}</span>
					<span class="threshold-lbl">Warning (75–95%)</span>
				</div>
				<div class="threshold-pill danger">
					<span class="threshold-dot"></span>
					<span class="threshold-val">{budgetStats.exceedingCount}</span>
					<span class="threshold-lbl">Critical (&gt;95%)</span>
				</div>
			</div>
			{#if budgetStats.unbudgeted > 0}
				<div class="unbudgeted-note">
					Unbudgeted flexible spend: <strong>{formatRupees(budgetStats.unbudgeted)}</strong>
				</div>
			{/if}
		</div>
	</div>

	<!-- Workspace Toolbar: Tabs, Search, and Type Filter -->
	<div class="workspace-toolbar card">
		<div class="toolbar-left">
			<div class="tab-pill-group" role="tablist">
				<button
					type="button"
					class="tab-pill"
					class:active={activeTab === 'active'}
					onclick={() => (activeTab = 'active')}
				>
					Active Categories ({data.categories.filter((c) => !c.archived).length})
				</button>
				<button
					type="button"
					class="tab-pill"
					class:active={activeTab === 'archived'}
					onclick={() => (activeTab = 'archived')}
				>
					Archived ({data.categories.filter((c) => !!c.archived).length})
				</button>
				<button
					type="button"
					class="tab-pill"
					class:active={activeTab === 'all'}
					onclick={() => (activeTab = 'all')}
				>
					All ({data.categories.length})
				</button>
			</div>

			<div class="type-filter-group">
				<button
					type="button"
					class="filter-chip"
					class:active={typeFilter === 'all'}
					onclick={() => (typeFilter = 'all')}
				>
					All Types
				</button>
				<button
					type="button"
					class="filter-chip"
					class:active={typeFilter === 'discretionary'}
					onclick={() => (typeFilter = 'discretionary')}
				>
					Discretionary
				</button>
				<button
					type="button"
					class="filter-chip"
					class:active={typeFilter === 'committed'}
					onclick={() => (typeFilter = 'committed')}
				>
					Committed (Fixed)
				</button>
			</div>
		</div>

		<div class="toolbar-right">
			<div class="search-box">
				<Search size={16} class="search-icon" />
				<input
					type="search"
					placeholder="Search categories..."
					bind:value={searchQuery}
					class="input search-input"
				/>
			</div>
		</div>
	</div>

	<!-- Unified Category & Budget Table (Section 3.A & 3.C) -->
	<div class="table-container card">
		{#if displayedCategories.length === 0}
			<div class="empty-state">
				<div class="empty-icon-box">
					<Layers size={32} />
				</div>
				<h3 class="empty-title">No categories found</h3>
				<p class="empty-text">
					{searchQuery
						? `No results match "${searchQuery}"`
						: 'No categories configured in this view.'}
				</p>
				<button type="button" class="btn btn-primary" onclick={() => (showCreateModal = true)}>
					<Plus size={18} />
					<span>Add your first category</span>
				</button>
			</div>
		{:else}
			<table class="category-table">
				<thead>
					<tr>
						<th>Category</th>
						<th>Type</th>
						<th>Monthly Allocation</th>
						<th>Spent This Month</th>
						<th>Remaining</th>
						<th>Utilization</th>
						<th>Status</th>
						<th class="text-right">Actions</th>
					</tr>
				</thead>
				<tbody>
					{#each displayedCategories as cat (cat.id)}
						{@const catType = getCategoryType(cat)}
						{@const line = budgetLineMap.get(cat.id)}
						{@const spentPaise = line?.spentPaise ?? (data.spendingMap?.[cat.id] || 0)}
						{@const limitPaise = line?.limitPaise}
						{@const hasLimit = limitPaise !== undefined && limitPaise > 0}
						{@const remainingPaise = hasLimit ? limitPaise - spentPaise : null}
						{@const pct = hasLimit ? Math.round((spentPaise / limitPaise) * 100) : null}

						{@const txCount = data.txCountMap?.[cat.id] || 0}

						<tr class="category-row" class:row-archived={cat.archived}>
							<!-- Category Name -->
							<td class="name-cell">
								<div class="name-wrapper">
									<strong class="cat-name">{cat.name}</strong>
									{#if cat.archived}
										<span class="archived-tag">Archived</span>
									{/if}
								</div>
							</td>

							<!-- Category Type -->
							<td>
								{#if catType === 'committed'}
									<span class="type-pill committed">Committed</span>
								{:else}
									<span class="type-pill discretionary">Discretionary</span>
								{/if}
							</td>

							<!-- Monthly Budget Allocation -->
							<td class="allocation-cell">
								{#if catType === 'discretionary'}
									<div class="allocation-display">
										{#if hasLimit}
											<span class="tabular-nums font-semibold">{formatRupees(limitPaise)}</span>
										{:else}
											<span class="no-limit-text">No limit configured</span>
										{/if}
										<button
											type="button"
											class="inline-edit-trigger"
											title="Edit monthly budget"
											onclick={() => openEdit(cat)}
										>
											<Edit3 size={13} />
										</button>
									</div>
								{:else}
									<a class="fixed-obligation-label" href={resolve('/accounts')}
										>{formatRupees(
											data.money?.categories.find((c) => c.categoryId === cat.id)?.monthlyPaise ?? 0
										)} · Edit in My Money</a
									>
								{/if}
							</td>

							<!-- Amount Spent This Month -->
							<td class="tabular-nums font-mono">
								{spentPaise > 0 ? formatRupees(spentPaise) : '₹0'}
							</td>

							<!-- Remaining Amount -->
							<td class="tabular-nums font-mono">
								{#if remainingPaise !== null}
									<span class={remainingPaise >= 0 ? 'text-success' : 'text-danger font-bold'}>
										{formatRupees(remainingPaise)}
									</span>
								{:else}
									<span class="text-muted">—</span>
								{/if}
							</td>

							<!-- Utilization Progress & Badge -->
							<td>
								{#if pct !== null}
									<div class="util-wrapper">
										<div class="table-progress-track">
											<div
												class="table-progress-fill"
												class:warning={pct >= 75 && pct <= 95}
												class:danger={pct > 95}
												style="width: {Math.min(100, pct)}%;"
											></div>
										</div>
										<span
											class="badge"
											class:badge-success={pct < 75}
											class:badge-warning={pct >= 75 && pct <= 95}
											class:badge-danger={pct > 95}
										>
											{pct}%
										</span>
									</div>
								{:else}
									<span class="text-muted text-xs">Unbudgeted</span>
								{/if}
							</td>

							<!-- Category Enabled / Status -->
							<td>
								{#if cat.archived}
									<span class="status-indicator archived">Disabled</span>
								{:else}
									<span class="status-indicator active">Active</span>
								{/if}
							</td>

							<!-- Actions -->
							<td class="actions-cell text-right">
								<div class="action-buttons">
									<!-- Edit -->
									<button
										type="button"
										class="action-btn"
										title="Edit category details"
										onclick={() => openEdit(cat)}
									>
										<Edit3 size={15} />
									</button>

									<!-- Archive / Restore -->
									{#if cat.archived}
										<form method="POST" action="?/archiveCategory" class="action-form">
											<input type="hidden" name="id" value={cat.id} />
											<input type="hidden" name="name" value={cat.name} />
											<input type="hidden" name="kind" value={cat.kind} />
											<input
												type="hidden"
												name="committed"
												value={cat.committed ? 'true' : 'false'}
											/>
											<input type="hidden" name="archived" value="false" />
											<button type="submit" class="action-btn restore" title="Restore category">
												<RotateCcw size={15} />
											</button>
										</form>
									{:else}
										<button
											type="button"
											class="action-btn archive"
											title="Archive category"
											onclick={() => openArchiveConfirm(cat)}
										>
											<Archive size={15} />
										</button>
									{/if}

									<!-- Delete (only enabled if 0 transactions) -->
									{#if txCount === 0}
										<button
											type="button"
											class="action-btn delete"
											title="Permanently remove unused category"
											onclick={() => openDeleteConfirm(cat)}
										>
											<Trash2 size={15} />
										</button>
									{/if}
								</div>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		{/if}
	</div>
</div>

<!-- ==========================================================================
     Create Category Modal (Section 3.E)
     ========================================================================== -->
<Modal open={showCreateModal} title="Add New Category" onclose={() => (showCreateModal = false)}>
	<form method="POST" action="?/createCategory" class="modal-form">
		<div class="form-group">
			<label for="create-cat-name" class="label">Category Name *</label>
			<input
				id="create-cat-name"
				name="name"
				required
				maxlength="200"
				placeholder="e.g. Groceries, Streaming Services, Gym"
				class="input"
				bind:value={newName}
			/>
		</div>

		<div class="form-group">
			<label for="create-cat-type" class="label">Category Type *</label>
			<select id="create-cat-type" name="type" class="input" bind:value={newType}>
				<option value="discretionary">Discretionary Expense (Flexible spending)</option>
				<option value="committed">Committed Expense (Fixed: Rent, EMIs, Bills)</option>
			</select>
			<span class="field-hint">
				{newType === 'discretionary'
					? 'Eligible for monthly budget allocation and spending limit tracking.'
					: 'Fixed financial obligations are accounted for prior to discretionary savings.'}
			</span>
		</div>

		{#if newType === 'discretionary'}
			<div class="form-group">
				<label for="create-cat-limit" class="label">Monthly Spending Limit (₹)</label>
				<input
					id="create-cat-limit"
					name="monthlyLimit"
					type="number"
					step="0.01"
					min="0"
					placeholder="e.g. 6000 (Optional)"
					class="input"
					bind:value={newMonthlyLimit}
				/>
				<span class="field-hint">You can leave this blank to track spending without a cap.</span>
			</div>
		{/if}

		{#if form?.error}
			<p class="form-error" role="alert">{form.error}</p>
		{/if}

		<div class="modal-footer">
			<button type="button" class="btn btn-secondary" onclick={() => (showCreateModal = false)}>
				Cancel
			</button>
			<button type="submit" class="btn btn-primary"> Save Category </button>
		</div>
	</form>
</Modal>

<!-- ==========================================================================
     Edit Category Modal
     ========================================================================== -->
{#if targetCategory}
	<Modal
		open={showEditModal}
		title="Edit Category"
		onclose={() => {
			if (!editSaving) showEditModal = false;
		}}
	>
		<form
			method="POST"
			action="?/updateCategory"
			class="modal-form"
			use:enhance={({ cancel }) => {
				if (editSaving) {
					cancel();
					return;
				}
				editSaving = true;
				editError = '';
				return async ({ result, update }) => {
					try {
						if (result.type === 'success') {
							await update();
							showEditModal = false;
						} else {
							editError =
								result.type === 'failure' && typeof result.data?.error === 'string'
									? result.data.error
									: 'Could not save the category. Please try again.';
						}
					} finally {
						editSaving = false;
					}
				};
			}}
		>
			<input type="hidden" name="id" value={targetCategory.id} />
			<input type="hidden" name="archived" value={targetCategory.archived ? 'true' : 'false'} />

			<div class="form-group">
				<label for="edit-cat-name" class="label">Category Name *</label>
				<input
					id="edit-cat-name"
					name="name"
					required
					maxlength="200"
					class="input"
					bind:value={editName}
				/>
			</div>

			<div class="form-group">
				<label for="edit-cat-type" class="label">Category Type</label>
				<select id="edit-cat-type" name="type" class="input" bind:value={editType}>
					<option value="discretionary">Discretionary Expense</option>
					<option value="committed">Committed Expense</option>
				</select>
			</div>

			{#if editType === 'discretionary'}
				<div class="form-group">
					<label for="edit-cat-limit" class="label">Monthly budget (INR)</label>
					<input
						id="edit-cat-limit"
						name="monthlyLimit"
						class="input"
						inputmode="decimal"
						bind:value={editLimit}
						disabled={!data.budgetReport || editSaving}
						aria-describedby="edit-limit-help"
					/>
					<small id="edit-limit-help"
						>{data.budgetReport
							? 'Leave blank to remove the monthly limit.'
							: 'Reload to edit the limit once budget data is available.'}</small
					>
				</div>
			{/if}
			{#if editError}<p role="alert" class="text-danger">{editError}</p>{/if}
			<div class="modal-footer">
				<button
					type="button"
					class="btn btn-secondary"
					onclick={() => (showEditModal = false)}
					disabled={editSaving}
				>
					Cancel
				</button>
				<button type="submit" class="btn btn-primary" disabled={editSaving}
					>{editSaving ? 'Saving...' : 'Save Changes'}</button
				>
			</div>
		</form>
	</Modal>
{/if}

<!-- ==========================================================================
     Archive Confirmation Modal (Section 3.B)
     ========================================================================== -->
{#if targetCategory}
	<Modal
		open={showArchiveModal}
		title="Archive Category"
		onclose={() => (showArchiveModal = false)}
	>
		<div class="archive-modal-content">
			<div class="warning-callout">
				<AlertTriangle size={20} class="warning-icon" />
				<div>
					<strong>Confirm Archival: {targetCategory.name}</strong>
					<p class="callout-desc">
						This category has <strong
							>{data.txCountMap?.[targetCategory.id] || 0} existing transactions</strong
						>.
					</p>
				</div>
			</div>

			<p class="modal-explanation">
				Archiving this category will hide it from active entry and future budget reporting. All
				historical transactions, accounts, and historical reports remain intact. No transactions
				will be deleted or moved. You can restore this category at any time.
			</p>

			<form method="POST" action="?/archiveCategory" class="modal-form">
				<input type="hidden" name="id" value={targetCategory.id} />
				<input type="hidden" name="name" value={targetCategory.name} />
				<input type="hidden" name="kind" value={targetCategory.kind} />
				<input type="hidden" name="committed" value={targetCategory.committed ? 'true' : 'false'} />
				<input type="hidden" name="archived" value="true" />

				<div class="modal-footer">
					<button
						type="button"
						class="btn btn-secondary"
						onclick={() => (showArchiveModal = false)}
					>
						Cancel
					</button>
					<button type="submit" class="btn btn-danger"> Confirm & Archive Category </button>
				</div>
			</form>
		</div>
	</Modal>
{/if}

<!-- ==========================================================================
     Delete Confirmation Modal (Only for 0 transactions)
     ========================================================================== -->
{#if targetCategory}
	<Modal open={showDeleteModal} title="Delete Category" onclose={() => (showDeleteModal = false)}>
		<div class="delete-modal-content">
			<p class="modal-explanation">
				Are you sure you want to permanently delete <strong>{targetCategory.name}</strong>? This
				category has no transactions and will be removed permanently.
			</p>

			<form method="POST" action="?/deleteCategory" class="modal-form">
				<input type="hidden" name="id" value={targetCategory.id} />
				<div class="modal-footer">
					<button type="button" class="btn btn-secondary" onclick={() => (showDeleteModal = false)}>
						Cancel
					</button>
					<button type="submit" class="btn btn-danger"> Permanently Delete </button>
				</div>
			</form>
		</div>
	</Modal>
{/if}

<style>
	.budget-workspace {
		max-width: 1400px;
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

	.workspace-pill {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		padding: 4px 10px;
		border-radius: 9999px;
		font-size: 0.78rem;
		font-weight: 600;
		background-color: var(--primary-subtle);
		color: var(--primary);
		margin-bottom: 8px;
	}

	.page-headline {
		font-size: 28px;
		font-weight: 700;
		color: var(--foreground);
		letter-spacing: -0.025em;
		margin-bottom: 4px;
	}

	.page-subheadline {
		font-size: 15px;
		color: var(--foreground-secondary);
	}

	/* Overview Grid */
	.budget-overview-grid {
		display: grid;
		grid-template-columns: repeat(4, 1fr);
		gap: 18px;
	}

	.overview-card {
		background-color: var(--surface);
		border: 1px solid var(--border);
		border-radius: 16px;
		padding: 20px 22px;
		display: flex;
		flex-direction: column;
		justify-content: space-between;
		box-shadow: var(--shadow-sm);
	}

	.card-kicker {
		font-size: 0.82rem;
		font-weight: 600;
		color: var(--foreground-secondary);
		margin-bottom: 8px;
		text-transform: uppercase;
		letter-spacing: 0.04em;
	}

	.card-stat {
		font-size: 1.85rem;
		font-weight: 700;
		color: var(--foreground);
		margin-bottom: 12px;
		line-height: 1.1;
	}

	.card-caption {
		font-size: 0.8rem;
		color: var(--foreground-muted);
	}

	.stat-progress-track {
		width: 100%;
		height: 6px;
		background-color: var(--surface-secondary);
		border-radius: 9999px;
		overflow: hidden;
		margin-bottom: 8px;
	}

	.stat-progress-fill {
		height: 100%;
		background-color: var(--accent);
		border-radius: 9999px;
		transition: width 0.3s ease;
	}

	.stat-progress-fill.warning {
		background-color: var(--warning);
	}

	.stat-progress-fill.danger {
		background-color: var(--danger);
	}

	/* Health Thresholds Card */
	.threshold-pill-group {
		display: flex;
		flex-direction: column;
		gap: 6px;
		margin-bottom: 8px;
	}

	.threshold-pill {
		display: flex;
		align-items: center;
		gap: 8px;
		font-size: 0.82rem;
		font-weight: 500;
	}

	.threshold-dot {
		width: 8px;
		height: 8px;
		border-radius: 50%;
	}

	.threshold-pill.normal .threshold-dot {
		background-color: var(--success);
	}
	.threshold-pill.warning .threshold-dot {
		background-color: var(--warning);
	}
	.threshold-pill.danger .threshold-dot {
		background-color: var(--danger);
	}

	.threshold-val {
		font-family: var(--font-mono);
		font-weight: 700;
		color: var(--foreground);
	}

	.threshold-lbl {
		color: var(--foreground-secondary);
		font-size: 0.78rem;
	}

	.unbudgeted-note {
		font-size: 0.78rem;
		color: var(--foreground-muted);
		border-top: 1px solid var(--border-subtle);
		padding-top: 8px;
	}

	/* Toolbar */
	.workspace-toolbar {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 16px;
		flex-wrap: wrap;
		padding: 16px 20px;
	}

	.toolbar-left {
		display: flex;
		align-items: center;
		gap: 16px;
		flex-wrap: wrap;
	}

	.tab-pill-group {
		display: inline-flex;
		background-color: var(--surface-secondary);
		padding: 3px;
		border-radius: 10px;
	}

	.tab-pill {
		padding: 6px 14px;
		border-radius: 8px;
		font-size: 0.84rem;
		font-weight: 500;
		color: var(--foreground-secondary);
		transition: all 0.15s ease;
	}

	.tab-pill.active {
		background-color: var(--surface);
		color: var(--primary);
		font-weight: 600;
		box-shadow: var(--shadow-sm);
	}

	.type-filter-group {
		display: flex;
		gap: 6px;
		flex-wrap: wrap;
	}

	.filter-chip {
		padding: 5px 12px;
		border-radius: 8px;
		font-size: 0.8rem;
		font-weight: 500;
		border: 1px solid var(--border);
		background-color: var(--surface);
		color: var(--foreground-secondary);
		transition: all 0.15s ease;
	}

	.filter-chip.active {
		border-color: var(--primary);
		background-color: var(--primary-subtle);
		color: var(--primary);
		font-weight: 600;
	}

	.search-box {
		position: relative;
		display: flex;
		align-items: center;
	}

	:global(.search-icon) {
		position: absolute;
		left: 12px;
		color: var(--foreground-muted);
		pointer-events: none;
	}

	.search-input {
		padding-left: 36px;
		width: 220px;
	}

	/* Table */
	.table-container {
		padding: 0;
		overflow-x: auto;
	}

	.category-table {
		width: 100%;
		border-collapse: collapse;
		text-align: left;
		font-size: 0.9rem;
	}

	.category-table th {
		background-color: var(--surface-secondary);
		color: var(--foreground-secondary);
		font-weight: 600;
		font-size: 0.8rem;
		text-transform: uppercase;
		letter-spacing: 0.04em;
		padding: 14px 20px;
		border-bottom: 1px solid var(--border);
	}

	.category-row {
		border-bottom: 1px solid var(--border-subtle);
		transition: background-color 0.15s ease;
	}

	.category-row:hover {
		background-color: var(--surface-hover);
	}

	.category-row.row-archived {
		opacity: 0.65;
	}

	.category-table td {
		padding: 16px 20px;
		color: var(--foreground);
		vertical-align: middle;
	}

	.name-cell {
		font-weight: 500;
	}

	.name-wrapper {
		display: flex;
		align-items: center;
		gap: 8px;
	}

	.archived-tag {
		font-size: 0.72rem;
		padding: 2px 6px;
		background-color: var(--surface-secondary);
		color: var(--foreground-muted);
		border-radius: 4px;
		font-weight: 500;
	}

	.type-pill {
		display: inline-block;
		padding: 3px 8px;
		border-radius: 6px;
		font-size: 0.75rem;
		font-weight: 600;
	}

	.type-pill.discretionary {
		background-color: var(--primary-subtle);
		color: var(--primary);
	}

	.type-pill.committed {
		background-color: var(--surface-secondary);
		color: var(--foreground-secondary);
		border: 1px solid var(--border);
	}

	.allocation-display {
		display: flex;
		align-items: center;
		gap: 8px;
	}

	.no-limit-text {
		color: var(--foreground-muted);
		font-style: italic;
		font-size: 0.85rem;
	}

	.inline-edit-trigger {
		opacity: 0.5;
		color: var(--foreground-secondary);
		padding: 4px;
		border-radius: 4px;
		transition: opacity 0.15s ease;
	}

	.allocation-display:hover .inline-edit-trigger {
		opacity: 1;
	}

	.fixed-obligation-label {
		color: var(--foreground-muted);
		font-size: 0.85rem;
	}

	.util-wrapper {
		display: flex;
		align-items: center;
		gap: 10px;
	}

	.table-progress-track {
		width: 70px;
		height: 6px;
		background-color: var(--surface-secondary);
		border-radius: 9999px;
		overflow: hidden;
	}

	.table-progress-fill {
		height: 100%;
		background-color: var(--accent);
		border-radius: 9999px;
	}

	.table-progress-fill.warning {
		background-color: var(--warning);
	}

	.table-progress-fill.danger {
		background-color: var(--danger);
	}

	.status-indicator {
		display: inline-flex;
		align-items: center;
		font-size: 0.78rem;
		font-weight: 500;
	}

	.status-indicator.active {
		color: var(--success);
	}

	.status-indicator.archived {
		color: var(--foreground-muted);
	}

	.action-buttons {
		display: inline-flex;
		align-items: center;
		gap: 4px;
	}

	.action-btn {
		width: 32px;
		height: 32px;
		border-radius: 8px;
		display: inline-flex;
		align-items: center;
		justify-content: center;
		color: var(--foreground-secondary);
		transition: all 0.15s ease;
	}

	.action-btn:hover {
		background-color: var(--surface-secondary);
		color: var(--foreground);
	}

	.action-btn.archive:hover {
		background-color: var(--warning-surface);
		color: var(--warning);
	}

	.action-btn.restore:hover {
		background-color: var(--success-surface);
		color: var(--success);
	}

	.action-btn.delete:hover {
		background-color: var(--danger-surface);
		color: var(--danger);
	}

	.action-form {
		margin: 0;
		padding: 0;
	}

	/* Modals */
	.modal-form {
		display: flex;
		flex-direction: column;
		gap: 16px;
	}

	.modal-explanation {
		font-size: 0.9rem;
		line-height: 1.5;
		color: var(--foreground-secondary);
		margin-bottom: 16px;
	}

	.warning-callout {
		display: flex;
		align-items: flex-start;
		gap: 12px;
		padding: 14px;
		border-radius: 10px;
		background-color: var(--warning-surface);
		color: var(--warning-foreground);
		margin-bottom: 16px;
	}

	.callout-desc {
		font-size: 0.85rem;
		margin-top: 4px;
	}

	.field-hint {
		font-size: 0.78rem;
		color: var(--foreground-muted);
		margin-top: 2px;
	}

	.form-error {
		color: var(--danger);
		font-size: 0.85rem;
		font-weight: 500;
	}

	@media (max-width: 1024px) {
		.budget-overview-grid {
			grid-template-columns: repeat(2, 1fr);
		}
	}

	@media (max-width: 640px) {
		.budget-overview-grid {
			grid-template-columns: 1fr;
		}
		.toolbar-left {
			width: 100%;
			flex-direction: column;
			align-items: flex-start;
		}
		.search-input {
			width: 100%;
		}
	}
</style>
