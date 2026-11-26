<script lang="ts">
	import '#lib/styles/workbench.css';
	import { untrack } from 'svelte';
	import { resolve } from '$app/paths';
	import { formatRupees, parseRupees } from '#lib/money.js';
	import { reverseEmiDraft } from '#lib/reverseEmi.js';
	import { calculateEMI, generateAmortizationSchedule } from '#lib/emi.js';
	import type { PageProps } from './$types';
	let { data }: PageProps = $props();
	let target = $state('');
	let saved = $state('0');
	let monthly = $state('');
	let firstDate = $state(untrack(() => data.firstSavingDate));
	let selectedProduct = $state('');
	let selectedGoal = $state('');
	let rate = $state<number | undefined>(12);
	let loanMonths = $state<number | undefined>(12);
	let loanDown = $state('0');
	let fee = $state('0');
	const draft = $derived(reverseEmiDraft(target, saved, monthly, firstDate));
	const progress = $derived(
		draft.targetPaise > 0
			? Math.round(((draft.targetPaise - draft.remainingPaise) / draft.targetPaise) * 100)
			: 0
	);
	const completion = $derived(
		draft.completionDate
			? new Intl.DateTimeFormat('en-IN', { dateStyle: 'medium', timeZone: 'Asia/Kolkata' }).format(
					new Date(draft.completionDate + 'T12:00:00Z')
				)
			: null
	);
	const loan = $derived.by(() => {
		const down = parseRupees(loanDown);
		const charges = parseRupees(fee);
		if (
			!draft.targetPaise ||
			down === null ||
			down > draft.targetPaise ||
			charges === null ||
			rate === undefined ||
			!Number.isFinite(rate) ||
			rate < 0 ||
			rate > 60 ||
			loanMonths === undefined ||
			!Number.isInteger(loanMonths) ||
			loanMonths < 1 ||
			loanMonths > 120
		)
			return null;
		return calculateEMI({
			productPricePaise: draft.targetPaise,
			downPaymentPaise: down,
			annualRatePercent: rate,
			tenureMonths: loanMonths,
			processingFeePaise: charges
		});
	});
	const schedule = $derived(
		loan
			? generateAmortizationSchedule(
					loan.principalPaise,
					rate ?? 0,
					loanMonths ?? 12,
					loan.monthlyInstallmentPaise
				)
			: []
	);
	function useProduct(id: string) {
		selectedProduct = id;
		const product = data.products.find((p) => p.id === id);
		if (product)
			target =
				product.currentPricePaise || product.targetPricePaise
					? ((product.currentPricePaise || product.targetPricePaise) / 100).toFixed(2)
					: '';
	}
</script>

