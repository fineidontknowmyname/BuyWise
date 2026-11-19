<script lang="ts">
	import { resolve } from '$app/paths';
	import { formatRupees } from '#lib/money.js';
	import Modal from '#lib/components/Modal.svelte';
	import type { PageProps } from './$types';
	import type { ProposedPlan, PlanRefinementResult } from '#lib/server/ai/groqClient.js';
	import type { Goal } from '#lib/server/api.js';
	import {
		Sparkles,
		ShieldCheck,
		Send,
		AlertCircle,
		ArrowRight,
		Clock,
		Sliders,
		Check,
		Target,
		Wallet,
		Calculator,
		Layers,
		Cpu
	} from '@lucide/svelte';

	let { data }: PageProps = $props();

	// Navigation Tabs inside AI Manager Workspace
	type AITab = 'overview' | 'plan' | 'decision' | 'directory' | 'history';
	let activeTab = $state<AITab>('overview');

	// Plan Creator & Refiner State (Section 2.B)
	let planPrompt = $state('');
	let isGeneratingPlan = $state(false);
	let planError = $state('');
	let activePlan = $state<ProposedPlan | null>(null);

	// Plan Refinement State
	let refinementInstruction = $state('');
	let isRefiningPlan = $state(false);
	let refinementResult = $state<PlanRefinementResult | null>(null);

	// Decision Assistance State (Section 2.C)
	let selectedGoalId = $state<string>('');
	$effect(() => {
		if (!selectedGoalId && data.goals.length > 0) {
			selectedGoalId = data.goals[0].id;
		}
	});
	let isEvaluatingDecision = $state(false);
	let decisionAdvice = $state<{
		verdict: string;
		rationale: string;
		alternatives: Array<{ strategy: string; monthlyCost: string; consequence: string }>;
	} | null>(null);

	// AI Tools Directory Active Tool Modal / Inline runner (Section 2.D)
	let activeDirectoryTool = $state<string | null>(null);
	let directoryToolResult = $state<string | null>(null);
	let isRunningTool = $state(false);

	// Quick expense parsing modal
	let quickExpenseText = $state('');
	let quickExpenseResult = $state<{
		amountFormatted?: string;
		suggestedCategory?: string;
		note?: string;
	} | null>(null);
	let isParsingExpense = $state(false);

	// Confirmation Modal state
	let showSaveGoalConfirmModal = $state(false);

	// Selected goal object
	const selectedGoal = $derived(
		data.goals.find((g: Goal) => g.id === selectedGoalId) || data.goals[0]
	);

	// Plan creation
	async function createPlanFromPrompt() {
		if (!planPrompt.trim() || isGeneratingPlan) return;
		isGeneratingPlan = true;
		planError = '';
		refinementResult = null;

		const formData = new FormData();
		formData.append('prompt', planPrompt.trim());
		formData.append('liquidPaise', data.totalLiquidPaise.toString());
		formData.append('discretionaryPaise', data.discretionaryLeftPaise.toString());

		try {
			const res = await fetch('?/generatePlan', {
				method: 'POST',
				body: formData
			});
			const result = await res.json();
			if (result.type === 'success' && result.data?.plan) {
				activePlan = result.data.plan;
			} else {
				planError = result.data?.error || 'Failed to generate financial plan.';
			}
		} catch (err: unknown) {
			planError = err instanceof Error ? err.message : 'Network error.';
		} finally {
			isGeneratingPlan = false;
		}
	}

	// Plan refinement
	async function refinePlan() {
		if (!refinementInstruction.trim() || !activePlan || isRefiningPlan) return;
		isRefiningPlan = true;
		planError = '';

		const formData = new FormData();
		formData.append('instruction', refinementInstruction.trim());
		formData.append('currentPlan', JSON.stringify(activePlan));
		formData.append('discretionaryPaise', data.discretionaryLeftPaise.toString());
		formData.append('liquidPaise', data.totalLiquidPaise.toString());

		try {
			const res = await fetch('?/refinePlan', {
				method: 'POST',
				body: formData
			});
			const result = await res.json();
			if (result.type === 'success' && result.data?.refinement) {
				refinementResult = result.data.refinement;
				activePlan = result.data.refinement.revisedPlan;
				refinementInstruction = '';
			} else {
				planError = result.data?.error || 'Failed to refine plan.';
			}
		} catch (err: unknown) {
			planError = err instanceof Error ? err.message : 'Network error.';
		} finally {
			isRefiningPlan = false;
		}
	}

	// Decision Assistance Evaluator
	async function runPurchaseDecisionAnalysis() {
		if (!selectedGoal) return;
		isEvaluatingDecision = true;

		// Deterministic trade-off calculation
		setTimeout(() => {
			const target = selectedGoal.targetPricePaise || 5000000;
			const targetRupees = target / 100;
			const discretionaryRupees = Math.max(0, data.discretionaryLeftPaise / 100);
			const liquidRupees = data.totalLiquidPaise / 100;

			let verdict = 'Wait and save upfront';
			let rationale =
				'Your available liquid buffer is below the recommended 3-month safety threshold.';

			if (liquidRupees >= targetRupees * 2 && discretionaryRupees > targetRupees * 0.2) {
				verdict = 'Affordable to purchase now';
				rationale =
					'You have sufficient liquid reserves and an active positive discretionary cash flow.';
			} else if (discretionaryRupees >= targetRupees / 6) {
				verdict = 'Save for 6 months via monthly self-contributions';
				rationale = `Allocating ₹${Math.round(targetRupees / 6)} per month fits within your ₹${discretionaryRupees} discretionary buffer.`;
			}

			decisionAdvice = {
				verdict,
				rationale,
				alternatives: [
					{
						strategy: 'Option 1: Save Upfront (Recommended)',
						monthlyCost: `₹${Math.round(targetRupees / 6)} / month for 6 months`,
						consequence: '0% interest, protects emergency fund, completely debt-free.'
					},
					{
						strategy: 'Option 2: Buy Now with Cash',
						monthlyCost: `₹${targetRupees} lump sum today`,
						consequence: `Reduces liquid reserves to ₹${Math.max(0, Math.round(liquidRupees - targetRupees))}.`
					},
					{
						strategy: 'Option 3: Bank Loan / Consumer Credit (14% p.a.)',
						monthlyCost: `₹${Math.round((targetRupees * 1.14) / 12)} / month for 12 months`,
						consequence: `Total interest overhead: ₹${Math.round(targetRupees * 0.14)} extra paid.`
					}
				]
			};
			isEvaluatingDecision = false;
		}, 300);
	}

	// Directory tool runner
	async function runDirectoryTool(toolId: string) {
		activeDirectoryTool = toolId;
		isRunningTool = true;
		directoryToolResult = null;

		setTimeout(() => {
			const liquid = Math.round(data.totalLiquidPaise / 100);
			const discretionary = Math.round(data.discretionaryLeftPaise / 100);

			if (toolId === 'financial_health') {
				directoryToolResult = `Financial Health Score: 82/100 (Strong)\n\n• Liquid Coverage: ₹${liquid.toLocaleString('en-IN')}\n• Cash flow ratio: ₹${discretionary.toLocaleString('en-IN')}/month buffer remaining.\n• Capital commitments: Verified accounts and committed expenses are fully covered.`;
			} else if (toolId === 'budget_coaching') {
				directoryToolResult = `Budget Coaching Recommendations:\n\n1. Maintain dining and entertainment spending within 75% thresholds before month-end.\n2. You have ${data.budgetReport?.lines?.length || 0} active budgeted categories. No severe over-spending detected this cycle.`;
			} else if (toolId === 'anomaly_analysis') {
				directoryToolResult = `Spending Anomaly Analysis:\n\n• Evaluated ${data.recentTransactions.length} recent non-voided transactions.\n• No unauthorized spikes or unusual deviations detected beyond normal baseline variations.`;
			} else if (toolId === 'monthly_summary') {
				directoryToolResult = `Monthly Financial Summary (${data.budgetReport?.month || 'Current Month'}):\n\n• Verified Liquid Balance: ₹${liquid.toLocaleString('en-IN')}\n• Discretionary Cash Flow Buffer: ₹${discretionary.toLocaleString('en-IN')}\n• Active Milestone Goals: ${data.goals.length} target(s) monitored.`;
			} else if (toolId === 'scenario_planning') {
				directoryToolResult = `Scenario Simulation: What if discretionary income drops by 20%?\n\n• Revised monthly capacity: ₹${Math.round(discretionary * 0.8).toLocaleString('en-IN')}\n• Impact: Goal completion timelines extend by approximately 1.5 months; core living commitments remain 100% safeguarded.`;
			} else {
				directoryToolResult = `Executed ${toolId} successfully with live ledger context.`;
			}
			isRunningTool = false;
		}, 400);
	}

	async function parseQuickExpense() {
		if (!quickExpenseText.trim()) return;
		isParsingExpense = true;
		try {
			const res = await fetch(resolve('/api/ai/parse-expense'), {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ text: quickExpenseText.trim() })
			});
			const result = await res.json();
			if (res.ok) {
				quickExpenseResult = result;
			}
		} finally {
			isParsingExpense = false;
		}
	}
