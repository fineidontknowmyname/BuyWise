<script lang="ts">
	import { resolve } from '$app/paths';
	import { formatRupees } from '#lib/money.js';
	import FinancialChart from '#lib/components/FinancialChart.svelte';
	import Modal from '#lib/components/Modal.svelte';
	import QuickExpenseModal from '#lib/components/QuickExpenseModal.svelte';
	import type { PageProps } from './$types';
	import {
		Wallet,
		Target,
		Plus,
		ArrowRight,
		ShieldCheck,
		Sparkles,
		TrendingUp,
		TrendingDown,
		Calendar,
		Clock
	} from '@lucide/svelte';

	let { data, form }: PageProps = $props();

	let showGoalModal = $state(false);
	let showQuickExpenseModal = $state(false);

	const labels = { up: 'Up', down: 'Down', unknown: 'Unknown' };

	// Dynamic time-aware greeting
	const greeting = $derived.by(() => {
		const hour = new Date().getHours();
		if (hour < 12) return 'Good morning';
		if (hour < 17) return 'Good afternoon';
		return 'Good evening';
	});

	// Current reporting period label
	const currentReportingPeriod = $derived.by(() => {
		const d = new Date();
		return d.toLocaleDateString('en-US', { month: 'long', year: 'numeric' });
	});

	// Budget calculations
	const budgetSpent = $derived(data.budgetReport?.totalSpentPaise ?? 0);
	const budgetLimit = $derived(data.budgetReport?.totalLimitPaise ?? 0);
	const budgetRemaining = $derived(data.budgetReport?.totalRemainingPaise ?? 0);

	const budgetUtilization = $derived(
		budgetLimit > 0 ? Math.min(100, Math.round((budgetSpent / budgetLimit) * 100)) : 0
	);

	const categoryMap = $derived(new Map(data.categories.map((c) => [c.id, c.name])));

	// Goal Modal URL extraction bindings
	let newGoalName = $state('');
	let newGoalUrl = $state('');
	let isAnalyzingUrl = $state(false);
	let extractUrlError = $state('');

	async function analyzeGoalUrl() {
		if (!newGoalUrl.trim()) return;
		isAnalyzingUrl = true;
		extractUrlError = '';
		try {
			const res = await fetch(resolve('/api/products/extract'), {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ url: newGoalUrl.trim() })
			});
			const d = await res.json();
			if (res.ok) {
				if (d.name && !newGoalName) newGoalName = `Buy ${d.name}`;
			} else {
				extractUrlError = d.error || 'Failed to analyze URL.';
			}
		} catch (err: unknown) {
			extractUrlError = err instanceof Error ? err.message : 'Network error.';
		} finally {
			isAnalyzingUrl = false;
		}
	}
</script>

<svelte:head>
	<title>BuyWise — Overview Dashboard</title>
</svelte:head>

