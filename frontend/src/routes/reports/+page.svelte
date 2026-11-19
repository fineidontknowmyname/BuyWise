<script lang="ts">
	import { resolve } from '$app/paths';
	import { formatRupees } from '#lib/money.js';
	import { Sparkles, Wallet, PieChart, Target, ShieldCheck } from '@lucide/svelte';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	// Calculations
	const totalSpentPaise = $derived(data.budget ? data.budget.totalSpentPaise : 0);
	const totalLimitPaise = $derived(data.budget ? data.budget.totalLimitPaise : 0);
	const discretionarySurplus = $derived(data.budget ? data.budget.unbudgetedDiscretionaryPaise : 0);

	const budgetSpentPct = $derived(
		totalLimitPaise > 0 ? Math.min(100, Math.round((totalSpentPaise / totalLimitPaise) * 100)) : 0
	);

	const totalGoalsTargetPaise = $derived(
		data.goals.reduce((acc, g) => acc + (g.targetPricePaise || 0), 0)
	);
	const totalGoalsSavedPaise = $derived(
		data.goals.reduce((acc, g) => acc + (g.savedAmountPaise || 0), 0)
	);
	const goalsReadinessPct = $derived(
		totalGoalsTargetPaise > 0
			? Math.min(100, Math.round((totalGoalsSavedPaise / totalGoalsTargetPaise) * 100))
			: 0
	);

	// AI Briefing state
	let isGeneratingBriefing = $state(false);
	let aiBriefingText = $state('');
	let aiBriefingError = $state('');

	async function generateAiBriefing() {
		isGeneratingBriefing = true;
		aiBriefingError = '';

		const prompt = `Generate a concise Executive Financial Briefing for this user:
Liquid Capital: ₹${(data.totalLiquidPaise / 100).toLocaleString('en-IN')}
Monthly Budget: Spent ₹${(totalSpentPaise / 100).toLocaleString('en-IN')} of ₹${(totalLimitPaise / 100).toLocaleString('en-IN')} (${budgetSpentPct}% utilized)
Discretionary Surplus: ₹${(discretionarySurplus / 100).toLocaleString('en-IN')}
Active Goals: ${data.goals.length} goals (Target: ₹${(totalGoalsTargetPaise / 100).toLocaleString('en-IN')}, Saved: ₹${(totalGoalsSavedPaise / 100).toLocaleString('en-IN')})

Provide a 3-part bulleted briefing:
1. Executive Assessment (Health status & budget discipline)
2. Spending & Savings Opportunities
3. Immediate Action Plan for this week.`;

		try {
			const res = await fetch(resolve('/api/ai/chat'), {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					messages: [{ role: 'user', content: prompt }]
				})
			});

			if (!res.ok) {
				const err = await res.json().catch(() => ({}));
				throw new Error(err.error || 'Failed to generate briefing.');
			}

			const result = await res.json();
			aiBriefingText = result.text;
		} catch (err: unknown) {
			aiBriefingError = err instanceof Error ? err.message : 'AI briefing unavailable.';
		} finally {
			isGeneratingBriefing = false;
		}
	}
</script>

