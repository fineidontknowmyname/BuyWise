<script lang="ts">
	import { formatRupees } from '#lib/money.js';
	let { weeks }: { weeks: { label: string; income: number; expenses: number }[] | null } = $props();
	const maximum = $derived(
		Math.max(1, ...(weeks ?? []).flatMap(({ income, expenses }) => [income, expenses]))
	);
</script>

<section class="card" aria-labelledby="cash-flow-title">
	<h3 id="cash-flow-title">Monthly cash flow</h3>
	{#if !weeks}<p>Cash flow figures are temporarily unavailable.</p>
	{:else if !weeks.some((week) => week.income || week.expenses)}<p>
			No income or expenses recorded this month.
		</p>
	{:else}
		{#each weeks as week (week.label)}
			<div class="week">
				<strong>{week.label}</strong>
				<div class="bar">
					<span>Income</span><i style:width={(week.income / maximum) * 100 + '%'}></i><span
						>{formatRupees(week.income)}</span
					>
				</div>
				<div class="bar expense">
					<span>Expenses</span><i style:width={(week.expenses / maximum) * 100 + '%'}></i><span
						>{formatRupees(week.expenses)}</span
					>
				</div>
			</div>
		{/each}
	{/if}
</section>

<style>
	section {
		padding: clamp(1rem, 3vw, 1.5rem);
	}
	h3 {
		margin: 0 0 1rem;
	}
	.week {
		display: grid;
		gap: 0.35rem;
		margin-top: 0.8rem;
	}
	.bar {
		display: grid;
		grid-template-columns: 4.5rem minmax(0, 1fr) auto;
		align-items: center;
		gap: 0.4rem;
		font-size: 0.85rem;
	}
	i {
		display: block;
		background: var(--success);
		height: 0.4rem;
		border-radius: 0.25rem;
	}
	.expense i {
		background: var(--danger);
	}
</style>