<div class="dashboard-page">
	<!-- Top Section (Section 6.D) -->
	<div class="page-header">
		<div>
			<div class="period-pill">
				<Calendar size={13} />
				<span>Reporting Period: {currentReportingPeriod}</span>
			</div>
			<h1 class="page-headline">{greeting}</h1>
			<p class="page-subheadline">
				Here is your centralized financial overview and purchase readiness summary.
			</p>
		</div>
		<div class="header-actions">
			<button
				type="button"
				class="btn btn-secondary"
				onclick={() => (showQuickExpenseModal = true)}
			>
				<Sparkles size={16} class="text-accent" />
				<span>AI Quick-Add Expense</span>
			</button>
			<button type="button" class="btn btn-primary" onclick={() => (showGoalModal = true)}>
				<Plus size={18} />
				<span>New Purchase Goal</span>
			</button>
		</div>
	</div>

	<!-- System Diagnostic Health Bar (Preserving exact test IDs) -->
	<div class="health-banner">
		<div class="health-item">
			<span class="health-dot" class:healthy={data.health.api === 'up'}></span>
			<span class="health-label">API:</span>
			<strong class="health-val" data-testid="api-status">{labels[data.health.api]}</strong>
		</div>
		<div class="health-sep"></div>
		<div class="health-item">
			<span class="health-dot" class:healthy={data.health.database === 'up'}></span>
			<span class="health-label">Database:</span>
			<strong class="health-val" data-testid="database-status"
				>{labels[data.health.database]}</strong
			>
		</div>
		{#if data.health.api === 'up' && data.health.database === 'up'}
			<div class="health-badge-ok">
				<ShieldCheck size={14} />
				<span>Core Services Operational</span>
			</div>
		{/if}
	</div>

	<!-- 4 Summary Metric Cards (Section 6.D: FinSet Visual System) -->
	<div class="stat-grid">
		<!-- Card 1: Total Liquid Balance -->
		<div class="stat-card">
			<div class="stat-top">
				<span class="stat-label">Total Liquid Balance</span>
				<div class="stat-icon-wrapper">
					<Wallet size={20} />
				</div>
			</div>
			<div class="stat-value tabular-nums">{formatRupees(data.totalLiquidBalancePaise)}</div>
			<div class="stat-footer">
				<span class="footer-link">
					{data.accounts.length} verified {data.accounts.length === 1 ? 'account' : 'accounts'}
				</span>
			</div>
		</div>

		<!-- Card 2: Monthly Income -->
		<div class="stat-card">
			<div class="stat-top">
				<span class="stat-label">Monthly Income</span>
				<div class="stat-icon-wrapper success">
					<TrendingUp size={20} />
				</div>
			</div>
			<div class="stat-value tabular-nums text-success">
				{formatRupees(data.monthlyIncomePaise)}
			</div>
			<div class="stat-footer">
				<span class="text-xs text-secondary">Inflows recorded this month</span>
			</div>
		</div>

		<!-- Card 3: Monthly Expenses -->
		<div class="stat-card">
			<div class="stat-top">
				<span class="stat-label">Monthly Expenses</span>
				<div class="stat-icon-wrapper danger">
					<TrendingDown size={20} />
				</div>
			</div>
			<div class="stat-value tabular-nums text-danger">
				{formatRupees(data.monthlyExpensePaise)}
			</div>
			<div class="stat-footer">
				<span class="text-xs text-secondary">Outflows across all categories</span>
			</div>
		</div>

		<!-- Card 4: Total Goal Savings / Progress -->
		<div class="stat-card">
			<div class="stat-top">
				<span class="stat-label">Total Goal Savings</span>
				<div class="stat-icon-wrapper accent">
					<Target size={20} />
				</div>
			</div>
			<div class="stat-value tabular-nums text-primary">{formatRupees(data.totalSavingsPaise)}</div>
			<div class="stat-footer">
				<a href={resolve('/goals')} class="footer-link-action">
					Across {data.goals.length}
					{data.goals.length === 1 ? 'goal' : 'goals'}
					<ArrowRight size={13} />
				</a>
			</div>
		</div>
	</div>

	<!-- Primary Content Section: Income vs Expense Chart & Budget Utilization (Section 6.D) -->
	<div class="analytics-grid">
		<!-- Left: Income-vs-Expense Chart with Week, Month, and Year tabs -->
		<div class="chart-col">
			<FinancialChart
				spentPaise={data.monthlyExpensePaise || budgetSpent}
				limitPaise={data.monthlyIncomePaise || budgetLimit}
				remainingPaise={budgetRemaining}
				title="Income vs. Expense Flow"
			/>
		</div>

		<!-- Right: Budget Utilization Card -->
		<div class="budget-col card">
			<div class="card-header">
				<div>
					<h3 class="card-title">Budget Utilization</h3>
					<p class="card-desc">Monthly spending limit thresholds</p>
				</div>
				<a href={resolve('/budget')} class="btn btn-ghost btn-sm">
					Manage <ArrowRight size={14} />
				</a>
			</div>

			<div class="budget-overview-box">
				<div class="budget-big-number">
					<span class="budget-spent-val tabular-nums">{formatRupees(budgetSpent)}</span>
					<span class="budget-limit-sub">of {formatRupees(budgetLimit)} limit</span>
				</div>

				<div class="budget-progress-track">
					<div
						class="budget-progress-fill"
						class:warning={budgetUtilization >= 75 && budgetUtilization <= 95}
						class:danger={budgetUtilization > 95}
						style="width: {budgetUtilization}%;"
					></div>
				</div>

				<div class="threshold-indicators">
					<div class="thresh-item" class:active={budgetUtilization < 75}>
						<span class="thresh-dot green"></span>
						<span class="text-xs">Normal (&lt;75%)</span>
					</div>
					<div
						class="thresh-item"
						class:active={budgetUtilization >= 75 && budgetUtilization <= 95}
					>
						<span class="thresh-dot amber"></span>
						<span class="text-xs">Warning (75–95%)</span>
					</div>
					<div class="thresh-item" class:active={budgetUtilization > 95}>
						<span class="thresh-dot red"></span>
						<span class="text-xs">Critical (&gt;95%)</span>
					</div>
				</div>

				<div class="discretionary-info-box">
					<span class="text-xs text-secondary">Discretionary Remaining:</span>
					<strong
						class="tabular-nums font-mono"
						class:text-success={budgetRemaining >= 0}
						class:text-danger={budgetRemaining < 0}
					>
						{formatRupees(budgetRemaining)}
					</strong>
				</div>
			</div>
		</div>
	</div>

	<!-- Lower Grid: Active Purchase Goals & Recent Activity -->
	<div class="lower-grid">
		<!-- Active Purchase Goals -->
		<div class="goals-col card">
			<div class="card-header">
				<div>
					<h3 class="card-title">Active Purchase Goals</h3>
					<p class="card-desc">Milestone targets and affordability readiness</p>
				</div>
				<a href={resolve('/goals')} class="btn btn-ghost btn-sm">
					View all <ArrowRight size={14} />
				</a>
			</div>

			{#if data.goals.length === 0}
				<div class="empty-state">
					<div class="empty-icon-box">
						<Target size={28} />
					</div>
					<h4 class="empty-title">No purchase goals yet</h4>
					<p class="empty-text">Create your first goal to evaluate whether you are ready to buy.</p>
					<button
						type="button"
						class="btn btn-primary btn-sm mt-3"
						onclick={() => (showGoalModal = true)}
					>
						+ Create goal
					</button>
				</div>
			{:else}
				<div class="goals-list">
					{#each data.goals.slice(0, 4) as goal (goal.id)}
						{@const target = goal.targetPricePaise || 0}
						{@const saved = goal.savedAmountPaise || 0}
						{@const pct = target > 0 ? Math.min(100, Math.round((saved / target) * 100)) : 0}

						<a href={resolve('/goals/[id]', { id: goal.id })} class="goal-item-card">
							<div class="goal-item-left">
								<div class="goal-icon-badge">
									<Target size={16} />
								</div>
								<div>
									<div class="goal-name">{goal.name}</div>
									<div class="goal-progress-numbers text-xs">
										<span class="tabular-nums">{formatRupees(saved)}</span> /
										<span class="tabular-nums">{formatRupees(target)}</span>
										({pct}%)
									</div>
									<div class="goal-track-mini">
										<div class="goal-fill-mini" style="width: {pct}%;"></div>
									</div>
								</div>
							</div>
							<div class="goal-item-right">
								<span class="badge badge-primary">{goal.priority || 'medium'}</span>
								<ArrowRight size={16} class="goal-arrow" />
							</div>
						</a>
					{/each}
				</div>
			{/if}
		</div>

		<!-- Recent Activity / Transactions -->
		<div class="activity-col card">
			<div class="card-header">
				<div>
					<h3 class="card-title">Recent Activity</h3>
					<p class="card-desc">Latest posted entries in your accounts</p>
				</div>
				<a href={resolve('/accounts')} class="btn btn-ghost btn-sm">
					View accounts <ArrowRight size={14} /></a
				>
			</div>

			{#if data.recentTransactions.length === 0}
				<div class="empty-state">
					<div class="empty-icon-box">
						<Wallet size={28} />
					</div>
					<h4 class="empty-title">No recent activity</h4>
					<p class="empty-text">Add transactions to your accounts to see cash flow activity.</p>
				</div>
			{:else}
				<div class="table-container">
					<table class="table">
						<thead>
							<tr>
								<th>Date</th>
								<th>Description</th>
								<th>Category</th>
								<th class="text-right">Amount</th>
							</tr>
						</thead>
						<tbody>
							{#each data.recentTransactions as tx (tx.id)}
								<tr>
									<td class="font-mono text-xs">{tx.occurredOn}</td>
									<td class="font-medium">{tx.note || 'Transaction'}</td>
									<td>
										<span class="badge badge-neutral text-xs">
											{categoryMap.get(tx.categoryId) ?? 'General'}
										</span>
									</td>
									<td class="text-right font-mono tabular-nums">
										<span
											class="amount-val"
											class:text-success={tx.kind === 'income'}
											class:text-danger={tx.kind === 'expense'}
										>
											{tx.kind === 'income' ? '+' : '-'}{formatRupees(tx.amountPaise)}
										</span>
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			{/if}
		</div>
	</div>
</div>

<!-- ==========================================================================
     New Goal Modal (Section 4: Direct Product URL Extraction)
     ========================================================================== -->
<Modal isOpen={showGoalModal} title="New Purchase Goal" onclose={() => (showGoalModal = false)}>
	<form method="POST" action="?/createGoal" class="modal-form">
		<div class="form-group">
			<label for="create-goal-name" class="label">Goal Name *</label>
			<input
				id="create-goal-name"
				name="name"
				required
				maxlength="200"
				placeholder="e.g. Ergonomic Office Chair, MacBook Pro"
				class="input"
				bind:value={newGoalName}
			/>
		</div>

		<div class="form-group">
			<label for="create-goal-url" class="label">Product URL (Optional)</label>
			<div class="url-input-row">
				<input
					id="create-goal-url"
					type="url"
					placeholder="https://www.amazon.in/dp/... or store URL"
					class="input"
					bind:value={newGoalUrl}
				/>
				<button
					type="button"
					class="btn btn-secondary btn-sm"
					disabled={!newGoalUrl.trim() || isAnalyzingUrl}
					onclick={analyzeGoalUrl}
				>
					{#if isAnalyzingUrl}
						<Clock size={14} class="animate-spin" />
						<span>Analyzing...</span>
					{:else}
						<Sparkles size={14} />
						<span>Analyze</span>
					{/if}
				</button>
			</div>
			{#if extractUrlError}
				<span class="text-xs text-danger mt-1">{extractUrlError}</span>
			{/if}
		</div>

		{#if form?.error}
			<p class="form-error" role="alert">{form.error}</p>
		{/if}

		<div class="modal-footer">
			<button type="button" class="btn btn-secondary" onclick={() => (showGoalModal = false)}>
				Cancel
			</button>
			<button type="submit" class="btn btn-primary"> Save Goal </button>
		</div>
	</form>
</Modal>

<!-- Quick Expense Modal -->
<QuickExpenseModal
	isOpen={showQuickExpenseModal}
	accounts={data.accounts}
	categories={data.categories}
	onclose={() => (showQuickExpenseModal = false)}
/>

<style>
	.dashboard-page {
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

	.period-pill {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		padding: 4px 10px;
		border-radius: 9999px;
		font-size: 0.78rem;
		font-weight: 600;
		background-color: var(--surface-secondary);
		color: var(--foreground-secondary);
		margin-bottom: 8px;
		border: 1px solid var(--border);
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

	/* Health Banner */
	.health-banner {
		display: flex;
		align-items: center;
		gap: 16px;
		background-color: var(--surface);
		border: 1px solid var(--border);
		border-radius: 12px;
		padding: 10px 18px;
		font-size: 0.84rem;
	}

	.health-item {
		display: flex;
		align-items: center;
		gap: 8px;
	}

	.health-dot {
		width: 8px;
		height: 8px;
		border-radius: 50%;
		background-color: var(--danger);
	}

	.health-dot.healthy {
		background-color: var(--success);
	}

	.health-sep {
		width: 1px;
		height: 16px;
		background-color: var(--border);
	}

	.health-badge-ok {
		margin-left: auto;
		display: flex;
		align-items: center;
		gap: 6px;
		color: var(--success);
		font-size: 0.8rem;
		font-weight: 600;
	}

	/* Stat Grid (4 Summary Cards - Section 6.D) */
	.stat-grid {
		display: grid;
		grid-template-columns: repeat(4, 1fr);
		gap: 20px;
	}

	.stat-card {
		background-color: var(--surface);
		border: 1px solid var(--border);
		border-radius: 16px;
		padding: 20px 22px;
		display: flex;
		flex-direction: column;
		justify-content: space-between;
		box-shadow: var(--shadow-sm);
		transition: all 0.2s ease;
	}

	.stat-card:hover {
		box-shadow: var(--shadow-md);
		border-color: var(--primary);
		transform: translateY(-2px);
	}

	.stat-top {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		margin-bottom: 14px;
	}

	.stat-label {
		font-size: 0.85rem;
		color: var(--foreground-secondary);
		font-weight: 500;
	}

	.stat-icon-wrapper {
		width: 44px;
		height: 44px;
		border-radius: 12px;
		display: flex;
		align-items: center;
		justify-content: center;
		background-color: var(--primary-subtle);
		color: var(--primary);
	}

	.stat-icon-wrapper.success {
		background-color: var(--success-surface);
		color: var(--success);
	}

	.stat-icon-wrapper.danger {
		background-color: var(--danger-surface);
		color: var(--danger);
	}

	.stat-icon-wrapper.accent {
		background-color: var(--accent-subtle);
		color: var(--foreground);
	}

	.stat-value {
		font-size: 1.75rem;
		font-weight: 700;
		color: var(--foreground);
		letter-spacing: -0.03em;
		line-height: 1.15;
		margin-bottom: 8px;
	}

	.stat-footer {
		display: flex;
		align-items: center;
		gap: 6px;
		font-size: 0.8rem;
		color: var(--foreground-muted);
	}

	.footer-link-action {
		display: inline-flex;
		align-items: center;
		gap: 4px;
		color: var(--primary);
		font-weight: 500;
	}

	/* Primary Content: Analytics Grid */
	.analytics-grid {
		display: grid;
		grid-template-columns: 2fr 1fr;
		gap: 20px;
	}

	.budget-col {
		display: flex;
		flex-direction: column;
		justify-content: space-between;
	}

	.budget-overview-box {
		display: flex;
		flex-direction: column;
		gap: 14px;
		margin-top: 10px;
	}

	.budget-big-number {
		display: flex;
		align-items: baseline;
		gap: 8px;
	}

	.budget-spent-val {
		font-size: 1.85rem;
		font-weight: 700;
		color: var(--foreground);
	}

	.budget-limit-sub {
		font-size: 0.85rem;
		color: var(--foreground-muted);
	}

	.budget-progress-track {
		width: 100%;
		height: 8px;
		background-color: var(--surface-secondary);
		border-radius: 9999px;
		overflow: hidden;
	}

	.budget-progress-fill {
		height: 100%;
		background-color: var(--accent);
		border-radius: 9999px;
		transition: width 0.3s ease;
	}

	.budget-progress-fill.warning {
		background-color: var(--warning);
	}

	.budget-progress-fill.danger {
		background-color: var(--danger);
	}

	.threshold-indicators {
		display: flex;
		gap: 12px;
		margin-top: 4px;
	}

	.thresh-item {
		display: flex;
		align-items: center;
		gap: 6px;
		color: var(--foreground-muted);
	}

	.thresh-item.active {
		color: var(--foreground);
		font-weight: 600;
	}

	.thresh-dot {
		width: 8px;
		height: 8px;
		border-radius: 50%;
	}

	.thresh-dot.green {
		background-color: var(--success);
	}
	.thresh-dot.amber {
		background-color: var(--warning);
	}
	.thresh-dot.red {
		background-color: var(--danger);
	}

	.discretionary-info-box {
		padding: 12px 14px;
		border-radius: 10px;
		background-color: var(--surface-secondary);
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin-top: 6px;
	}

	/* Lower Grid */
	.lower-grid {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 20px;
	}

	.goals-list {
		display: flex;
		flex-direction: column;
		gap: 10px;
	}

	.goal-item-card {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 12px 14px;
		background-color: var(--surface-secondary);
		border-radius: 12px;
		border: 1px solid var(--border-subtle);
		transition: all 0.15s ease;
	}

	.goal-item-card:hover {
		border-color: var(--primary);
		background-color: var(--surface-hover);
	}

	.goal-item-left {
		display: flex;
		align-items: center;
		gap: 12px;
	}

	.goal-icon-badge {
		width: 36px;
		height: 36px;
		border-radius: 10px;
		background-color: var(--primary-subtle);
		color: var(--primary);
		display: flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
	}

	.goal-name {
		font-size: 0.92rem;
		font-weight: 600;
		color: var(--foreground);
	}

	.goal-track-mini {
		width: 120px;
		height: 4px;
		background-color: var(--border);
		border-radius: 9999px;
		overflow: hidden;
		margin-top: 4px;
	}

	.goal-fill-mini {
		height: 100%;
		background-color: var(--accent);
		border-radius: 9999px;
	}

	.goal-item-right {
		display: flex;
		align-items: center;
		gap: 8px;
	}

	.table-container {
		overflow-x: auto;
	}

	.table {
		width: 100%;
		border-collapse: collapse;
		text-align: left;
	}

	.table th {
		padding: 10px 14px;
		background-color: var(--surface-secondary);
		color: var(--foreground-secondary);
		font-size: 0.78rem;
		font-weight: 600;
		text-transform: uppercase;
		border-bottom: 1px solid var(--border);
	}

	.table td {
		padding: 12px 14px;
		border-bottom: 1px solid var(--border-subtle);
		font-size: 0.88rem;
	}

	.url-input-row {
		display: flex;
		gap: 8px;
	}

	@media (max-width: 1024px) {
		.stat-grid {
			grid-template-columns: repeat(2, 1fr);
		}
		.analytics-grid {
			grid-template-columns: 1fr;
		}
		.lower-grid {
			grid-template-columns: 1fr;
		}
	}

	@media (max-width: 640px) {
		.stat-grid {
			grid-template-columns: 1fr;
		}
	}
</style>