<svelte:head><title>Reverse EMI calculator | BuyWise</title></svelte:head>
<div class="workbench">
	<header class="wb-heading">
		<div>
			<p class="wb-eyebrow">Explore a purchase</p>
			<h1>Save first. Buy on your terms.</h1>
			<p class="wb-muted">Estimate how a monthly saving amount turns into a purchase.</p>
		</div>
		<span class="wb-badge">Draft · not saved</span>
	</header>
	{#if data.error}<p class="wb-notice" role="status">{data.error}</p>{/if}
	<div class="wb-columns calculator-columns">
		<div class="wb-stack">
			<section class="wb-card">
				<div class="wb-card-head">
					<h2>Your purchase</h2>
					<button
						class="btn btn-ghost"
						type="button"
						onclick={() => {
							target = '30000';
							saved = '0';
							monthly = '3000';
							selectedProduct = '';
						}}>Try an example</button
					>
				</div>
				<label
					>Purchase target (₹)<input
						inputmode="decimal"
						bind:value={target}
						placeholder="30,000"
						aria-label="Purchase target in rupees"
					/></label
				>
				<div class="wb-fields">
					<label>Already saved (₹)<input inputmode="decimal" bind:value={saved} /></label><label
						>Save each month (₹)<input
							inputmode="decimal"
							bind:value={monthly}
							placeholder="3,000"
						/></label
					>
				</div>
				<label>First saving date<input type="date" bind:value={firstDate} min={data.today} /></label
				>
				{#if data.products.length}<details>
						<summary class="small-summary">Use a saved product price</summary><label
							class="extra-fields"
							>Tracked product<select
								value={selectedProduct}
								onchange={(e) => useProduct(e.currentTarget.value)}
								><option value="">Choose a product</option
								>{#each data.products as product (product.id)}<option value={product.id}
										>{product.name}{product.currentPricePaise
											? ' · saved price'
											: product.targetPricePaise
												? ' · saved target'
												: ' · enter price'}</option
									>{/each}</select
							></label
						>
						<p class="wb-muted extra-fields">
							This fills the amount only. Check the retailer for a current price.
						</p>
					</details>{/if}
			</section>
			<details class="wb-fold">
				<summary>Compare with a loan</summary>
				<div class="wb-fold-body">
					<p class="wb-muted">Estimate the cost of borrowing for the same purchase.</p>
					<div class="wb-fields">
						<label>Down payment (₹)<input inputmode="decimal" bind:value={loanDown} /></label><label
							>Term (months)<input
								type="number"
								min="1"
								max="120"
								step="1"
								bind:value={loanMonths}
							/></label
						><label
							>Annual interest (%)<input
								type="number"
								min="0"
								max="60"
								step="0.1"
								bind:value={rate}
							/></label
						><label>Fees (₹)<input inputmode="decimal" bind:value={fee} /></label>
					</div>
					{#if loan}<dl class="wb-lines">
							<div>
								<dt>Monthly loan payment</dt>
								<dd>{formatRupees(loan.monthlyInstallmentPaise)}</dd>
							</div>
							<div>
								<dt>Interest + fees</dt>
								<dd>{formatRupees(loan.extraCostVsCashPaise)}</dd>
							</div>
							<div>
								<dt>Total purchase cost</dt>
								<dd>{formatRupees(loan.totalRepaymentPaise)}</dd>
							</div>
						</dl>
						<details>
							<summary class="small-summary">Repayment schedule</summary>
							<div class="wb-table-wrap">
								<table>
									<thead
										><tr><th>Month</th><th>Payment</th><th>Interest</th><th>Balance</th></tr></thead
									><tbody
										>{#each schedule as row (row.month)}<tr
												><td>{row.month}</td><td>{formatRupees(row.emiPaise)}</td><td
													>{formatRupees(row.interestPaise)}</td
												><td>{formatRupees(row.closingBalancePaise)}</td></tr
											>{/each}</tbody
									>
								</table>
							</div>
						</details>{:else}<p class="wb-muted">
							Enter a target and valid loan terms to compare.
						</p>{/if}
				</div>
			</details>
		</div>
		<aside class="wb-stack wb-sticky">
			<section class="wb-card saving-result" aria-label="Draft saving estimate" aria-live="polite">
				<p class="wb-eyebrow">Your saving timeline</p>
				{#if !target}<div class="result-empty">
						<span aria-hidden="true">↗</span>
						<h2>A little each month.<br />A purchase to look forward to.</h2>
						<p class="wb-muted">Enter a target and your saving pace to see a date.</p>
					</div>{:else if draft.error}<p class="wb-notice" role="status">
						{draft.error}
					</p>{:else if draft.contributions === 0}<p class="wb-amount">Target reached</p>
					<p class="wb-muted">
						Your entered savings cover this target. Check your available funds before buying.
					</p>{:else}<p class="wb-amount">
						{draft.contributions}<span class="amount-unit"> monthly contributions</span>
					</p>
					<p class="completion">Around {completion}</p>
					<progress aria-label="Entered savings progress" value={progress} max="100"></progress>
					<dl class="wb-lines">
						<div>
							<dt>Still to save</dt>
							<dd>{formatRupees(draft.remainingPaise)}</dd>
						</div>
						<div>
							<dt>Each month</dt>
							<dd>{formatRupees(parseRupees(monthly) ?? 0)}</dd>
						</div>
						<div>
							<dt>Interest paid</dt>
							<dd>₹0</dd>
						</div>
					</dl>
					<p class="wb-muted">Assumes a fixed price and regular monthly contributions.</p>
					<a class="btn btn-primary" href={resolve('/goals')}>Create a purchase goal →</a>{/if}
			</section>
			<section class="wb-card">
				<h2>Already have a goal?</h2>
				<p class="wb-muted">
					Open its saved plan to use actual contributions, account funding and shared capacity.
				</p>
				{#if data.goals.length}<label
						>Purchase goal<select bind:value={selectedGoal}
							><option value="">Choose your goal</option>{#each data.goals as goal (goal.id)}<option
									value={goal.id}>{goal.name}</option
								>{/each}</select
						></label
					>{#if selectedGoal}<a
							class="btn btn-secondary"
							href={resolve('/goals/[id]/plan', { id: selectedGoal })}>Open saved plan →</a
						>{/if}{:else}<a href={resolve('/goals')}>View purchase goals →</a>{/if}
			</section>
		</aside>
	</div>
</div>

<style>
	.calculator-columns {
		grid-template-columns: minmax(0, 1fr) minmax(0, 1.1fr);
	}
	.saving-result {
		border-top: 3px solid var(--brand-dot);
	}
	.small-summary {
		cursor: pointer;
		font-size: 12px;
		color: var(--foreground-secondary);
	}
	.extra-fields {
		margin-top: 14px;
	}
	.amount-unit {
		font-size: 18px;
		font-weight: 450;
		letter-spacing: -0.02em;
	}
	.completion {
		font-size: 18px;
		font-weight: 500;
		color: var(--foreground);
	}
	.result-empty {
		display: grid;
		gap: 18px;
		padding-block: 18px;
	}
	.result-empty > span {
		font-size: 36px;
		color: var(--brand-dot);
	}
	.result-empty h2 {
		font-size: 24px;
		line-height: 1.35;
		font-weight: 500;
	}
	@media (max-width: 1200px) {
		.calculator-columns {
			grid-template-columns: 1fr;
		}
	}
</style>
