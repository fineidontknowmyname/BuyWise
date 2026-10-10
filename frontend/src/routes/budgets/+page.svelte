<script lang="ts">
	import { resolve } from '$app/paths';
	import { formatRupees } from '#lib/money.js';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();

	const lineFor = $derived(new Map((data.report?.lines ?? []).map((l) => [l.categoryId, l])));
</script>

<svelte:head>
	<title>Budgets · Self-EMI</title>
</svelte:head>

<main>
	<p><a href={resolve('/')}>Home</a></p>
	<h1>Monthly budgets</h1>
	<p>
		Each limit repeats every month until you change it. Committed expenses such as rent are fixed
		costs and are not budgeted.
	</p>

	{#if form?.error}
		<p role="alert">{form.error}</p>
	{/if}

	{#if data.error}
		<p role="alert">{data.error}</p>
	{:else if data.report}
		<h2>{data.report.month}</h2>
		<table>
			<thead>
				<tr><th>Category</th><th>Limit</th><th>Spent</th><th>Left</th><th>New limit (₹)</th></tr>
			</thead>
			<tbody>
				{#each data.categories as category (category.id)}
					{@const line = lineFor.get(category.id)}
					<tr data-testid="budget-row">
						<td>{category.name}</td>
						<td>{line ? formatRupees(line.limitPaise) : 'Not set'}</td>
						<td>{line ? formatRupees(line.spentPaise) : ''}</td>
						<td class:over={line && line.remainingPaise < 0}>
							{line ? formatRupees(line.remainingPaise) : ''}
						</td>
						<td>
							<form method="POST" action="?/set">
								<input type="hidden" name="categoryId" value={category.id} />
								<input
									name="limit"
									inputmode="decimal"
									required
									aria-label="New limit for {category.name}"
								/>
								<button type="submit">Save</button>
							</form>
						</td>
					</tr>
				{/each}
			</tbody>
		</table>

		<h2>Planned discretionary capacity</h2>
		<ul>
			<li>
				Total monthly limits: <strong data-testid="total-limit"
					>{formatRupees(data.report.totalLimitPaise)}</strong
				>
			</li>
			<li>
				Spent so far this month: <strong data-testid="total-spent"
					>{formatRupees(data.report.totalSpentPaise)}</strong
				>
			</li>
			<li>
				Left to spend: <strong data-testid="total-left"
					>{formatRupees(data.report.totalRemainingPaise)}</strong
				>
			</li>
			<li>
				Discretionary spending with no budget set: <strong data-testid="unbudgeted"
					>{formatRupees(data.report.unbudgetedDiscretionaryPaise)}</strong
				>
			</li>
		</ul>
	{/if}
</main>

<style>
	.over {
		color: #b91c1c;
		font-weight: 600;
	}
</style>
