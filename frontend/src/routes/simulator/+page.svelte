<script lang="ts">
	import { askAssistant, type AssistantAnswer } from '#lib/assistant.js';
	import AssistantEvidence from '#lib/components/AssistantEvidence.svelte';
	import { formatRupees } from '#lib/money.js';
	import { calculateEMI, generateAmortizationSchedule } from '#lib/emi.js';
	import type { Product, Goal } from '#lib/server/api.js';
	import {
		Calculator,
		Sparkles,
		ShieldCheck,
		AlertTriangle,
		XCircle,
		PiggyBank,
		CreditCard,
		Wallet,
		ChevronDown,
		ChevronUp,
		Layers
	} from '@lucide/svelte';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	// Presets & inputs
	let selectedSource = $state<'custom' | 'product' | 'goal'>('custom');
	let selectedItemId = $state('');

	let purchasePrice = $state(60000);
	let downPayment = $state(10000);
	let tenureMonths = $state(12);
	let annualInterestRate = $state(13.5);
	let processingFeeRupees = $state(500);

	let showSchedule = $state(false);

	// AI Advice state
	let isAnalyzing = $state(false);
	let aiAdvice = $state<AssistantAnswer | null>(null);
	let aiError = $state('');

	// Populate from selected product or goal
	function handleSourceChange(type: 'custom' | 'product' | 'goal', id = '') {
		selectedSource = type;
		selectedItemId = id;

		if (type === 'product' && id) {
			const prod = (data.products || []).find((p: Product) => p.id === id);
			if (prod && prod.currentPricePaise > 0) {
				purchasePrice = Math.round(prod.currentPricePaise / 100);
				downPayment = Math.round(purchasePrice * 0.1);
			}
		} else if (type === 'goal' && id) {
			const goal = (data.goals || []).find((g: Goal) => g.id === id);
			if (goal && (goal.targetPricePaise || 0) > 0) {
				purchasePrice = Math.round((goal.targetPricePaise || 0) / 100);
				downPayment = Math.round((goal.savedAmountPaise || 0) / 100);
			}
		}
	}

	// Down payment percentage helper
	const downPaymentPct = $derived(
		purchasePrice > 0 ? Math.min(100, Math.round((downPayment / purchasePrice) * 100)) : 0
	);

	function setDownPaymentPercent(pct: number) {
		downPayment = Math.round((purchasePrice * pct) / 100);
	}

	// Real-time deterministic EMI calculation
	const simulation = $derived(
		calculateEMI({
			productPricePaise: Math.round(purchasePrice * 100),
			downPaymentPaise: Math.round(downPayment * 100),
			annualRatePercent: annualInterestRate,
			tenureMonths,
			processingFeePaise: Math.round(processingFeeRupees * 100),
			monthlyDiscretionaryBudgetPaise: data.discretionarySurplusPaise
		})
	);

	// Amortization schedule
	const schedule = $derived(
		generateAmortizationSchedule(
			simulation.principalPaise,
			annualInterestRate,
			tenureMonths,
			simulation.monthlyInstallmentPaise
		)
	);

	// Safety assessment based on discretionary income
	const safetyStatus = $derived.by(() => {
		const util = simulation.budgetUtilizationPercent;
		if (util === 0) return { level: 'unknown', text: 'No budget data', color: 'muted' };
		if (util <= 25) return { level: 'safe', text: 'Comfortable & Safe', color: 'success' };
		if (util <= 40) return { level: 'moderate', text: 'Moderate Commitment', color: 'warning' };
		return { level: 'risky', text: 'High Risk (Exceeds 40% surplus)', color: 'danger' };
	});

	async function getAiAdvice() {
		isAnalyzing = true;
		aiError = '';
		try {
			aiAdvice = await askAssistant({ topic: 'capacity' });
		} catch (err) {
			aiError = err instanceof Error ? err.message : 'The assistant is temporarily unavailable.';
		} finally {
			isAnalyzing = false;
		}
	}
</script>

