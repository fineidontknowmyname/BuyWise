<script lang="ts">
	import '#lib/styles/workbench.css';
	import { enhance } from '$app/forms';
	import { resolve } from '$app/paths';
	import { formatRupees } from '#lib/money.js';
	import type { PageProps } from './$types';
	let { data, form }: PageProps = $props();
</script>

<svelte:head><title>Transfers & reserves | BuyWise</title></svelte:head>
<div class="workbench">
	<header class="wb-heading">
		<div>
			<p class="wb-eyebrow">My Money</p>
			<h1>Transfers & reserves</h1>
			<p class="wb-muted">Move recorded money between accounts and protect emergency savings.</p>
		</div>
		<a href={resolve('/accounts')}>Back to your money plan →</a>
	</header>
	{#if data.error || form?.error}<p class="wb-notice" role="alert">
			{form?.error ?? data.error}
		</p>{/if}
	{#if form?.success}<p class="wb-notice" role="status">{form.success}</p>{/if}
	{#if data.foundation}
		<section class="wb-card">
			<h2>Account purposes</h2>
			<p class="wb-muted">Emergency reserves are excluded from funds available for purchases.</p>
			{#each data.foundation.accounts as account (account.id)}<form
					method="POST"
					action="?/purpose"
					use:enhance
				>
					<input type="hidden" name="accountId" value={account.id} />
					<div class="wb-fields">
						<label
							>{account.name} · {formatRupees(account.balancePaise)}<select
								name="purpose"
								value={account.purpose}
								><option value="everyday">Everyday</option><option value="general_savings"
									>Savings</option
								><option value="emergency_reserve">Emergency reserve</option></select
							></label
						>
					</div>
					<button class="btn btn-secondary">Save purpose</button>
				</form>{/each}
		</section>
		<section class="wb-card">
			<h2>Record an account transfer</h2>
			<form method="POST" action="?/transfer" use:enhance>
				<input
					type="hidden"
					name="idempotencyKey"
					value={form?.idempotencyKey ?? data.idempotencyKey}
				/>
				<div class="wb-fields">
					{#each ['fromAccountId', 'toAccountId'] as name (name)}<label
							>{name === 'fromAccountId' ? 'From' : 'To'}<select {name} required
								><option value="">Choose account</option
								>{#each data.foundation.accounts as account (account.id)}<option value={account.id}
										>{account.name}</option
									>{/each}</select
							></label
						>{/each}<label>Amount (₹)<input name="amount" inputmode="decimal" required /></label
					><label>Date<input name="occurredOn" type="date" value={data.today} required /></label
					><label>Note<input name="note" maxlength="500" /></label>
				</div>
				<button class="btn btn-primary">Record transfer</button>
			</form>
		</section>
		<details class="wb-fold">
			<summary>Transfer history</summary>
			<div class="wb-fold-body">
				{#each data.transfers as entry (entry.id)}<div class="wb-row">
						<span
							>{formatRupees(entry.amountPaise)} · {entry.occurredOn}
							{entry.reversed ? '· Reversed' : ''}</span
						>{#if !entry.reversed && !entry.reversesId && !entry.purchaseManaged}<form
								method="POST"
								action="?/reverse"
								use:enhance
							>
								<input type="hidden" name="transferId" value={entry.id} /><button
									class="btn btn-secondary">Reverse transfer</button
								>
							</form>{/if}
					</div>{:else}<p class="wb-muted">No transfers recorded.</p>{/each}
			</div>
		</details>
	{/if}
</div>
