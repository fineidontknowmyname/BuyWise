<script lang="ts">
	import { untrack } from 'svelte';
	import { resolve } from '$app/paths';
	import { reverseEmiDraft } from '#lib/reverseEmi.js';
	import { formatRupees, parseRupees } from '#lib/money.js';
	import type { Account } from '#lib/server/api.js';
	import type { MoneyPlan } from '#lib/moneyPlan.js';
	let {
		target,
		money,
		accounts,
		initial
	}: {
		target: string;
		money: MoneyPlan | null;
		accounts: Account[];
		initial: { saved: string; monthly: string; firstDate: string };
	} = $props();
	let saved = $state(untrack(() => initial.saved));
	let monthly = $state(untrack(() => initial.monthly));
	let firstDate = $state(untrack(() => initial.firstDate));
	const draft = $derived(reverseEmiDraft(target, saved, monthly, firstDate));
</script>

<section class="saving-fields">
	<h3>Your saving plan</h3>
	<p>
		{money?.configured
			? `${formatRupees(money.unassignedMonthlyPaise)} per month left for a new goal · ${formatRupees(money.savingsNowPaise)} free savings now.`
			: 'Set your monthly income and spending in My Money before committing monthly savings.'}
	</p>
	<div class="saving-grid">
		<label
			>Use existing savings (₹)<input
				name="savingsNow"
				inputmode="decimal"
				bind:value={saved}
				required
			/></label
		>
		<label
			>Save each month (₹)<input
				name="monthlyContribution"
				inputmode="decimal"
				bind:value={monthly}
			/></label
		>
		<label
			>First saving date<input
				name="firstSavingOn"
				type="date"
				bind:value={firstDate}
				required
			/></label
		>
		{#if (parseRupees(saved) ?? 0) > 0}<label
				>Account holding those savings<select name="savingsAccount" required
					><option value="">Choose account</option>{#each accounts as account (account.id)}<option
							value={account.id}>{account.name}</option
						>{/each}</select
				></label
			>{/if}
	</div>
	{#if draft.completionDate}<div class="saving-preview">
			<span>Estimated completion</span><strong
				>{new Intl.DateTimeFormat('en-IN', { dateStyle: 'medium' }).format(
					new Date(draft.completionDate + 'T12:00:00Z')
				)}</strong
			>
		</div>{/if}
	<p>
		Monthly amounts are a plan. Progress increases when you record savings. <a
			href={resolve('/accounts')}>Review My Money →</a
		>
	</p>
</section>

<style>
	.saving-fields {
		display: grid;
		gap: 14px;
		padding: 20px;
		border: 1px solid var(--border);
		border-radius: 14px;
	}
	.saving-fields h3 {
		font-size: 16px;
		margin: 0;
	}
	.saving-fields p {
		font-size: 12px;
		color: var(--foreground-secondary);
		margin: 0;
	}
	.saving-grid {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 14px;
	}
	.saving-grid label {
		display: grid;
		gap: 6px;
		font-size: 13px;
	}
	.saving-grid input,
	.saving-grid select {
		width: 100%;
		min-width: 0;
	}
	.saving-preview {
		display: flex;
		flex-wrap: wrap;
		justify-content: space-between;
		gap: 8px;
		font-size: 13px;
	}
	@media (max-width: 600px) {
		.saving-grid {
			grid-template-columns: 1fr;
		}
	}
</style>