<div class="reports-page">
	<header class="page-header">
		<div>
			<h1 class="page-title">Financial Reports & Health Briefing</h1>
			<p class="page-subtitle">
				Consolidated performance overview, budget adherence, and AI executive commentary.
			</p>
		</div>
	</header>

	<!-- KPI Summary Row -->
	<div class="kpi-grid">
		<div class="card kpi-card">
			<div class="kpi-top">
				<span class="kpi-label">Liquid Capital</span>
				<div class="kpi-icon-box bg-primary">
					<Wallet size={18} />
				</div>
			</div>
			<span class="kpi-val">{formatRupees(data.totalLiquidPaise)}</span>
			<span class="kpi-meta">Across {data.accounts.length} active account(s)</span>
		</div>

		<div class="card kpi-card">
			<div class="kpi-top">
				<span class="kpi-label">Monthly Spent</span>
				<div class="kpi-icon-box bg-warning">
					<PieChart size={18} />
				</div>
			</div>
			<span class="kpi-val">{formatRupees(totalSpentPaise)}</span>
			<span class="kpi-meta">{budgetSpentPct}% of limit ({formatRupees(totalLimitPaise)})</span>
		</div>

		<div class="card kpi-card">
			<div class="kpi-top">
				<span class="kpi-label">Discretionary Surplus</span>
				<div class="kpi-icon-box bg-success">
					<ShieldCheck size={18} />
				</div>
			</div>
			<span class="kpi-val">{formatRupees(discretionarySurplus)}</span>
			<span class="kpi-meta text-success">Unallocated buffer</span>
		</div>

		<div class="card kpi-card">
			<div class="kpi-top">
				<span class="kpi-label">Goals Readiness</span>
				<div class="kpi-icon-box bg-info">
					<Target size={18} />
				</div>
			</div>
			<span class="kpi-val">{goalsReadinessPct}%</span>
			<span class="kpi-meta"
				>{formatRupees(totalGoalsSavedPaise)} of {formatRupees(totalGoalsTargetPaise)}</span
			>
		</div>
	</div>

	<!-- AI Executive Briefing Card -->
	<section class="card ai-briefing-card">
		<div class="ai-briefing-header">
			<div class="ai-title-wrap">
				<Sparkles size={20} class="sparkle-icon" />
				<div>
					<h2 class="card-title">BuyWise AI Executive Briefing</h2>
					<p class="card-subtitle">Synthesizes all accounts, spending trends, and targets</p>
				</div>
			</div>
			{#if !aiBriefingText}
				<button
					type="button"
					class="btn btn-primary btn-sm"
					onclick={generateAiBriefing}
					disabled={isGeneratingBriefing}
				>
					{#if isGeneratingBriefing}
						<span class="spinner-small"></span>
						<span>Analyzing Data...</span>
					{:else}
						<Sparkles size={14} />
						<span>Generate Executive Briefing</span>
					{/if}
				</button>
			{/if}
		</div>

		{#if aiBriefingText}
			<div class="briefing-content">
				<p>{aiBriefingText}</p>
				<button
					type="button"
					class="btn btn-secondary btn-sm re-brief-btn"
					onclick={generateAiBriefing}
					disabled={isGeneratingBriefing}
				>
					Re-generate Briefing
				</button>
			</div>
		{:else if aiBriefingError}
			<div class="alert alert-warning">
				{aiBriefingError}
			</div>
		{:else}
			<p class="briefing-prompt-hint">
				Click "Generate Executive Briefing" to produce a customized, real-time assessment of your
				financial strengths, risk points, and recommended actions for this week.
			</p>
		{/if}
	</section>

	<div class="reports-layout">
		<!-- Left: Category Adherence -->
		<div class="layout-left">
			<section class="card section-card">
				<div class="section-title-row">
					<h2 class="card-title">Category Budget Adherence</h2>
					<a href={resolve('/budgets')} class="text-link">Manage Budgets</a>
				</div>

				{#if !data.budget || data.budget.lines.length === 0}
					<p class="empty-text">No category budget limits configured for this month.</p>
				{:else}
					<div class="category-lines-list">
						{#each data.budget.lines as line (line.categoryId)}
							{@const linePct =
								line.limitPaise > 0
									? Math.min(100, Math.round((line.spentPaise / line.limitPaise) * 100))
									: 0}
							{@const isOver = line.spentPaise > line.limitPaise}

							<div class="budget-line-item">
								<div class="line-header">
									<span class="line-name">{line.categoryName}</span>
									<span class="line-amounts">
										<span class="font-bold">{formatRupees(line.spentPaise)}</span>
										<span class="text-muted">/ {formatRupees(line.limitPaise)}</span>
									</span>
								</div>

								<div class="progress-bar-bg">
									<div
										class="progress-bar-fill"
										class:over-budget={isOver}
										style="width: {linePct}%;"
									></div>
								</div>

								<div class="line-footer">
									<span class="line-pct">{linePct}% utilized</span>
									{#if isOver}
										<span class="text-danger font-bold"
											>Exceeded by {formatRupees(line.spentPaise - line.limitPaise)}</span
										>
									{:else}
										<span class="text-muted">{formatRupees(line.remainingPaise)} remaining</span>
									{/if}
								</div>
							</div>
						{/each}
					</div>
				{/if}
			</section>
		</div>

		<!-- Right: Purchase Goals Progress -->
		<div class="layout-right">
			<section class="card section-card">
				<div class="section-title-row">
					<h2 class="card-title">Purchase Goals Tracking</h2>
					<a href={resolve('/goals')} class="text-link">View All</a>
				</div>

				{#if data.goals.length === 0}
					<p class="empty-text">No goals recorded yet.</p>
				{:else}
					<div class="goals-mini-list">
						{#each data.goals as goal (goal.id)}
							{@const gPct =
								(goal.targetPricePaise || 0) > 0
									? Math.min(
											100,
											Math.round(
												((goal.savedAmountPaise || 0) / (goal.targetPricePaise || 1)) * 100
											)
										)
									: 0}

							<div class="goal-mini-item">
								<div class="mini-header">
									<span class="mini-title">{goal.name}</span>
									<span class="mini-badge {goal.priority || 'medium'}">
										{goal.priority || 'medium'}
									</span>
								</div>

								<div class="progress-bar-bg">
									<div class="progress-bar-fill" style="width: {gPct}%;"></div>
								</div>

								<div class="mini-footer">
									<span>{formatRupees(goal.savedAmountPaise || 0)} saved</span>
									<span class="font-bold">{gPct}%</span>
								</div>
							</div>
						{/each}
					</div>
				{/if}
			</section>
		</div>
	</div>
</div>

<style>
	.reports-page {
		max-width: 1200px;
		margin: 0 auto;
		display: flex;
		flex-direction: column;
		gap: 24px;
	}

	.page-header {
		display: flex;
		flex-direction: column;
		gap: 4px;
	}

	.page-title {
		font-size: 28px;
		font-weight: 700;
		color: var(--text-primary);
		letter-spacing: -0.02em;
	}

	.page-subtitle {
		font-size: 15px;
		color: var(--text-secondary);
	}

	.kpi-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
		gap: 16px;
	}

	.card {
		background-color: var(--card-bg);
		border: 1px solid var(--card-border);
		border-radius: var(--radius-lg);
		box-shadow: var(--shadow-sm);
	}

	.kpi-card {
		padding: 20px;
		display: flex;
		flex-direction: column;
		gap: 6px;
	}

	.kpi-top {
		display: flex;
		align-items: center;
		justify-content: space-between;
	}

	.kpi-label {
		font-size: 12px;
		text-transform: uppercase;
		letter-spacing: 0.05em;
		color: var(--text-muted);
		font-weight: 600;
	}

	.kpi-icon-box {
		width: 32px;
		height: 32px;
		border-radius: var(--radius-sm);
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.kpi-icon-box.bg-primary {
		background-color: rgba(22, 139, 219, 0.12);
		color: var(--primary-blue);
	}
	.kpi-icon-box.bg-warning {
		background-color: rgba(183, 121, 31, 0.12);
		color: var(--warning);
	}
	.kpi-icon-box.bg-success {
		background-color: rgba(23, 134, 91, 0.12);
		color: var(--success);
	}
	.kpi-icon-box.bg-info {
		background-color: rgba(72, 184, 242, 0.12);
		color: var(--accent-cyan);
	}

	.kpi-val {
		font-size: 24px;
		font-weight: 800;
		color: var(--text-primary);
		font-variant-numeric: tabular-nums;
		margin-top: 4px;
	}

	.kpi-meta {
		font-size: 12px;
		color: var(--text-secondary);
	}

	/* AI Briefing Card */
	.ai-briefing-card {
		padding: 24px;
		display: flex;
		flex-direction: column;
		gap: 16px;
		background: linear-gradient(135deg, var(--card-bg) 0%, var(--surface-secondary) 100%);
		border: 1px solid rgba(22, 139, 219, 0.25);
	}

	.ai-briefing-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 16px;
		flex-wrap: wrap;
	}

	.ai-title-wrap {
		display: flex;
		align-items: center;
		gap: 12px;
	}

	:global(.sparkle-icon) {
		color: var(--primary-blue);
	}

	.card-title {
		font-size: 17px;
		font-weight: 700;
		color: var(--text-primary);
	}

	.card-subtitle {
		font-size: 13px;
		color: var(--text-secondary);
	}

	.briefing-content {
		font-size: 14px;
		line-height: 1.6;
		color: var(--text-primary);
		white-space: pre-line;
	}

	.re-brief-btn {
		margin-top: 12px;
	}

	.briefing-prompt-hint {
		font-size: 14px;
		color: var(--text-secondary);
		line-height: 1.5;
	}

	/* Reports layout */
	.reports-layout {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 24px;
	}

	@media (max-width: 900px) {
		.reports-layout {
			grid-template-columns: 1fr;
		}
	}

	.section-card {
		padding: 24px;
		display: flex;
		flex-direction: column;
		gap: 20px;
	}

	.section-title-row {
		display: flex;
		align-items: center;
		justify-content: space-between;
	}

	.text-link {
		color: var(--primary-blue);
		font-size: 13px;
		font-weight: 500;
		text-decoration: none;
	}

	.text-link:hover {
		text-decoration: underline;
	}

	.category-lines-list,
	.goals-mini-list {
		display: flex;
		flex-direction: column;
		gap: 16px;
	}

	.budget-line-item,
	.goal-mini-item {
		display: flex;
		flex-direction: column;
		gap: 6px;
	}

	.line-header,
	.mini-header {
		display: flex;
		justify-content: space-between;
		font-size: 13px;
	}

	.line-name,
	.mini-title {
		font-weight: 600;
		color: var(--text-primary);
	}

	.line-amounts {
		font-variant-numeric: tabular-nums;
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

	.progress-bar-fill.over-budget {
		background-color: var(--danger);
	}

	.line-footer,
	.mini-footer {
		display: flex;
		justify-content: space-between;
		font-size: 11px;
	}

	.line-pct {
		font-weight: 600;
		color: var(--text-secondary);
	}

	.mini-badge {
		font-size: 10px;
		text-transform: uppercase;
		font-weight: 700;
		padding: 2px 6px;
		border-radius: 4px;
	}

	.mini-badge.high {
		background-color: rgba(214, 69, 69, 0.15);
		color: var(--danger);
	}
	.mini-badge.medium {
		background-color: rgba(183, 121, 31, 0.15);
		color: var(--warning);
	}
	.mini-badge.low {
		background-color: var(--surface-secondary);
		color: var(--text-secondary);
	}

	.empty-text {
		color: var(--text-muted);
		font-size: 13px;
		text-align: center;
		padding: 20px 0;
	}

	.font-bold {
		font-weight: 700;
	}

	.text-muted {
		color: var(--text-muted);
	}

	.text-danger {
		color: var(--danger);
	}

	.text-success {
		color: var(--success);
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
