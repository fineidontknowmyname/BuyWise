<script lang="ts">
	import { enhance } from '$app/forms';
	import { resolve } from '$app/paths';
	import { untrack } from 'svelte';
	import { AlertTriangle, CheckCircle2, FileCheck2 } from '@lucide/svelte';
	import { formatRupees, parseRupees } from '#lib/money.js';
	import type { ExpenseImport } from '#lib/expenseImport.js';
	import type { Account, Category } from '#lib/server/api.js';

	let {
		batch,
		accounts,
		categories,
		today,
		error,
		saved = false
	}: {
		batch: ExpenseImport;
		accounts: Account[];
		categories: Category[];
		today: string;
		error?: string;
		saved?: boolean;
	} = $props();
	const drafts = $state(
		untrack(() =>
			batch.rows.map((row) => ({
				index: row.index,
				selected:
					row.status === 'pending' &&
					row.kind === 'expense' &&
					!!row.amountPaise &&
					!!row.occurredOn &&
					!row.possibleDuplicate,
				amount: row.amountPaise === null ? '' : String(row.amountPaise / 100),
				date: row.occurredOn,
				note: row.description,
				account: accounts.length === 1 ? accounts[0].id : '',
				category: '',
				duplicate: false
			}))
		)
	);
	let posting = $state(false);
	let finishing = $state(false);
	let bulkAccount = $state('');
	let bulkCategory = $state('');
	const closed = $derived(['reversed', 'discarded'].includes(batch.status));
	const posted = $derived(batch.rows.filter((row) => row.transactionId).length);
	const selected = $derived(
		drafts.filter((row) => row.selected && batch.rows[row.index]?.status === 'pending')
	);
	const selectedTotal = $derived(
		selected.reduce((total, row) => total + (parseRupees(row.amount) ?? 0), 0)
	);
	const validSelection = $derived(
		selected.length > 0 &&
			selected.every(
				(row) =>
					row.account &&
					row.category &&
					(parseRupees(row.amount) ?? 0) > 0 &&
					row.date &&
					row.date <= today
			)
	);

	function applyDefaults() {
		for (const row of selected) {
			if (bulkAccount) row.account = bulkAccount;
			if (bulkCategory) row.category = bulkCategory;
		}
	}
</script>