<div class="simulator-page">
	<header class="page-header">
		<div>
			<h1 class="page-title">EMI & Affordability Simulator</h1>
			<p class="page-subtitle">
				Compare Cash vs. Self-EMI vs. Loan Financing with verified math and budget impact.
			</p>
		</div>
	</header>

	<!-- Source Selection Tabs -->
	<div class="source-selector">
		<button
			type="button"
			class="source-tab"
			class:active={selectedSource === 'custom'}
			onclick={() => handleSourceChange('custom')}
		>
			<Calculator size={16} />
			<span>Custom Simulation</span>
		</button>

		{#if (data.products || []).length > 0}
			<button
				type="button"
				class="source-tab"
				class:active={selectedSource === 'product'}
				onclick={() => handleSourceChange('product', data.products?.[0]?.id)}
			>
				<Layers size={16} />
				<span>From Tracked Product</span>
			</button>
		{/if}

		{#if (data.goals || []).length > 0}
			<button
				type="button"
				class="source-tab"
				class:active={selectedSource === 'goal'}
				onclick={() => handleSourceChange('goal', data.goals?.[0]?.id)}
			>
				<PiggyBank size={16} />
				<span>From Purchase Goal</span>
			</button>
		{/if}
	</div>

	<!-- Item Pickers if product or goal is selected -->
	{#if selectedSource === 'product' && (data.products || []).length > 0}
		<div class="picker-bar">
			<label for="product-select" class="picker-label">Select Product:</label>
			<select
				id="product-select"
				class="picker-select"
				value={selectedItemId}
				onchange={(e) => handleSourceChange('product', e.currentTarget.value)}
			>
				{#each data.products || [] as prod (prod.id)}
					<option value={prod.id}>
						{prod.name} — {formatRupees(prod.currentPricePaise)}
					</option>
				{/each}
			</select>
		</div>
	{:else if selectedSource === 'goal' && data.goals.length > 0}
		<div class="picker-bar">
			<label for="goal-select" class="picker-label">Select Goal:</label>
			<select
				id="goal-select"
				class="picker-select"
				value={selectedItemId}
				onchange={(e) => handleSourceChange('goal', e.currentTarget.value)}
			>
				{#each data.goals as goal (goal.id)}
					<option value={goal.id}>
						{goal.name} — Target: {formatRupees(goal.targetPricePaise || 0)}
					</option>
				{/each}
			</select>
		</div>
	{/if}

	<div class="simulator-layout">
		<!-- Left: Controls & Sliders -->
		<div class="controls-column">
			<div class="card controls-card">
				<h2 class="card-heading">Loan Parameters</h2>

				<!-- Purchase Amount -->
				<div class="form-group">
					<div class="group-header">
						<label for="purchase-price" class="form-label">Purchase Amount (₹)</label>
						<span class="value-display font-bold">₹{purchasePrice.toLocaleString('en-IN')}</span>
					</div>
					<input
						id="purchase-price"
						type="range"
						min="1000"
						max="500000"
						step="1000"
						bind:value={purchasePrice}
						class="range-slider"
					/>
					<div class="quick-chips">
						{#each [25000, 50000, 80000, 150000] as preset (preset)}
							<button
								type="button"
								class="chip-btn"
								class:selected={purchasePrice === preset}
								onclick={() => (purchasePrice = preset)}
							>
								₹{(preset / 1000).toFixed(0)}k
							</button>
						{/each}
					</div>
				</div>

				<!-- Down Payment -->
				<div class="form-group">
					<div class="group-header">
						<label for="down-payment" class="form-label">Down Payment (₹)</label>
						<span class="value-display">
							₹{downPayment.toLocaleString('en-IN')} ({downPaymentPct}%)
						</span>
					</div>
					<input
						id="down-payment"
						type="range"
						min="0"
						max={purchasePrice}
						step="1000"
						bind:value={downPayment}
						class="range-slider"
					/>
					<div class="quick-chips">
						{#each [0, 10, 20, 30, 50] as pct (pct)}
							<button
								type="button"
								class="chip-btn"
								class:selected={downPaymentPct === pct}
								onclick={() => setDownPaymentPercent(pct)}
							>
								{pct}%
							</button>
						{/each}
					</div>
				</div>

				<!-- Tenure -->
				<div class="form-group">
					<div class="group-header">
						<span class="form-label">Tenure (Months)</span>
						<span class="value-display font-bold">{tenureMonths} Months</span>
					</div>
					<div class="tenure-grid">
						{#each [3, 6, 9, 12, 18, 24, 36] as t (t)}
							<button
								type="button"
								class="tenure-btn"
								class:selected={tenureMonths === t}
								onclick={() => (tenureMonths = t)}
							>
								{t}m
							</button>
						{/each}
					</div>
				</div>

				<!-- Interest Rate -->
				<div class="form-group">
					<div class="group-header">
						<label for="interest-rate" class="form-label">Interest Rate (% p.a.)</label>
						<span class="value-display font-bold">{annualInterestRate}%</span>
					</div>
					<input
						id="interest-rate"
						type="range"
						min="0"
						max="36"
						step="0.5"
						bind:value={annualInterestRate}
						class="range-slider"
					/>
					<div class="rate-presets">
						<button
							type="button"
							class="preset-pill"
							class:selected={annualInterestRate === 0}
							onclick={() => (annualInterestRate = 0)}
						>
							0% (No-Cost)
						</button>
						<button
							type="button"
							class="preset-pill"
							class:selected={annualInterestRate === 10.5}
							onclick={() => (annualInterestRate = 10.5)}
						>
							10.5% (Personal)
						</button>
						<button
							type="button"
							class="preset-pill"
							class:selected={annualInterestRate === 13.5}
							onclick={() => (annualInterestRate = 13.5)}
						>
							13.5% (Card EMI)
						</button>
						<button
							type="button"
							class="preset-pill"
							class:selected={annualInterestRate === 16}
							onclick={() => (annualInterestRate = 16)}
						>
							16% (Consumer)
						</button>
					</div>
				</div>

				<!-- Processing Fee -->
				<div class="form-group">
					<div class="group-header">
						<label for="processing-fee" class="form-label">Processing Fee (₹)</label>
						<input
							id="processing-fee"
							type="number"
							min="0"
							step="100"
							bind:value={processingFeeRupees}
							class="input-compact"
						/>
					</div>
				</div>
			</div>
		</div>

		<!-- Right: Metrics & Comparison Scenarios -->
		<div class="results-column">
			<!-- Highlight Banner: Monthly EMI & Budget Assessment -->
			<div class="card result-hero-card">
				<div class="hero-top">
					<div class="emi-metric">
						<span class="emi-label">Monthly EMI Installment</span>
						<span class="emi-val">{formatRupees(simulation.monthlyInstallmentPaise)}</span>
						<span class="emi-tenure">for {tenureMonths} months</span>
					</div>

					<div class="safety-badge {safetyStatus.color}">
						{#if safetyStatus.level === 'safe'}
							<ShieldCheck size={18} />
						{:else if safetyStatus.level === 'moderate'}
							<AlertTriangle size={18} />
						{:else}
							<XCircle size={18} />
						{/if}
						<span>{safetyStatus.text}</span>
					</div>
				</div>

				<!-- Budget Utilization Bar -->
				<div class="utilization-wrap">
					<div class="utilization-header">
						<span class="util-label">Discretionary Budget Impact:</span>
						<span class="util-val">
							{simulation.budgetUtilizationPercent}% of monthly surplus ({formatRupees(
								data.discretionarySurplusPaise
							)})
						</span>
					</div>
					<div class="progress-bar-bg">
						<div
							class="progress-bar-fill {safetyStatus.color}"
							style="width: {Math.min(100, simulation.budgetUtilizationPercent)}%;"
						></div>
					</div>
				</div>

				<div class="cost-breakdown-row">
					<div class="breakdown-item">
						<span class="b-label">Financed Principal</span>
						<span class="b-val">{formatRupees(simulation.principalPaise)}</span>
					</div>
					<div class="breakdown-item">
						<span class="b-label">Total Interest</span>
						<span class="b-val highlight-warn">{formatRupees(simulation.totalInterestPaise)}</span>
					</div>
					<div class="breakdown-item">
						<span class="b-label">Total Extra Cost</span>
						<span class="b-val highlight-danger"
							>{formatRupees(simulation.extraCostVsCashPaise)}</span
						>
					</div>
					<div class="breakdown-item">
						<span class="b-label">Total Outflow</span>
						<span class="b-val">{formatRupees(simulation.totalRepaymentPaise)}</span>
					</div>
				</div>
			</div>

			<!-- Three-Way Comparison Grid -->
			<div class="comparison-grid">
				<!-- Option 1: Cash -->
				<div class="card option-card">
					<div class="option-header">
						<Wallet size={20} class="option-icon cash" />
						<h3 class="option-title">1. Cash Upfront</h3>
					</div>
					<p class="option-desc">Buy outright using existing funds. Zero debt and zero interest.</p>
					<div class="option-stats">
						<div class="stat-line">
							<span>Upfront Outflow:</span>
							<span class="font-bold">{formatRupees(simulation.scenarios.cash.upfrontPaise)}</span>
						</div>
						<div class="stat-line">
							<span>Monthly Burden:</span>
							<span>₹0.00</span>
						</div>
						<div class="stat-line">
							<span>Extra Interest:</span>
							<span class="font-bold text-success">₹0.00</span>
						</div>
					</div>
					<div class="option-footer">
						{#if data.totalLiquidPaise >= simulation.scenarios.cash.upfrontPaise}
							<span class="badge-pill bg-success">Affordable today</span>
						{:else}
							<span class="badge-pill bg-warning">Exceeds liquid cash</span>
						{/if}
					</div>
				</div>

				<!-- Option 2: Self-EMI / Save First -->
				<div class="card option-card featured">
					<div class="featured-badge">Smart Choice</div>
					<div class="option-header">
						<PiggyBank size={20} class="option-icon save" />
						<h3 class="option-title">2. Self-EMI (Save First)</h3>
					</div>
					<p class="option-desc">
						Save ₹{(simulation.monthlyInstallmentPaise / 100).toLocaleString('en-IN')}/mo in your
						account first. Pay ₹0 interest.
					</p>
					<div class="option-stats">
						<div class="stat-line">
							<span>Time to Target:</span>
							<span class="font-bold text-primary-blue"
								>{simulation.monthsToSaveUpfront} Months</span
							>
						</div>
						<div class="stat-line">
							<span>Monthly Savings:</span>
							<span>{formatRupees(simulation.scenarios.savings.monthlyPaise)}</span>
						</div>
						<div class="stat-line">
							<span>Interest Saved:</span>
							<span class="font-bold text-success"
								>{formatRupees(simulation.totalInterestPaise)}</span
							>
						</div>
					</div>
					<div class="option-footer">
						<span class="badge-pill bg-primary">Builds Wealth</span>
					</div>
				</div>

				<!-- Option 3: Loan EMI -->
				<div class="card option-card">
					<div class="option-header">
						<CreditCard size={20} class="option-icon emi" />
						<h3 class="option-title">3. Loan / Credit EMI</h3>
					</div>
					<p class="option-desc">
						Get it immediately. Pay compound interest and monthly commitment.
					</p>
					<div class="option-stats">
						<div class="stat-line">
							<span>Upfront Down Payment:</span>
							<span>{formatRupees(simulation.scenarios.emi.upfrontPaise)}</span>
						</div>
						<div class="stat-line">
							<span>Monthly EMI:</span>
							<span class="font-bold">{formatRupees(simulation.scenarios.emi.monthlyPaise)}</span>
						</div>
						<div class="stat-line">
							<span>Extra Cost:</span>
							<span class="font-bold text-danger"
								>{formatRupees(simulation.scenarios.emi.extraCostPaise)}</span
							>
						</div>
					</div>
					<div class="option-footer">
						<span class="badge-pill bg-neutral">{tenureMonths} mo commitment</span>
					</div>
				</div>
			</div>

			<!-- AI Trade-off Analysis Box -->
			<div class="card ai-card">
				<div class="ai-card-header">
					<div class="ai-brand">
						<Sparkles size={18} class="ai-icon" />
						<h3>Monthly saving capacity</h3>
					</div>
					{#if !aiAdvice}
						<button
							type="button"
							class="btn btn-primary btn-sm"
							onclick={getAiAdvice}
							disabled={isAnalyzing}
						>
							{#if isAnalyzing}
								<span class="spinner-small"></span>
								<span>Evaluating...</span>
							{:else}
								<Sparkles size={14} />
								<span>Explain saving capacity</span>
							{/if}
						</button>
					{/if}
				</div>

				{#if aiAdvice}
					<div class="ai-text-body">
						<AssistantEvidence answer={aiAdvice} />
						<button
							type="button"
							class="btn btn-secondary btn-sm re-run-btn"
							onclick={getAiAdvice}
							disabled={isAnalyzing}
						>
							Refresh explanation
						</button>
					</div>
				{:else if aiError}
					<div class="alert alert-warning">
						{aiError}
					</div>
				{:else}
					<p class="ai-hint">
						Review the monthly capacity from your financial profile alongside the simulation above.
					</p>
				{/if}
			</div>

			<!-- Amortization Schedule Accordion -->
			<div class="card schedule-card">
				<button
					type="button"
					class="schedule-toggle-btn"
					onclick={() => (showSchedule = !showSchedule)}
				>
					<div class="toggle-title">
						<span>Amortization Repayment Schedule</span>
						<span class="schedule-count">({schedule.length} installments)</span>
					</div>
					{#if showSchedule}
						<ChevronUp size={18} />
					{:else}
						<ChevronDown size={18} />
					{/if}
				</button>

				{#if showSchedule}
					<div class="table-wrap">
						<table class="data-table">
							<thead>
								<tr>
									<th>Month</th>
									<th>Opening Balance</th>
									<th>EMI</th>
									<th>Principal</th>
									<th>Interest</th>
									<th>Closing Balance</th>
								</tr>
							</thead>
							<tbody>
								{#each schedule as row (row.month)}
									<tr>
										<td class="tabular font-bold">Month {row.month}</td>
										<td class="tabular">{formatRupees(row.openingBalancePaise)}</td>
										<td class="tabular font-bold">{formatRupees(row.emiPaise)}</td>
										<td class="tabular">{formatRupees(row.principalPaise)}</td>
										<td class="tabular text-danger">{formatRupees(row.interestPaise)}</td>
										<td class="tabular">{formatRupees(row.closingBalancePaise)}</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
				{/if}
			</div>
		</div>
	</div>
</div>

<style>
	.simulator-page {
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

	.source-selector {
		display: flex;
		gap: 8px;
		border-bottom: 1px solid var(--border-color);
		padding-bottom: 8px;
		overflow-x: auto;
	}

	.source-tab {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 8px 14px;
		border-radius: var(--radius-md);
		border: 1px solid transparent;
		background: none;
		color: var(--text-secondary);
		font-size: 14px;
		font-weight: 500;
		cursor: pointer;
		transition: all var(--transition-fast);
		white-space: nowrap;
	}

	.source-tab:hover {
		background-color: var(--surface-secondary);
		color: var(--text-primary);
	}

	.source-tab.active {
		background-color: var(--surface-secondary);
		color: var(--primary-blue);
		border-color: var(--border-color);
		font-weight: 600;
	}

	.picker-bar {
		display: flex;
		align-items: center;
		gap: 12px;
		background-color: var(--card-bg);
		border: 1px solid var(--border-color);
		border-radius: var(--radius-md);
		padding: 10px 16px;
	}

	.picker-label {
		font-size: 13px;
		font-weight: 600;
		color: var(--text-primary);
	}

	.picker-select {
		flex: 1;
		padding: 6px 12px;
		border-radius: var(--radius-sm);
		border: 1px solid var(--border-color);
		background-color: var(--surface-secondary);
		color: var(--text-primary);
		font-size: 14px;
		outline: none;
	}

	.simulator-layout {
		display: grid;
		grid-template-columns: 380px 1fr;
		gap: 24px;
	}

	@media (max-width: 960px) {
		.simulator-layout {
			grid-template-columns: 1fr;
		}
	}

	.card {
		background-color: var(--card-bg);
		border: 1px solid var(--card-border);
		border-radius: var(--radius-lg);
		box-shadow: var(--shadow-sm);
	}

	.controls-card {
		padding: 24px;
		display: flex;
		flex-direction: column;
		gap: 20px;
	}

	.card-heading {
		font-size: 17px;
		font-weight: 700;
		color: var(--text-primary);
	}

	.form-group {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}

	.group-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
	}

	.form-label {
		font-size: 13px;
		font-weight: 600;
		color: var(--text-secondary);
	}

	.value-display {
		font-size: 14px;
		color: var(--text-primary);
		font-variant-numeric: tabular-nums;
	}

	.range-slider {
		width: 100%;
		accent-color: var(--primary-blue);
		cursor: pointer;
	}

	.quick-chips {
		display: flex;
		gap: 6px;
		flex-wrap: wrap;
	}

	.chip-btn {
		padding: 3px 8px;
		font-size: 11px;
		font-weight: 500;
		border-radius: 4px;
		border: 1px solid var(--border-color);
		background-color: var(--surface-secondary);
		color: var(--text-secondary);
		cursor: pointer;
		transition: all var(--transition-fast);
	}

	.chip-btn.selected,
	.chip-btn:hover {
		background-color: var(--primary-blue);
		color: #ffffff;
		border-color: var(--primary-blue);
	}

	.tenure-grid {
		display: grid;
		grid-template-columns: repeat(4, 1fr);
		gap: 6px;
	}

	.tenure-btn {
		padding: 8px;
		font-size: 13px;
		font-weight: 600;
		border-radius: var(--radius-sm);
		border: 1px solid var(--border-color);
		background-color: var(--surface-secondary);
		color: var(--text-secondary);
		cursor: pointer;
		text-align: center;
		transition: all var(--transition-fast);
	}

	.tenure-btn.selected {
		background-color: var(--primary-blue);
		color: #ffffff;
		border-color: var(--primary-blue);
	}

	.rate-presets {
		display: flex;
		gap: 6px;
		flex-wrap: wrap;
	}

	.preset-pill {
		padding: 4px 8px;
		border-radius: 999px;
		border: 1px solid var(--border-color);
		background-color: var(--surface-secondary);
		font-size: 11px;
		font-weight: 500;
		color: var(--text-secondary);
		cursor: pointer;
	}

	.preset-pill.selected {
		background-color: var(--primary-blue);
		color: #ffffff;
		border-color: var(--primary-blue);
	}

	.input-compact {
		width: 100px;
		padding: 4px 8px;
		border-radius: var(--radius-sm);
		border: 1px solid var(--border-color);
		background-color: var(--surface-secondary);
		color: var(--text-primary);
		font-size: 13px;
		text-align: right;
		outline: none;
	}

	/* Results column */
	.results-column {
		display: flex;
		flex-direction: column;
		gap: 20px;
	}

	.result-hero-card {
		padding: 24px;
		display: flex;
		flex-direction: column;
		gap: 20px;
		background: linear-gradient(135deg, var(--card-bg) 0%, var(--surface-secondary) 100%);
		border: 1px solid rgba(22, 139, 219, 0.25);
	}

	.hero-top {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		gap: 16px;
		flex-wrap: wrap;
	}

	.emi-metric {
		display: flex;
		flex-direction: column;
	}

	.emi-label {
		font-size: 12px;
		text-transform: uppercase;
		letter-spacing: 0.05em;
		color: var(--text-muted);
		font-weight: 600;
	}

	.emi-val {
		font-size: 32px;
		font-weight: 800;
		color: var(--text-primary);
		font-variant-numeric: tabular-nums;
		line-height: 1.1;
		margin: 4px 0;
	}

	.emi-tenure {
		font-size: 13px;
		color: var(--text-secondary);
	}

	.safety-badge {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		padding: 6px 12px;
		border-radius: 999px;
		font-size: 12px;
		font-weight: 600;
	}

	.safety-badge.success {
		background-color: rgba(23, 134, 91, 0.15);
		color: var(--success);
	}

	.safety-badge.warning {
		background-color: rgba(183, 121, 31, 0.15);
		color: var(--warning);
	}

	.safety-badge.danger {
		background-color: rgba(214, 69, 69, 0.15);
		color: var(--danger);
	}

	.utilization-wrap {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}

	.utilization-header {
		display: flex;
		justify-content: space-between;
		font-size: 12px;
	}

	.util-label {
		color: var(--text-secondary);
	}

	.util-val {
		font-weight: 600;
		color: var(--text-primary);
	}

	.progress-bar-bg {
		width: 100%;
		height: 8px;
		background-color: var(--border-color);
		border-radius: 999px;
		overflow: hidden;
	}

	.progress-bar-fill {
		height: 100%;
		border-radius: 999px;
		transition: width 0.3s ease;
	}

	.progress-bar-fill.success {
		background-color: var(--success);
	}

	.progress-bar-fill.warning {
		background-color: var(--warning);
	}

	.progress-bar-fill.danger {
		background-color: var(--danger);
	}

	.cost-breakdown-row {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(130px, 1fr));
		gap: 16px;
		padding-top: 16px;
		border-top: 1px solid var(--border-color);
	}

	.breakdown-item {
		display: flex;
		flex-direction: column;
	}

	.b-label {
		font-size: 11px;
		text-transform: uppercase;
		letter-spacing: 0.05em;
		color: var(--text-muted);
		font-weight: 600;
	}

	.b-val {
		font-size: 15px;
		font-weight: 700;
		color: var(--text-primary);
		font-variant-numeric: tabular-nums;
		margin-top: 2px;
	}

	.highlight-warn {
		color: var(--warning);
	}

	.highlight-danger {
		color: var(--danger);
	}

	/* Three-way comparison grid */
	.comparison-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
		gap: 16px;
	}

	.option-card {
		padding: 18px;
		display: flex;
		flex-direction: column;
		gap: 12px;
		position: relative;
	}

	.option-card.featured {
		border-color: var(--primary-blue);
		box-shadow: 0 4px 12px rgba(22, 139, 219, 0.15);
	}

	.featured-badge {
		position: absolute;
		top: -10px;
		right: 16px;
		background-color: var(--primary-blue);
		color: #ffffff;
		font-size: 10px;
		font-weight: 700;
		padding: 2px 8px;
		border-radius: 999px;
		text-transform: uppercase;
		letter-spacing: 0.04em;
	}

	.option-header {
		display: flex;
		align-items: center;
		gap: 8px;
	}

	:global(.option-icon.cash) {
		color: var(--success);
	}
	:global(.option-icon.save) {
		color: var(--primary-blue);
	}
	:global(.option-icon.emi) {
		color: var(--warning);
	}

	.option-title {
		font-size: 15px;
		font-weight: 700;
		color: var(--text-primary);
	}

	.option-desc {
		font-size: 12px;
		color: var(--text-secondary);
		line-height: 1.4;
	}

	.option-stats {
		display: flex;
		flex-direction: column;
		gap: 6px;
		margin-top: auto;
		padding-top: 10px;
		border-top: 1px solid var(--border-color);
		font-size: 12px;
	}

	.stat-line {
		display: flex;
		justify-content: space-between;
		font-variant-numeric: tabular-nums;
	}

	.option-footer {
		padding-top: 4px;
	}

	.badge-pill {
		display: inline-block;
		font-size: 11px;
		font-weight: 600;
		padding: 2px 8px;
		border-radius: 4px;
	}

	.bg-success {
		background-color: rgba(23, 134, 91, 0.15);
		color: var(--success);
	}

	.bg-warning {
		background-color: rgba(183, 121, 31, 0.15);
		color: var(--warning);
	}

	.bg-primary {
		background-color: rgba(22, 139, 219, 0.15);
		color: var(--primary-blue);
	}

	.bg-neutral {
		background-color: var(--surface-secondary);
		color: var(--text-secondary);
	}

	/* AI Card */
	.ai-card {
		padding: 20px;
		display: flex;
		flex-direction: column;
		gap: 12px;
		border: 1px solid rgba(22, 139, 219, 0.2);
	}

	.ai-card-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 12px;
		flex-wrap: wrap;
	}

	.ai-brand {
		display: flex;
		align-items: center;
		gap: 8px;
		font-size: 15px;
		font-weight: 600;
		color: var(--primary-blue);
	}

	.ai-text-body {
		font-size: 14px;
		line-height: 1.6;
		color: var(--text-primary);
		white-space: pre-line;
	}

	.re-run-btn {
		margin-top: 10px;
	}

	.ai-hint {
		font-size: 13px;
		color: var(--text-secondary);
	}

	/* Schedule Card */
	.schedule-card {
		padding: 16px 20px;
	}

	.schedule-toggle-btn {
		width: 100%;
		display: flex;
		align-items: center;
		justify-content: space-between;
		background: none;
		border: none;
		color: var(--text-primary);
		font-size: 15px;
		font-weight: 600;
		cursor: pointer;
		padding: 0;
	}

	.toggle-title {
		display: flex;
		align-items: center;
		gap: 8px;
	}

	.schedule-count {
		font-size: 12px;
		color: var(--text-muted);
		font-weight: normal;
	}

	.table-wrap {
		margin-top: 16px;
		overflow-x: auto;
	}

	.data-table {
		width: 100%;
		border-collapse: collapse;
		font-size: 12px;
	}

	.data-table th,
	.data-table td {
		padding: 8px 10px;
		text-align: left;
		border-bottom: 1px solid var(--border-color);
	}

	.data-table th {
		color: var(--text-muted);
		font-weight: 600;
		font-size: 11px;
		text-transform: uppercase;
		letter-spacing: 0.04em;
	}

	.tabular {
		font-variant-numeric: tabular-nums;
	}

	.font-bold {
		font-weight: 700;
	}

	.text-success {
		color: var(--success);
	}

	.text-danger {
		color: var(--danger);
	}

	.text-primary-blue {
		color: var(--primary-blue);
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