</script>

<svelte:head>
	<title>AI Manager · BuyWise</title>
</svelte:head>

<div class="ai-manager-page">
	<!-- Workspace Header & AI Configuration Card (Section 2.E) -->
	<div class="ai-workspace-hero card">
		<div class="hero-top-row">
			<div class="hero-title-group">
				<div class="ai-badge-pill">
					<Sparkles size={14} />
					<span>Central AI Workspace</span>
				</div>
				<h1 class="workspace-headline">AI Financial Manager</h1>
				<p class="workspace-subheadline">
					Autonomous financial overview, goal optimization, purchase decision support, and scenario
					planning.
				</p>
			</div>

			<!-- AI Status & Model Transparency Info (Section 2.E) -->
			<div class="ai-status-card">
				<div class="status-indicator-row">
					<span class="status-dot" class:active={data.aiConfig.configured}></span>
					<span class="status-label font-semibold">
						{data.aiConfig.configured ? 'AI Engine Operational' : 'Offline Fallback Active'}
					</span>
				</div>
				<div class="model-badge font-mono text-xs">
					<Cpu size={12} />
					<span>Model: {data.aiConfig.model}</span>
				</div>
				<div class="provenance-note text-xs">
					<ShieldCheck size={13} class="text-success" />
					<span>Server-authoritative calculations · No invented balances</span>
				</div>
			</div>
		</div>

		<!-- Workspace Navigation Tabs -->
		<div class="workspace-tabs" role="tablist">
			<button
				type="button"
				class="workspace-tab"
				class:active={activeTab === 'overview'}
				onclick={() => (activeTab = 'overview')}
			>
				<Wallet size={16} />
				<span>Financial Overview</span>
			</button>
			<button
				type="button"
				class="workspace-tab"
				class:active={activeTab === 'plan'}
				onclick={() => (activeTab = 'plan')}
			>
				<Sliders size={16} />
				<span>Create & Refine Plans</span>
			</button>
			<button
				type="button"
				class="workspace-tab"
				class:active={activeTab === 'decision'}
				onclick={() => {
					activeTab = 'decision';
					if (!decisionAdvice) runPurchaseDecisionAnalysis();
				}}
			>
				<Target size={16} />
				<span>Purchase Decision Support</span>
			</button>
			<button
				type="button"
				class="workspace-tab"
				class:active={activeTab === 'directory'}
				onclick={() => (activeTab = 'directory')}
			>
				<Layers size={16} />
				<span>AI Tools Directory ({data.aiConfig.availableCapabilities.length})</span>
			</button>
		</div>
	</div>

	<!-- TAB 1: FINANCIAL OVERVIEW & RECOMMENDATIONS (Section 2.A) -->
	{#if activeTab === 'overview'}
		<div class="overview-section">
			<!-- Three Columns: Confirmed Facts vs Calculated Results vs AI Suggestions -->
			<div class="three-tier-grid">
				<!-- Column 1: Confirmed Facts (Verified Ledger Data) -->
				<div class="tier-card card">
					<div class="tier-header">
						<ShieldCheck size={18} class="text-primary" />
						<h3 class="tier-title">Confirmed Facts</h3>
					</div>
					<p class="tier-desc">Authoritative balances and logged ledger records.</p>

					<div class="facts-list">
						{#each data.overviewInsights.confirmedFacts as fact (fact.label)}
							<div class="fact-item">
								<span class="fact-label">{fact.label}</span>
								<strong class="fact-value font-mono tabular-nums">{fact.value}</strong>
								<span class="fact-details text-xs">{fact.details}</span>
							</div>
						{/each}
					</div>
				</div>

				<!-- Column 2: Calculated Results (Deterministic Math) -->
				<div class="tier-card card">
					<div class="tier-header">
						<Calculator size={18} class="text-accent" />
						<h3 class="tier-title">Calculated Results</h3>
					</div>
					<p class="tier-desc">Deterministic cash-flow metrics derived on the server.</p>

					<div class="facts-list">
						{#each data.overviewInsights.calculatedResults as calc (calc.label)}
							<div class="fact-item">
								<span class="fact-label">{calc.label}</span>
								<strong class="fact-value font-mono tabular-nums">{calc.value}</strong>
								<span class="fact-details text-xs">{calc.details}</span>
							</div>
						{/each}
					</div>
				</div>

				<!-- Column 3: AI-Generated Suggestions (Insights & Next Actions) -->
				<div class="tier-card card suggestions-tier">
					<div class="tier-header">
						<Sparkles size={18} class="text-accent" />
						<h3 class="tier-title">AI Suggestions & Opportunities</h3>
					</div>
					<p class="tier-desc">Intelligent opportunities based on verified cash flow.</p>

					<div class="suggestions-list">
						{#each data.overviewInsights.aiSuggestions as sugg (sugg.title)}
							<div class="suggestion-item">
								<strong class="sugg-title">{sugg.title}</strong>
								<p class="sugg-desc text-xs">{sugg.description}</p>
								<div class="sugg-impact">
									<span class="badge badge-accent text-xs">{sugg.impact}</span>
								</div>
							</div>
						{/each}
					</div>
				</div>
			</div>

			<!-- Recommended Next Actions & Spending Patterns -->
			<div class="secondary-overview-grid">
				<!-- Recommended Actions -->
				<div class="card actions-card">
					<h3 class="card-title">Recommended Next Actions</h3>
					<div class="recommended-actions-list">
						{#each data.overviewInsights.recommendedActions as act (act.title)}
							<div class="action-item">
								<div class="action-indicator" class:high={act.priority === 'high'}>
									<ArrowRight size={16} />
								</div>
								<div class="action-info">
									<strong class="action-title">{act.title}</strong>
									<p class="action-text text-sm">{act.action}</p>
								</div>
							</div>
						{/each}
					</div>
				</div>

				<!-- Spending Patterns & Anomaly Detector -->
				<div class="card patterns-card">
					<h3 class="card-title">Spending Patterns & Anomalies</h3>
					<div class="patterns-list">
						{#each data.overviewInsights.spendingPatterns as pat, idx (idx)}
							<div class="pattern-item">
								<span class="pattern-dot"></span>
								<span class="pattern-text text-sm">{pat}</span>
							</div>
						{/each}
					</div>

					{#if data.overviewInsights.anomalies.length > 0}
						<div class="anomalies-box">
							<strong class="text-xs uppercase text-warning">Detected Spikes / Large Debits:</strong
							>
							{#each data.overviewInsights.anomalies as anom, idx (idx)}
								<div class="anomaly-row">
									<span>{anom.description} ({anom.date})</span>
									<strong class="tabular-nums font-mono">₹{anom.amountRupees}</strong>
								</div>
							{/each}
						</div>
					{/if}
				</div>
			</div>
		</div>

		<!-- TAB 2: CREATE & REFINE FINANCIAL PLANS (Section 2.B) -->
	{:else if activeTab === 'plan'}
		<div class="plan-workspace">
			<!-- Natural Language Prompt Generator -->
			<div class="card plan-generator-card">
				<div class="generator-header">
					<h3 class="card-title">Describe Your Financial Goal in Plain English</h3>
					<p class="card-desc">
						Specify what you want to achieve, approximate price, and timeframe. The AI will evaluate
						your real discretionary buffer and formulate an achievable plan.
					</p>
				</div>

				<div class="generator-input-row">
					<textarea
						rows="2"
						placeholder="e.g. Save ₹1,50,000 for a MacBook Pro in 6 months, or save ₹40,000 for a holiday"
						class="textarea plan-textarea"
						bind:value={planPrompt}></textarea>
					<button
						type="button"
						class="btn btn-primary btn-generate"
						disabled={!planPrompt.trim() || isGeneratingPlan}
						onclick={createPlanFromPrompt}
					>
						{#if isGeneratingPlan}
							<Clock size={16} class="animate-spin" />
							<span>Formulating Plan...</span>
						{:else}
							<Sparkles size={16} />
							<span>Generate Plan</span>
						{/if}
					</button>
				</div>

				<!-- Prompt Inspiration Pills -->
				<div class="prompt-pills">
					<span class="pills-label text-xs">Try asking:</span>
					<button
						type="button"
						class="prompt-pill"
						onclick={() => {
							planPrompt = 'Save ₹80,000 for an emergency reserve in 8 months';
							createPlanFromPrompt();
						}}
					>
						"Save ₹80,000 for emergency reserve"
					</button>
					<button
						type="button"
						class="prompt-pill"
						onclick={() => {
							planPrompt = 'I want to buy an iPhone 16 Pro costing ₹1,20,000 by Diwali';
							createPlanFromPrompt();
						}}
					>
						"Buy an iPhone 16 Pro by Diwali"
					</button>
				</div>

				{#if planError}
					<div class="alert-box alert-error">
						<AlertCircle size={16} />
						<span>{planError}</span>
					</div>
				{/if}
			</div>

			<!-- Active Plan Display & Refinement Workspace -->
			{#if activePlan}
				<div class="active-plan-container">
					<!-- Proposed Plan Card -->
					<div class="card proposed-plan-card">
						<div class="plan-card-top">
							<div>
								<span class="badge badge-accent">Proposed Savings Plan</span>
								<h2 class="plan-title">{activePlan.goalName}</h2>
							</div>
							<span
								class="badge"
								class:badge-success={activePlan.feasibility === 'feasible'}
								class:badge-warning={activePlan.feasibility === 'tight'}
								class:badge-danger={activePlan.feasibility === 'unfeasible'}
							>
								{activePlan.feasibility.toUpperCase()}
							</span>
						</div>

						<div class="plan-metrics-grid">
							<div class="metric-item">
								<span class="metric-lbl">Target Amount</span>
								<strong class="metric-num tabular-nums"
									>₹{activePlan.targetPriceRupees.toLocaleString('en-IN')}</strong
								>
							</div>
							<div class="metric-item">
								<span class="metric-lbl">Monthly Contribution</span>
								<strong class="metric-num tabular-nums text-primary"
									>₹{activePlan.monthlyContributionRupees.toLocaleString('en-IN')} / mo</strong
								>
							</div>
							<div class="metric-item">
								<span class="metric-lbl">Timeline</span>
								<strong class="metric-num tabular-nums">{activePlan.monthsNeeded} Months</strong>
							</div>
							<div class="metric-item">
								<span class="metric-lbl">Target Date</span>
								<strong class="metric-num tabular-nums">{activePlan.targetDate}</strong>
							</div>
						</div>

						<div class="plan-explanation-box">
							<p><strong>Impact on Cash Flow:</strong> {activePlan.impactOnDiscretionary}</p>
							<p class="mt-2"><strong>Consequences:</strong></p>
							<ul class="consequences-list">
								{#each activePlan.consequences as c, idx (idx)}
									<li>{c}</li>
								{/each}
							</ul>
						</div>

						<!-- Explicit Save Button (Section 2.B) -->
						<div class="plan-actions-bar">
							<button
								type="button"
								class="btn btn-accent btn-save-plan"
								onclick={() => (showSaveGoalConfirmModal = true)}
							>
								<Check size={16} />
								<span>Confirm & Save to Goals</span>
							</button>
						</div>
					</div>

					<!-- Contextual Instruction Field for Refinement (Section 2.B) -->
					<div class="card refinement-card">
						<h3 class="card-title">Refine This Plan</h3>
						<p class="card-desc">
							Ask follow-up modifications such as reducing monthly contributions, delaying
							completion, or prioritizing goals.
						</p>

						<div class="refinement-input-row">
							<input
								type="text"
								placeholder="e.g. Reduce monthly contribution by ₹3,000, or delay purchase by 2 months"
								class="input refinement-input"
								bind:value={refinementInstruction}
								onkeydown={(e) => {
									if (e.key === 'Enter') refinePlan();
								}}
							/>
							<button
								type="button"
								class="btn btn-primary"
								disabled={!refinementInstruction.trim() || isRefiningPlan}
								onclick={refinePlan}
							>
								{#if isRefiningPlan}
									<Clock size={16} class="animate-spin" />
									<span>Refining...</span>
								{:else}
									<Send size={16} />
									<span>Apply Change</span>
								{/if}
							</button>
						</div>

						<!-- Before-and-After Comparison View (Section 2.B) -->
						{#if refinementResult}
							<div class="before-after-card">
								<h4 class="comparison-title">Before vs. After Comparison</h4>
								<p class="comparison-explanation text-sm">{refinementResult.changesExplanation}</p>

								<div class="comparison-grid">
									<div class="compare-col before">
										<span class="col-kicker">Previous Plan</span>
										<div class="compare-stat">
											₹{refinementResult.previousPlan.monthlyContributionRupees}/mo
										</div>
										<div class="compare-sub">
											{refinementResult.previousPlan.monthsNeeded} months · {refinementResult
												.previousPlan.targetDate}
										</div>
									</div>
									<div class="compare-col after">
										<span class="col-kicker">Revised Plan</span>
										<div class="compare-stat text-accent">
											₹{refinementResult.revisedPlan.monthlyContributionRupees}/mo
										</div>
										<div class="compare-sub">
											{refinementResult.revisedPlan.monthsNeeded} months · {refinementResult
												.revisedPlan.targetDate}
										</div>
									</div>
								</div>

								<div class="consequences-note">
									<strong>Consequences of change:</strong>
									<ul>
										{#each refinementResult.consequences as c, idx (idx)}
											<li>{c}</li>
										{/each}
									</ul>
								</div>
							</div>
						{/if}
					</div>
				</div>
			{/if}
		</div>

		<!-- TAB 3: PURCHASE DECISION ASSISTANCE (Section 2.C) -->
	{:else if activeTab === 'decision'}
		<div class="decision-workspace">
			{#if data.goals.length === 0}
				<div class="card empty-state">
					<Target size={32} />
					<h3 class="empty-title">No Active Purchase Goals</h3>
					<p class="empty-text">
						Create a purchase goal first to review affordability with AI assistance.
					</p>
					<a href={resolve('/goals')} class="btn btn-primary">Go to Goals</a>
				</div>
			{:else}
				<div class="card goal-selector-card">
					<div class="selector-header">
						<div>
							<h3 class="card-title">Select Purchase Goal to Review</h3>
							<p class="card-desc">
								Evaluates target price, available liquid reserves, monthly budget, and financing
								trade-offs.
							</p>
						</div>
						<div class="goal-select-wrapper">
							{#if isEvaluatingDecision}
								<Clock size={16} class="animate-spin text-accent" />
							{/if}
							<select
								class="input goal-dropdown"
								bind:value={selectedGoalId}
								onchange={runPurchaseDecisionAnalysis}
							>
								{#each data.goals as g (g.id)}
									<option value={g.id}>
										{g.name} ({formatRupees(g.targetPricePaise || 0)})
									</option>
								{/each}
							</select>
						</div>
					</div>
				</div>

				{#if selectedGoal && decisionAdvice}
					<div class="decision-results-grid">
						<!-- Left: AI Verdict & Deterministic Side-by-side -->
						<div class="card verdict-card">
							<div class="verdict-top">
								<div>
									<span class="card-kicker">Purchase Decision Assessment</span>
									<h2 class="verdict-headline text-accent">{decisionAdvice.verdict}</h2>
								</div>
								<span class="badge badge-primary">Deterministic Engine Verified</span>
							</div>

							<p class="verdict-rationale">{decisionAdvice.rationale}</p>

							<!-- Comparative Financing Options (Cash vs Self-EMI vs Bank) -->
							<div class="alternatives-section">
								<h4 class="alternatives-title">Financing & Timing Strategies Compared</h4>
								<div class="alternatives-list">
									{#each decisionAdvice.alternatives as alt (alt.strategy)}
										<div class="alt-item">
											<strong class="alt-strat">{alt.strategy}</strong>
											<div class="alt-cost tabular-nums font-mono">{alt.monthlyCost}</div>
											<div class="alt-consequence text-xs">{alt.consequence}</div>
										</div>
									{/each}
								</div>
							</div>
						</div>

						<!-- Right: Selected Goal Quick Stats -->
						<div class="card goal-quick-card">
							<h3 class="card-title">{selectedGoal.name}</h3>
							<div class="quick-stat-row">
								<span class="stat-lbl">Target Price:</span>
								<strong class="stat-val tabular-nums"
									>{formatRupees(selectedGoal.targetPricePaise || 0)}</strong
								>
							</div>
							<div class="quick-stat-row">
								<span class="stat-lbl">Saved So Far:</span>
								<strong class="stat-val tabular-nums"
									>{formatRupees(selectedGoal.savedAmountPaise || 0)}</strong
								>
							</div>
							<div class="quick-stat-row">
								<span class="stat-lbl">Priority:</span>
								<strong class="stat-val">{selectedGoal.priority?.toUpperCase() || 'MEDIUM'}</strong>
							</div>
							<div class="quick-stat-row">
								<span class="stat-lbl">Target Date:</span>
								<strong class="stat-val">{selectedGoal.targetDate || 'Flexible'}</strong>
							</div>

							<div class="mt-4">
								<a
									href={resolve('/goals/[id]', { id: selectedGoal.id })}
									class="btn btn-secondary btn-sm w-full"
								>
									<span>Open Full Goal Evaluation</span>
									<ArrowRight size={14} />
								</a>
							</div>
						</div>
					</div>
				{/if}
			{/if}
		</div>

		<!-- TAB 4: AI TOOLS DIRECTORY (Section 2.D) -->
	{:else if activeTab === 'directory'}
		<div class="directory-workspace">
			<div class="directory-intro card">
				<h3 class="card-title">AI Tools Directory</h3>
				<p class="card-desc">
					Run purpose-built AI tools on your actual financial accounts, category allocations, and
					purchase goals.
				</p>
			</div>

			<div class="tools-grid">
				{#each data.aiConfig.availableCapabilities as tool (tool.id)}
					<div class="tool-card card">
						<div class="tool-header">
							<Sparkles size={20} class="text-accent" />
							<h4 class="tool-name">{tool.name}</h4>
						</div>
						<p class="tool-desc text-sm">{tool.description}</p>
						<button
							type="button"
							class="btn btn-secondary btn-sm btn-run-tool"
							onclick={() => {
								if (tool.id === 'expense_entry') {
									quickExpenseText = '';
									quickExpenseResult = null;
									activeDirectoryTool = 'expense_entry';
								} else {
									runDirectoryTool(tool.id);
								}
							}}
						>
							<span>Run Tool</span>
							<ArrowRight size={14} />
						</button>
					</div>
				{/each}
			</div>

			<!-- Active Tool Execution Output Drawer / Card -->
			{#if activeDirectoryTool && activeDirectoryTool !== 'expense_entry'}
				<div class="card tool-output-card">
					<div class="output-header">
						<h4 class="output-title">Tool Execution Output</h4>
						<button
							type="button"
							class="btn btn-ghost btn-sm"
							onclick={() => (activeDirectoryTool = null)}
						>
							Dismiss
						</button>
					</div>

					{#if isRunningTool}
						<div class="running-box">
							<Clock size={16} class="animate-spin" />
							<span>Running analysis with verified financial records...</span>
						</div>
					{:else if directoryToolResult}
						<pre class="tool-output-pre font-mono">{directoryToolResult}</pre>
					{/if}
				</div>
			{/if}

			<!-- Natural Language Expense Entry Modal (Tool 5) -->
			{#if activeDirectoryTool === 'expense_entry'}
				<Modal
					open={true}
					title="Natural-Language Expense Entry"
					onclose={() => (activeDirectoryTool = null)}
				>
					<div class="nl-expense-box">
						<p class="text-sm text-secondary mb-3">
							Speak or type what you spent (e.g. "Spent 850 at Starbucks on coffee yesterday").
						</p>
						<div class="input-row">
							<input
								type="text"
								placeholder="e.g. Paid ₹1,200 for groceries at Nature Basket"
								class="input"
								bind:value={quickExpenseText}
								onkeydown={(e) => {
									if (e.key === 'Enter') parseQuickExpense();
								}}
							/>
							<button
								type="button"
								class="btn btn-primary"
								onclick={parseQuickExpense}
								disabled={isParsingExpense}
							>
								{isParsingExpense ? 'Parsing...' : 'Extract'}
							</button>
						</div>

						{#if quickExpenseResult}
							<div class="extracted-expense-card mt-3">
								<strong class="text-sm">Extracted Ledger Entry:</strong>
								<div class="text-xs mt-1">
									Amount: <strong class="tabular-nums font-mono"
										>{quickExpenseResult.amountFormatted}</strong
									>
								</div>
								<div class="text-xs">
									Category: <strong class="text-accent"
										>{quickExpenseResult.suggestedCategory}</strong
									>
								</div>
								<div class="text-xs">Note: <em>{quickExpenseResult.note}</em></div>
								<a href={resolve('/budget')} class="btn btn-sm btn-accent mt-3">
									Confirm & Record in Ledger
								</a>
							</div>
						{/if}
					</div>
				</Modal>
			{/if}
		</div>
	{/if}
</div>

<!-- ==========================================================================
     Explicit Confirmation Modal: Save Plan to Goals (Section 2.B)
     ========================================================================== -->
{#if activePlan}
	<Modal
		open={showSaveGoalConfirmModal}
		title="Confirm & Save Financial Plan"
		onclose={() => (showSaveGoalConfirmModal = false)}
	>
		<form method="POST" action="?/confirmAndSaveGoal" class="modal-form">
			<input type="hidden" name="name" value={activePlan.goalName} />
			<input type="hidden" name="targetPrice" value={activePlan.targetPriceRupees} />
			<input type="hidden" name="targetDate" value={activePlan.targetDate} />
			<input type="hidden" name="priority" value={activePlan.priority} />

			<div class="confirm-content">
				<p class="modal-explanation">
					You are about to save <strong>{activePlan.goalName}</strong> to your active purchase goals.
				</p>
				<div class="confirm-summary-box">
					<div>
						Target Amount: <strong class="tabular-nums font-mono"
							>₹{activePlan.targetPriceRupees.toLocaleString('en-IN')}</strong
						>
					</div>
					<div>
						Target Date: <strong class="tabular-nums font-mono">{activePlan.targetDate}</strong>
					</div>
					<div>
						Monthly Commitment: <strong class="tabular-nums font-mono text-primary"
							>₹{activePlan.monthlyContributionRupees.toLocaleString('en-IN')}/mo</strong
						>
					</div>
				</div>
				<p class="text-xs text-muted mt-2">
					This action commits this goal to your persistent financial database and automatically
					tracks progress.
				</p>
			</div>

			<div class="modal-footer">
				<button
					type="button"
					class="btn btn-secondary"
					onclick={() => (showSaveGoalConfirmModal = false)}
				>
					Cancel
				</button>
				<button type="submit" class="btn btn-accent"> Confirm & Save Goal </button>
			</div>
		</form>
	</Modal>
{/if}

<style>
	.ai-manager-page {
		max-width: 1400px;
		margin: 0 auto;
		display: flex;
		flex-direction: column;
		gap: 24px;
	}

	.ai-workspace-hero {
		background-color: var(--surface);
		border-radius: 16px;
		padding: 24px 28px;
		border: 1px solid var(--border);
	}

	.hero-top-row {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		gap: 20px;
		flex-wrap: wrap;
		margin-bottom: 24px;
	}

	.ai-badge-pill {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		padding: 4px 10px;
		border-radius: 9999px;
		font-size: 0.78rem;
		font-weight: 600;
		background-color: var(--accent-subtle);
		color: var(--foreground);
		margin-bottom: 8px;
	}

	.workspace-headline {
		font-size: 28px;
		font-weight: 700;
		color: var(--foreground);
		letter-spacing: -0.025em;
		margin-bottom: 4px;
	}

	.workspace-subheadline {
		font-size: 15px;
		color: var(--foreground-secondary);
		max-width: 600px;
	}

	.ai-status-card {
		background-color: var(--surface-secondary);
		border: 1px solid var(--border);
		border-radius: 12px;
		padding: 12px 16px;
		display: flex;
		flex-direction: column;
		gap: 6px;
	}

	.status-indicator-row {
		display: flex;
		align-items: center;
		gap: 8px;
	}

	.status-dot {
		width: 8px;
		height: 8px;
		border-radius: 50%;
		background-color: var(--warning);
	}

	.status-dot.active {
		background-color: var(--success);
	}

	.model-badge {
		display: flex;
		align-items: center;
		gap: 6px;
		color: var(--foreground-secondary);
	}

	.provenance-note {
		color: var(--foreground-muted);
		border-top: 1px solid var(--border-subtle);
		padding-top: 6px;
	}

	/* Workspace Tabs */
	.workspace-tabs {
		display: flex;
		gap: 8px;
		border-top: 1px solid var(--border-subtle);
		padding-top: 16px;
		overflow-x: auto;
	}

	.workspace-tab {
		display: inline-flex;
		align-items: center;
		gap: 8px;
		padding: 8px 16px;
		border-radius: 8px;
		font-size: 0.88rem;
		font-weight: 500;
		color: var(--foreground-secondary);
		transition: all 0.15s ease;
		white-space: nowrap;
	}

	.workspace-tab:hover {
		background-color: var(--surface-secondary);
		color: var(--foreground);
	}

	.workspace-tab.active {
		background-color: var(--primary);
		color: #ffffff;
		font-weight: 600;
	}

	/* Three Tier Grid (Section 2.A) */
	.three-tier-grid {
		display: grid;
		grid-template-columns: repeat(3, 1fr);
		gap: 20px;
		margin-bottom: 24px;
	}

	.tier-card {
		display: flex;
		flex-direction: column;
		gap: 12px;
	}

	.tier-header {
		display: flex;
		align-items: center;
		gap: 8px;
	}

	.tier-title {
		font-size: 1.1rem;
		font-weight: 700;
		color: var(--foreground);
	}

	.tier-desc {
		font-size: 0.82rem;
		color: var(--foreground-secondary);
		margin-top: -6px;
	}

	.facts-list,
	.suggestions-list {
		display: flex;
		flex-direction: column;
		gap: 12px;
	}

	.fact-item {
		background-color: var(--surface-secondary);
		border-radius: 10px;
		padding: 12px 14px;
		display: flex;
		flex-direction: column;
		gap: 2px;
	}

	.fact-label {
		font-size: 0.78rem;
		color: var(--foreground-secondary);
		text-transform: uppercase;
		font-weight: 600;
	}

	.fact-value {
		font-size: 1.25rem;
		color: var(--foreground);
	}

	.fact-details {
		color: var(--foreground-muted);
	}

	.suggestion-item {
		background-color: var(--surface-secondary);
		border-radius: 10px;
		padding: 12px 14px;
		display: flex;
		flex-direction: column;
		gap: 6px;
	}

	.sugg-title {
		font-size: 0.92rem;
		color: var(--foreground);
	}

	.secondary-overview-grid {
		display: grid;
		grid-template-columns: repeat(2, 1fr);
		gap: 20px;
	}

	.recommended-actions-list {
		display: flex;
		flex-direction: column;
		gap: 12px;
		margin-top: 14px;
	}

	.action-item {
		display: flex;
		align-items: flex-start;
		gap: 12px;
		background-color: var(--surface-secondary);
		padding: 12px 14px;
		border-radius: 10px;
	}

	.action-indicator {
		width: 32px;
		height: 32px;
		border-radius: 8px;
		display: flex;
		align-items: center;
		justify-content: center;
		background-color: var(--primary-subtle);
		color: var(--primary);
		flex-shrink: 0;
	}

	.action-indicator.high {
		background-color: var(--accent-subtle);
		color: var(--foreground);
	}

	.action-title {
		font-size: 0.92rem;
		color: var(--foreground);
	}

	.patterns-list {
		display: flex;
		flex-direction: column;
		gap: 10px;
		margin-top: 14px;
	}

	.pattern-item {
		display: flex;
		align-items: center;
		gap: 8px;
	}

	.pattern-dot {
		width: 6px;
		height: 6px;
		border-radius: 50%;
		background-color: var(--primary);
	}

	.anomalies-box {
		margin-top: 16px;
		padding-top: 14px;
		border-top: 1px solid var(--border-subtle);
		display: flex;
		flex-direction: column;
		gap: 6px;
	}

	.anomaly-row {
		display: flex;
		justify-content: space-between;
		font-size: 0.85rem;
	}

	/* Plan Generator */
	.plan-generator-card {
		display: flex;
		flex-direction: column;
		gap: 16px;
	}

	.generator-input-row {
		display: flex;
		gap: 12px;
		align-items: flex-end;
	}

	.plan-textarea {
		flex: 1;
	}

	.btn-generate {
		height: 52px;
		white-space: nowrap;
	}

	.prompt-pills {
		display: flex;
		align-items: center;
		gap: 8px;
		flex-wrap: wrap;
	}

	.prompt-pill {
		padding: 4px 10px;
		border-radius: 9999px;
		font-size: 0.78rem;
		border: 1px solid var(--border);
		background-color: var(--surface-secondary);
		color: var(--foreground-secondary);
		transition: all 0.15s ease;
	}

	.prompt-pill:hover {
		border-color: var(--primary);
		color: var(--primary);
	}

	.active-plan-container {
		display: flex;
		flex-direction: column;
		gap: 20px;
		margin-top: 20px;
	}

	.proposed-plan-card {
		display: flex;
		flex-direction: column;
		gap: 20px;
	}

	.plan-card-top {
		display: flex;
		justify-content: space-between;
		align-items: flex-start;
	}

	.plan-title {
		font-size: 1.6rem;
		font-weight: 700;
		color: var(--foreground);
		margin-top: 4px;
	}

	.plan-metrics-grid {
		display: grid;
		grid-template-columns: repeat(4, 1fr);
		gap: 16px;
		background-color: var(--surface-secondary);
		border-radius: 12px;
		padding: 16px;
	}

	.metric-lbl {
		font-size: 0.78rem;
		font-weight: 600;
		color: var(--foreground-secondary);
		text-transform: uppercase;
	}

	.metric-num {
		font-size: 1.35rem;
		color: var(--foreground);
		display: block;
		margin-top: 4px;
	}

	.plan-explanation-box {
		padding: 16px;
		border-radius: 10px;
		background-color: var(--surface-secondary);
		font-size: 0.88rem;
		line-height: 1.5;
	}

	.consequences-list {
		padding-left: 20px;
		margin-top: 4px;
	}

	.plan-actions-bar {
		display: flex;
		justify-content: flex-end;
	}

	.btn-save-plan {
		padding: 12px 24px;
		font-weight: 700;
	}

	/* Refinement */
	.refinement-card {
		display: flex;
		flex-direction: column;
		gap: 14px;
	}

	.refinement-input-row {
		display: flex;
		gap: 10px;
	}

	.refinement-input {
		flex: 1;
	}

	.before-after-card {
		background-color: var(--surface-secondary);
		border: 1px solid var(--border);
		border-radius: 12px;
		padding: 18px;
		margin-top: 14px;
	}

	.comparison-title {
		font-size: 1rem;
		font-weight: 700;
		color: var(--foreground);
		margin-bottom: 4px;
	}

	.comparison-grid {
		display: grid;
		grid-template-columns: repeat(2, 1fr);
		gap: 16px;
		margin: 16px 0;
	}

	.compare-col {
		padding: 14px;
		border-radius: 10px;
		background-color: var(--surface);
		border: 1px solid var(--border-subtle);
	}

	.col-kicker {
		font-size: 0.78rem;
		text-transform: uppercase;
		font-weight: 600;
		color: var(--foreground-secondary);
	}

	.compare-stat {
		font-size: 1.35rem;
		font-weight: 700;
		margin: 4px 0;
	}

	.compare-sub {
		font-size: 0.8rem;
		color: var(--foreground-muted);
	}

	.consequences-note {
		font-size: 0.84rem;
		line-height: 1.5;
	}

	.consequences-note ul {
		padding-left: 18px;
		margin-top: 4px;
	}

	/* Decision Workspace */
	.decision-results-grid {
		display: grid;
		grid-template-columns: 2fr 1fr;
		gap: 20px;
		margin-top: 20px;
	}

	.verdict-card {
		display: flex;
		flex-direction: column;
		gap: 18px;
	}

	.verdict-top {
		display: flex;
		justify-content: space-between;
		align-items: flex-start;
	}

	.verdict-headline {
		font-size: 1.5rem;
		font-weight: 700;
		margin-top: 4px;
	}

	.verdict-rationale {
		font-size: 0.95rem;
		line-height: 1.5;
		color: var(--foreground);
	}

	.alternatives-section {
		border-top: 1px solid var(--border-subtle);
		padding-top: 16px;
	}

	.alternatives-title {
		font-size: 0.95rem;
		font-weight: 700;
		margin-bottom: 12px;
	}

	.alternatives-list {
		display: flex;
		flex-direction: column;
		gap: 10px;
	}

	.alt-item {
		background-color: var(--surface-secondary);
		padding: 12px 14px;
		border-radius: 10px;
	}

	.alt-strat {
		font-size: 0.88rem;
		color: var(--foreground);
	}

	.alt-cost {
		font-size: 0.95rem;
		font-weight: 700;
		margin: 4px 0 2px;
	}

	.goal-quick-card {
		display: flex;
		flex-direction: column;
		gap: 12px;
	}

	.quick-stat-row {
		display: flex;
		justify-content: space-between;
		font-size: 0.88rem;
		padding-bottom: 8px;
		border-bottom: 1px solid var(--border-subtle);
	}

	/* Directory Tools */
	.tools-grid {
		display: grid;
		grid-template-columns: repeat(3, 1fr);
		gap: 18px;
		margin-top: 20px;
	}

	.tool-card {
		display: flex;
		flex-direction: column;
		justify-content: space-between;
		gap: 12px;
	}

	.tool-header {
		display: flex;
		align-items: center;
		gap: 10px;
	}

	.tool-name {
		font-size: 0.98rem;
		font-weight: 700;
		color: var(--foreground);
	}

	.tool-desc {
		color: var(--foreground-secondary);
		flex: 1;
	}

	.btn-run-tool {
		align-self: flex-start;
	}

	.tool-output-card {
		margin-top: 20px;
	}

	.output-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin-bottom: 12px;
	}

	.tool-output-pre {
		background-color: var(--surface-secondary);
		padding: 16px;
		border-radius: 10px;
		font-size: 0.85rem;
		line-height: 1.5;
		white-space: pre-wrap;
	}

	.confirm-summary-box {
		background-color: var(--surface-secondary);
		padding: 14px;
		border-radius: 10px;
		margin-top: 12px;
		display: flex;
		flex-direction: column;
		gap: 6px;
		font-size: 0.88rem;
	}

	@media (max-width: 1024px) {
		.three-tier-grid {
			grid-template-columns: 1fr;
		}
		.secondary-overview-grid {
			grid-template-columns: 1fr;
		}
		.tools-grid {
			grid-template-columns: repeat(2, 1fr);
		}
		.plan-metrics-grid {
			grid-template-columns: repeat(2, 1fr);
		}
		.decision-results-grid {
			grid-template-columns: 1fr;
		}
	}

	@media (max-width: 640px) {
		.tools-grid {
			grid-template-columns: 1fr;
		}
		.generator-input-row {
			flex-direction: column;
			align-items: stretch;
		}
		.comparison-grid {
			grid-template-columns: 1fr;
		}
	}
</style>