<section class="card import-review" id="import-review" aria-labelledby="review-title">
	<header class="review-header">
		<div>
			<p class="eyebrow">Review before recording</p>
			<h2 id="review-title">{batch.filename}</h2>
		</div>
		<span class="badge">{batch.status} · {batch.rows.length} rows</span>
	</header>
	<p class="muted">
		Only selected rows will become expenses. Leave income, refunds, transfers and statement balances
		out. Correct uncertain fields against your original file.
	</p>
	{#if error}<p class="notice error" role="alert">{error}</p>{/if}
	{#if saved}<p class="notice success" role="status">
			<CheckCircle2 size={18} />Selected expenses recorded. Their records are linked below.
		</p>{/if}
	{#if closed}
		<p class="notice" role="status">
			This import is {batch.status}. Its history is retained, and it cannot post more expenses.
		</p>
	{:else if !accounts.length || !categories.length}
		<p class="notice">
			Create an <a href={resolve('/accounts')}>account</a> and an active
			<a href={resolve('/budget')}>expense category</a> before recording. Your extracted rows are saved
			here.
		</p>
	{/if}
	<form
		method="POST"
		action={'?import=' + encodeURIComponent(batch.id) + '&/recordImport'}
		aria-busy={posting}
		use:enhance={() => {
			posting = true;
			return async ({ update }) => {
				try {
					await update({ reset: false });
				} finally {
					posting = false;
				}
			};
		}}
	>
		<input type="hidden" name="importId" value={batch.id} />
		<input type="hidden" name="revision" value={batch.revision} />
		{#if !closed && batch.rows.some((row) => row.status === 'pending')}
			<div class="bulk-fields">
				<label
					>Account for selected rows<select class="input" bind:value={bulkAccount}
						><option value="">Keep individual choices</option
						>{#each accounts as account (account.id)}<option value={account.id}
								>{account.name}</option
							>{/each}</select
					></label
				>
				<label
					>Category for selected rows<select class="input" bind:value={bulkCategory}
						><option value="">Keep individual choices</option
						>{#each categories as category (category.id)}<option value={category.id}
								>{category.name}</option
							>{/each}</select
					></label
				>
				<button
					type="button"
					class="btn btn-secondary"
					onclick={applyDefaults}
					disabled={!selected.length || (!bulkAccount && !bulkCategory)}>Apply to selected</button
				>
			</div>
		{/if}
		<div class="review-rows">
			{#each batch.rows as row (row.index)}
				{@const draft = drafts[row.index]}
				{@const editable = row.status === 'pending' && !closed}
				<article class="review-row" class:recorded={!!row.transactionId}>
					<div class="row-heading">
						<label class="selection"
							><input
								type="checkbox"
								name="selected"
								value={row.index}
								bind:checked={draft.selected}
								disabled={!editable || posting || !accounts.length || !categories.length}
							/><span>Row {row.index + 1} · {row.description || 'Description needs review'}</span
							></label
						>
						<span class="row-kind">{row.status !== 'pending' ? row.status : row.kind}</span>
					</div>
					{#if editable}
						{#if row.possibleDuplicate}<p class="row-warning">
								<AlertTriangle size={16} />Possible match by date and amount. Check whether this is
								a separate purchase.
							</p>{/if}
						{#if row.warnings.length}<ul class="row-warnings">
								{#each row.warnings as warning, index (index)}<li>{warning}</li>{/each}
							</ul>{/if}
						<fieldset disabled={posting || !draft.selected}>
							<legend class="sr-only">Review row {row.index + 1}</legend>
							<div class="row-fields">
								<label
									>Amount (INR)<input
										class="input"
										name={'amount_' + row.index}
										inputmode="decimal"
										bind:value={draft.amount}
										required={draft.selected}
									/></label
								>
								<label
									>Date<input
										class="input"
										name={'date_' + row.index}
										type="date"
										max={today}
										bind:value={draft.date}
										required={draft.selected}
									/></label
								>
								<label
									>Paid from<select
										class="input"
										name={'account_' + row.index}
										bind:value={draft.account}
										required={draft.selected}
										><option value="">Choose account</option
										>{#each accounts as account (account.id)}<option value={account.id}
												>{account.name}</option
											>{/each}</select
									></label
								>
								<label
									>Category<select
										class="input"
										name={'category_' + row.index}
										bind:value={draft.category}
										required={draft.selected}
										><option value="">Choose category</option
										>{#each categories as category (category.id)}<option value={category.id}
												>{category.name}</option
											>{/each}</select
									></label
								>
							</div>
							<label
								>Description<input
									class="input"
									name={'note_' + row.index}
									maxlength="500"
									bind:value={draft.note}
								/></label
							>
							{#if row.reference}<p class="muted reference">Reference: {row.reference}</p>{/if}
							<label class="choice"
								><input
									type="checkbox"
									name={'duplicate_' + row.index}
									value="yes"
									bind:checked={draft.duplicate}
								/>If this matches another record, I confirm it is a separate purchase.</label
							>
						</fieldset>
					{:else}
						<div class="recorded-summary">
							<FileCheck2 size={18} /><span
								>{row.amountPaise === null ? 'Amount unavailable' : formatRupees(row.amountPaise)} · {row.occurredOn ||
									'Date unavailable'}</span
							>{#if row.transactionId && row.accountId}
								<a
									href="{resolve('/accounts/[id]', {
										id: row.accountId
									})}#transaction-{row.transactionId}">View expense record</a
								>
							{:else if row.transactionId}<span>Linked expense retained</span>{/if}
						</div>
					{/if}
				</article>
			{/each}
		</div>
		{#if !closed && batch.rows.some((row) => row.status === 'pending')}
			<div class="review-footer">
				<p>
					<strong>{selected.length} selected · {formatRupees(selectedTotal)}</strong><span
						class="muted">Account balances and expense totals change only after confirmation.</span
					>
				</p>
				<label class="choice"
					><input type="checkbox" name="reviewed" value="yes" required />I checked the selected rows
					and confirm they represent spending.</label
				>
				<button class="btn btn-accent" disabled={posting || !validSelection}
					>{posting ? 'Recording expenses…' : 'Confirm selected expenses'}</button
				>
			</div>
		{/if}
	</form>
	{#if !closed}
		<details class="finish-import">
			<summary>{posted ? 'Reverse this import' : 'Discard this import'}</summary>
			<p class="muted">
				{posted
					? 'Reversing voids the linked expenses, updates balances and spending totals, and closes unrecorded rows. History stays available.'
					: 'Discarding closes this review without recording any expenses.'}
			</p>
			<form
				method="POST"
				action={'?import=' + encodeURIComponent(batch.id) + '&/finishImport'}
				use:enhance={() => {
					finishing = true;
					return async ({ update }) => {
						try {
							await update({ reset: false });
						} finally {
							finishing = false;
						}
					};
				}}
			>
				<input type="hidden" name="importId" value={batch.id} /><input
					type="hidden"
					name="operation"
					value={posted ? 'reverse' : 'discard'}
				/>
				{#if posted}<label
						>Reason<input class="input" name="reason" required maxlength="450" /></label
					>{/if}
				<label class="choice"
					><input type="checkbox" name="confirmed" value="yes" required />I understand the changes
					described above.</label
				>
				<button class="btn btn-secondary" disabled={finishing}
					>{finishing ? 'Updating import…' : posted ? 'Confirm reversal' : 'Discard review'}</button
				>
			</form>
		</details>
	{/if}
</section>

<style>
	.import-review,
	form,
	fieldset,
	.review-rows {
		display: grid;
		gap: var(--space-4);
		min-width: 0;
	}
	.import-review {
		padding: var(--space-6);
		scroll-margin-top: 96px;
	}
	.review-header,
	.row-heading,
	.recorded-summary {
		display: flex;
		justify-content: space-between;
		align-items: center;
		gap: var(--space-3);
		flex-wrap: wrap;
	}
	h2,
	.row-heading {
		overflow-wrap: anywhere;
	}
	.eyebrow,
	.muted,
	.row-kind,
	.row-warnings {
		color: var(--foreground-secondary);
		font-size: var(--text-small);
	}
	.eyebrow {
		margin-bottom: var(--space-1);
	}
	.badge {
		padding: var(--space-2) var(--space-3);
		border-radius: 999px;
		background: var(--surface-secondary);
		font-size: var(--text-caption);
		text-transform: capitalize;
	}
	.bulk-fields,
	.row-fields {
		display: grid;
		grid-template-columns: repeat(4, minmax(0, 1fr));
		gap: var(--space-3);
		align-items: end;
	}
	.bulk-fields {
		grid-template-columns: repeat(2, minmax(0, 1fr)) auto;
		padding: var(--space-4);
		background: var(--surface-secondary);
		border-radius: var(--radius-control);
	}
	label {
		display: grid;
		gap: var(--space-2);
		font-size: var(--text-small);
		min-width: 0;
	}
	.choice,
	.selection {
		display: flex;
		align-items: center;
		gap: var(--space-3);
	}
	.choice {
		min-height: var(--control-height);
	}
	input[type='checkbox'] {
		width: 20px;
		height: 20px;
		flex: none;
		accent-color: var(--brand-dot);
	}
	.selection {
		font-weight: 600;
		min-height: var(--control-height);
		flex: 1;
	}
	.row-kind {
		text-transform: capitalize;
	}
	.review-row {
		display: grid;
		gap: var(--space-3);
		border: 1px solid var(--border);
		padding: var(--space-4);
		border-radius: var(--radius-control);
	}
	.review-row.recorded {
		background: var(--surface-muted);
	}
	fieldset {
		border: 0;
	}
	.row-warning {
		display: flex;
		gap: var(--space-2);
		color: var(--warning-foreground);
		font-size: var(--text-small);
	}
	.row-warning :global(svg) {
		flex: none;
	}
	.row-warnings {
		padding-inline-start: var(--space-4);
	}
	.reference {
		overflow-wrap: anywhere;
	}
	.review-footer {
		padding: var(--space-4);
		display: grid;
		gap: var(--space-3);
		background: var(--surface-secondary);
		border-radius: var(--radius-control);
	}
	.review-footer p span {
		display: block;
		margin-top: var(--space-1);
	}
	.review-footer .btn {
		justify-self: start;
	}
	.notice {
		display: flex;
		align-items: start;
		gap: var(--space-2);
		padding: var(--space-3);
		border-radius: var(--radius-control);
		background: var(--surface-secondary);
	}
	.notice.error {
		background: var(--danger-surface);
		color: var(--danger-foreground);
	}
	.notice.success {
		background: var(--success-surface);
		color: var(--success-foreground);
	}
	.finish-import {
		border-top: 1px solid var(--border);
		padding-top: var(--space-4);
	}
	.finish-import summary {
		min-height: var(--control-height);
		cursor: pointer;
	}
	.finish-import p {
		margin-bottom: var(--space-3);
	}
	.sr-only {
		position: absolute;
		width: 1px;
		height: 1px;
		overflow: hidden;
		clip-path: inset(50%);
	}
	@media (max-width: 900px) {
		.row-fields,
		.bulk-fields {
			grid-template-columns: repeat(2, minmax(0, 1fr));
		}
	}
	@media (max-width: 540px) {
		.import-review {
			padding: var(--space-4);
		}
		.row-fields,
		.bulk-fields {
			grid-template-columns: minmax(0, 1fr);
		}
	}
</style>
