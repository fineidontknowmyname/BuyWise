<script lang="ts">
	import { enhance } from '$app/forms';
	import { resolve } from '$app/paths';
	import { untrack } from 'svelte';
	import {
		Plus,
		Upload,
		ReceiptText,
		Inbox,
		ArrowUpRight,
		AlertTriangle,
		FileText,
		CheckCircle2,
		Search
	} from '@lucide/svelte';
	import { formatRupees } from '#lib/money.js';
	import {
		maxExpenseFileBytes,
		parseExpenseCSV,
		suggestExpenseMapping,
		type ExpenseMapping
	} from '#lib/expenseImport.js';
	import Modal from '#lib/components/Modal.svelte';
	import ExpenseImportReview from '#lib/components/ExpenseImportReview.svelte';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();
	let view = $state(untrack(() => (data.selectedImport ? 'imports' : 'records')));
	let manualOpen = $state(false);
	let manualSaving = $state(false);
	let uploading = $state(false);
	let candidateBusy = $state(false);
	let fileInput = $state<HTMLInputElement>();
	let csvHeaders = $state<string[]>([]);
	let fileName = $state('');
	let fileProblem = $state('');
	let isDocument = $state(false);
	let fileReading = $state(false);
	let fileSelection = 0;
	let mapping = $state<ExpenseMapping>({
		date: '',
		amount: '',
		debit: '',
		credit: '',
		description: '',
		reference: '',
		kind: ''
	});
	let lastImport = $state(untrack(() => data.selectedImport?.id));
	const activeCategories = $derived(
		(data.categories ?? []).filter((category) => !category.archived)
	);
	const readyForEntry = $derived(!!data.accounts?.length && activeCategories.length > 0);
	const pendingRows = $derived(
		data.imports
			?.filter((batch) => !['reversed', 'discarded'].includes(batch.status))
			.reduce((sum, batch) => sum + batch.rows.filter((row) => row.status === 'pending').length, 0)
	);
	const reviewCount = $derived(
		data.candidates === null || pendingRows === undefined
			? null
			: data.candidates.length + pendingRows
	);
	const mappingLabels: Record<keyof ExpenseMapping, string> = {
		date: 'Date',
		amount: 'Amount',
		debit: 'Debit / withdrawal',
		credit: 'Credit / deposit',
		description: 'Description / merchant',
		reference: 'Reference',
		kind: 'Transaction type'
	};
	const mappingKeys = Object.keys(mappingLabels) as (keyof ExpenseMapping)[];

	$effect(() => {
		if (data.selectedImport && data.selectedImport.id !== lastImport) {
			lastImport = data.selectedImport.id;
			view = 'imports';
		}
		if (form?.uploadError || data.selectedImportError) view = 'imports';
	});
	$effect(() => {
		if (form?.manualError) manualOpen = true;
		if (form?.candidateError) view = 'review';
	});

	async function inspectFile() {
		const selection = ++fileSelection;
		csvHeaders = [];
		fileProblem = '';
		fileName = '';
		isDocument = false;
		fileReading = false;
		const file = fileInput?.files?.[0];
		if (!file) return;
		fileName = file.name;
		if (file.size > maxExpenseFileBytes) {
			fileProblem = 'Choose a file of up to 400 KB.';
			return;
		}
		const extension = file.name.split('.').pop()?.toLowerCase();
		if (!['csv', 'pdf', 'png', 'jpg', 'jpeg'].includes(extension ?? '')) {
			fileProblem = 'Use CSV, PDF, JPEG or PNG.';
			return;
		}
		isDocument = extension !== 'csv';
		if (isDocument) return;
		fileReading = true;
		try {
			const parsed = parseExpenseCSV(await file.text());
			if (selection === fileSelection) {
				csvHeaders = parsed.headers;
				mapping = suggestExpenseMapping(parsed.headers);
			}
		} catch (error) {
			if (selection === fileSelection)
				fileProblem = error instanceof Error ? error.message : 'This CSV could not be read.';
		} finally {
			if (selection === fileSelection) fileReading = false;
		}
	}

	function displayDate(value: string) {
		return new Intl.DateTimeFormat('en-IN', {
			day: 'numeric',
			month: 'short',
			year: 'numeric',
			timeZone: 'Asia/Kolkata'
		}).format(new Date(value + 'T00:00:00Z'));
	}
	type Draft = {
		accountId?: string;
		categoryId?: string;
		amount?: string;
		occurredOn?: string;
		note?: string;
	};
</script>

<svelte:head><title>My expenses | BuyWise</title></svelte:head>

{#snippet expenseFields(draft: Draft)}
	<div class="form-fields">
		<label
			>Amount (INR)<input
				class="input"
				name="amount"
				inputmode="decimal"
				placeholder="0.00"
				required
				value={draft.amount ?? ''}
			/></label
		>
		<label
			>Date<input
				class="input"
				type="date"
				name="occurredOn"
				required
				max={data.today}
				value={draft.occurredOn ?? data.today}
			/></label
		>
		<label
			>Paid from<select class="input" name="accountId" required value={draft.accountId ?? ''}
				><option value="">Choose an account</option
				>{#each data.accounts ?? [] as account (account.id)}<option value={account.id}
						>{account.name}</option
					>{/each}</select
			></label
		>
		<label
			>Category<select class="input" name="categoryId" required value={draft.categoryId ?? ''}
				><option value="">Choose a category</option
				>{#each activeCategories as category (category.id)}<option value={category.id}
						>{category.name}</option
					>{/each}</select
			></label
		>
	</div>
	<label
		>Description (optional)<input
			class="input"
			name="note"
			maxlength="500"
			placeholder="What was this expense for?"
			value={draft.note ?? ''}
		/></label
	>
{/snippet}

<div class="expenses-workspace">
	<header class="page-heading">
		<div>
			<p class="eyebrow">My Money</p>
			<h1>My expenses</h1>
			<p class="muted">
				A clear record of what you spent, with every import reviewed before it counts.
			</p>
		</div>
		<div class="heading-actions">
			<button type="button" class="btn btn-secondary" onclick={() => (view = 'imports')}
				><Upload size={17} />Import file</button
			>
			<button type="button" class="btn btn-accent" onclick={() => (manualOpen = true)}
				><Plus size={18} />Add expense</button
			>
		</div>
	</header>
	{#if form?.manualSaved}<p class="notice success" role="status">
			<CheckCircle2 size={18} />Expense recorded. Spending and account records have been updated.
		</p>{/if}
	{#if form?.candidateSaved}<p class="notice success" role="status">
			Reviewed expense recorded.
		</p>{/if}
	{#if form?.candidateRejected}<p class="notice" role="status">Suggestion dismissed.</p>{/if}
	{#if form?.importFinished}<p class="notice success" role="status">
			Import updated. Its current status is shown below.
		</p>{/if}
	<dl class="expense-metrics">
		<div>
			<dt><ReceiptText size={17} />Recorded spending</dt>
			<dd>{data.expenses ? formatRupees(data.expenses.totalPaise) : 'Unavailable'}</dd>
			<p>Matching filters · {data.filters.month}</p>
		</div>
		<div>
			<dt><FileText size={17} />Expense records</dt>
			<dd>{data.expenses ? data.expenses.count : 'Unavailable'}</dd>
			<p>Confirmed, non-voided spending</p>
		</div>
		<div>
			<dt><Inbox size={17} />Awaiting review</dt>
			<dd>{reviewCount ?? 'Unavailable'}</dd>
			<p>Recent imports and suggestions · not counted as spending</p>
		</div>
	</dl>
	<div class="workspace-tabs" role="group" aria-label="Expense workspace views">
		<button
			type="button"
			class:active={view === 'records'}
			aria-pressed={view === 'records'}
			onclick={() => (view = 'records')}>Recorded expenses</button
		>
		<button
			type="button"
			class:active={view === 'review'}
			aria-pressed={view === 'review'}
			onclick={() => (view = 'review')}
			>Review inbox{#if data.candidates?.length}<span>{data.candidates.length}</span>{/if}</button
		>
		<button
			type="button"
			class:active={view === 'imports'}
			aria-pressed={view === 'imports'}
			onclick={() => (view = 'imports')}>File imports</button
		>
	</div>
	{#if view === 'records'}
		<section class="card panel" aria-labelledby="records-title">
			<header>
				<h2 id="records-title">Recorded expenses</h2>
				<p class="muted">Search notes or narrow down a month, account or category.</p>
			</header>
			<form method="GET" action={resolve('/expenses')} class="expense-filters">
				<label
					>Month<input
						class="input"
						type="month"
						name="month"
						value={data.filters.month}
						required
					/></label
				>
				<label
					>Account<select class="input" name="account" value={data.filters.account}
						><option value="">All accounts</option
						>{#each data.accounts ?? [] as account (account.id)}<option value={account.id}
								>{account.name}</option
							>{/each}</select
					></label
				>
				<label
					>Category<select class="input" name="category" value={data.filters.category}
						><option value="">All categories</option
						>{#each data.categories ?? [] as category (category.id)}<option value={category.id}
								>{category.name}{category.archived ? ' (archived)' : ''}</option
							>{/each}</select
					></label
				>
				<label
					>Description<input
						class="input"
						type="search"
						name="search"
						maxlength="200"
						placeholder="Search notes"
						value={data.filters.search}
					/></label
				>
				<button class="btn btn-secondary"><Search size={17} />Apply</button>
			</form>
			{#if data.accounts === null || data.categories === null}<p class="notice" role="alert">
					Account or category labels could not load. Reload before adding expenses.
				</p>{/if}
			{#if !data.expenses}
				<div class="empty-state">
					<AlertTriangle size={28} />
					<h3>Expenses could not be loaded</h3>
					<p>{data.expensesError}</p>
					<a class="btn btn-secondary" href={resolve('/expenses')} data-sveltekit-reload
						>Try again</a
					>
				</div>
			{:else if !data.expenses.items.length}
				<div class="empty-state">
					<ReceiptText size={32} />
					<h3>No recorded expenses in this view</h3>
					<p>
						Add an expense, import a file, or adjust the filters. Planned obligations are managed in
						My Money.
					</p>
					<button type="button" class="btn btn-secondary" onclick={() => (manualOpen = true)}
						>Add your first expense</button
					>
				</div>
			{:else}
				<div class="expense-list">
					{#each data.expenses.items as expense (expense.id)}
						<article class="expense-item">
							<span class="expense-icon"><ReceiptText size={19} aria-hidden="true" /></span>
							<div class="expense-description">
								<h3>
									{expense.note ||
										data.categories?.find((category) => category.id === expense.categoryId)?.name ||
										'Recorded expense'}
								</h3>
								<p>
									{data.categories?.find((category) => category.id === expense.categoryId)?.name ??
										'Category unavailable'} · {data.accounts?.find(
										(account) => account.id === expense.accountId
									)?.name ?? 'Account unavailable'}
								</p>
								<span
									>{displayDate(expense.occurredOn)}{expense.purchaseManaged
										? ' · Linked purchase'
										: ''}</span
								>
							</div>
							<div class="expense-amount">
								<strong>{formatRupees(expense.amountPaise)}</strong><a
									href="{resolve('/accounts/[id]', {
										id: expense.accountId
									})}#transaction-{expense.id}">View record <ArrowUpRight size={14} /></a
								>
							</div>
						</article>
					{/each}
				</div>
				{#if data.expenses.count > data.expenses.items.length}<p class="muted">
						Showing the latest {data.expenses.items.length} of {data.expenses.count} matching records.
						Narrow the filters to find older entries. The total includes every match.
					</p>{/if}
			{/if}
		</section>
	{:else if view === 'review'}
		<section class="card panel" aria-labelledby="inbox-title">
			<header>
				<h2 id="inbox-title">Review inbox</h2>
				<p class="muted">Suggestions stay outside spending totals until confirmed.</p>
			</header>
			{#if form?.candidateError}<p class="notice error" role="alert">{form.candidateError}</p>{/if}
			{#if data.candidates === null}<p class="notice">
					Suggestions could not load. Try this page again.
				</p>
			{:else if !data.candidates.length}<div class="empty-state">
					<Inbox size={32} />
					<h3>You are all caught up</h3>
					<p>
						Expense suggestions appear here when available. Uploaded files have a separate review
						under File imports.
					</p>
					<button type="button" class="btn btn-secondary" onclick={() => (view = 'imports')}
						>Import a file</button
					>
				</div>
			{:else}{#each data.candidates as item (item.id)}
					{@const feedback = form?.candidateId === item.id ? form : null}
					<details class="candidate" open={!!feedback?.candidateError}>
						<summary
							><span>{item.merchant || 'Expense suggestion'}</span><strong
								>{formatRupees(item.amountPaise)}</strong
							></summary
						>
						<p class="muted">Notification suggestion · {item.occurredOn} · {item.sourceApp}</p>
						{#if item.categorySuggestion}<p class="muted">
								Suggested category: {item.categorySuggestion.name}. Confirm it below.
							</p>{/if}
						{#if item.possibleDuplicate || feedback?.knownDuplicate}<p class="notice warning">
								A similar date and amount exists. Check before recording.
							</p>{/if}
						{#if readyForEntry}
							<form
								method="POST"
								action="?/confirm"
								use:enhance={() => {
									candidateBusy = true;
									return async ({ update }) => {
										try {
											await update({ reset: false });
										} finally {
											candidateBusy = false;
										}
									};
								}}
							>
								<input type="hidden" name="candidateId" value={item.id} />
								{@render expenseFields(
									feedback?.candidateDraft ?? {
										amount: String(item.amountPaise / 100),
										occurredOn: item.occurredOn,
										categoryId: item.categorySuggestion?.id,
										note: item.merchant
									}
								)}
								<label class="choice"
									><input type="checkbox" name="allowDuplicate" value="yes" />If it matches another
									record, this is a separate purchase.</label
								>
								<label class="choice"
									><input type="checkbox" name="confirmed" value="yes" required />I reviewed these
									details and confirm this is spending.</label
								>
								<button class="btn btn-accent" disabled={candidateBusy}
									>{candidateBusy ? 'Recording…' : 'Confirm expense'}</button
								>
							</form>
						{:else}<p class="muted">
								Add an account and an active expense category before recording.
							</p>{/if}
						<form method="POST" action="?/reject" use:enhance>
							<input type="hidden" name="candidateId" value={item.id} /><button
								class="btn btn-secondary">Dismiss suggestion</button
							>
						</form>
					</details>
				{/each}{/if}
		</section>
	{/if}
	<div class="import-content" hidden={view !== 'imports'}>
		<div class="import-layout">
			<section class="card panel" aria-labelledby="upload-title">
				<div class="upload-mark"><Upload size={26} /></div>
				<h2 id="upload-title">Bring your expenses in</h2>
				<p class="muted">
					Upload, review, then record. Nothing is added to your spending automatically.
				</p>
				{#if form?.uploadError || fileProblem}<p class="notice error" role="alert">
						{fileProblem || form?.uploadError}
					</p>{/if}
				<form
					method="POST"
					action="?/upload"
					enctype="multipart/form-data"
					use:enhance={() => {
						uploading = true;
						return async ({ update }) => {
							try {
								await update({ reset: false });
							} finally {
								uploading = false;
							}
						};
					}}
				>
					<label class="file-picker"
						>Choose a statement or receipt<input
							bind:this={fileInput}
							type="file"
							name="file"
							accept=".csv,.pdf,.jpg,.jpeg,.png"
							required
							onchange={inspectFile}
						/></label
					>
					<p class="file-help">
						CSV, PDF, JPEG or PNG · up to 400 KB · 200 rows · PDFs up to 3 pages
					</p>
					{#if csvHeaders.length}<fieldset class="mapping-fields">
							<legend>Match your CSV columns</legend>
							{#each mappingKeys as key (key)}<label
									>{mappingLabels[key]}<select
										class="input"
										name={'map_' + key}
										bind:value={mapping[key]}
										><option value="">Not mapped</option
										>{#each csvHeaders as header (header)}<option value={header}>{header}</option
											>{/each}</select
									></label
								>{/each}
						</fieldset>{/if}
					<div class="form-fields">
						<label
							>Date format<select class="input" name="dateOrder"
								><option value="DMY">Day / month / year</option><option value="MDY"
									>Month / day / year</option
								></select
							></label
						>
						<label
							>Single amount column<select class="input" name="signedMode"
								><option value="positive">Positive amounts are expenses</option><option
									value="negative">Negative amounts are expenses</option
								></select
							></label
						>
					</div>
					{#if isDocument}<label class="choice"
							><input type="checkbox" name="documentConsent" value="yes" required />I agree to send
							this document's text or images to the configured AI provider to extract draft rows.</label
						>{/if}
					<p class="file-help">
						CSV is parsed without AI. PDFs and images require consent. Raw files are not retained;
						extracted rows remain in your review history.
					</p>
					<button class="btn btn-accent" disabled={uploading || fileReading || !!fileProblem}
						>{uploading
							? 'Reading your file…'
							: fileReading
								? 'Checking CSV…'
								: 'Extract & review'}</button
					>
					{#if uploading}<p role="status" class="muted">
							Reading the file. Keep this page open; your balances have not changed.
						</p>{/if}
					{#if fileName}<p class="file-help">Selected: {fileName}</p>{/if}
				</form>
			</section>
			<section class="card panel import-history" aria-labelledby="history-title">
				<h2 id="history-title">Recent imports</h2>
				<p class="muted">Reopen a saved review or inspect what was recorded.</p>
				{#if data.imports === null}<p class="notice">
						{data.importsError || 'Import history is unavailable.'}
					</p>
				{:else if !data.imports.length}<div class="history-empty">
						<FileText size={24} />
						<p>Your first file review will appear here.</p>
					</div>
				{:else}<ul>
						{#each data.imports as batch (batch.id)}<li>
								<a href="{resolve('/expenses')}?import={encodeURIComponent(batch.id)}#import-review"
									><strong>{batch.filename}</strong><span
										>{batch.status} · {batch.rows.length} rows</span
									></a
								>
							</li>{/each}
					</ul>{/if}
			</section>
		</div>
		{#if data.selectedImportError}<p class="notice error" role="alert">
				{data.selectedImportError}
			</p>{/if}
		{#if data.selectedImport}{#key data.selectedImport.id + ':' + data.selectedImport.revision}<ExpenseImportReview
					batch={data.selectedImport}
					accounts={data.accounts ?? []}
					categories={activeCategories}
					today={data.today}
					error={form?.importError}
					saved={form?.importSaved}
				/>{/key}{/if}
	</div>
</div>

<Modal
	isOpen={manualOpen}
	title="Add an expense"
	onclose={() => {
		if (!manualSaving) manualOpen = false;
	}}
>
	<p class="muted">
		Record money already spent. Planned bills and saving commitments belong in My Money.
	</p>
	{#if form?.manualError}<p class="notice error" role="alert">{form.manualError}</p>{/if}
	{#if data.accounts === null || data.categories === null}<p class="notice">
			Account or category information is unavailable. Reload before recording.
		</p>
	{:else if !readyForEntry}<p class="notice">
			Create an <a href={resolve('/accounts')}>account</a> and an active
			<a href={resolve('/budget')}>expense category</a> to get started.
		</p>
	{:else}
		<form
			method="POST"
			action="?/manual"
			use:enhance={() => {
				manualSaving = true;
				return async ({ update, result }) => {
					try {
						await update({ reset: result.type === 'success' });
						if (result.type === 'success') manualOpen = false;
					} finally {
						manualSaving = false;
					}
				};
			}}
		>
			<input
				type="hidden"
				name="key"
				value={form?.requestKey ?? data.requestKey}
			/>{@render expenseFields(form?.manualDraft ?? {})}
			<div class="modal-actions">
				<button
					type="button"
					class="btn btn-secondary"
					disabled={manualSaving}
					onclick={() => (manualOpen = false)}>Cancel</button
				><button class="btn btn-accent" disabled={manualSaving}
					>{manualSaving ? 'Recording…' : 'Record expense'}</button
				>
			</div>
		</form>
	{/if}
</Modal>

<style>
	.expenses-workspace {
		display: grid;
		gap: var(--space-6);
		min-width: 0;
	}
	.page-heading,
	.heading-actions,
	.modal-actions {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: var(--space-3);
		flex-wrap: wrap;
	}
	.page-heading > div:first-child {
		flex: 1 1 300px;
	}
	.page-heading h1 {
		margin-block: var(--space-1) var(--space-2);
	}
	.eyebrow,
	.muted {
		color: var(--foreground-secondary);
	}
	.eyebrow {
		font-size: var(--text-small);
		font-weight: 600;
	}
	.expense-metrics {
		display: grid;
		grid-template-columns: repeat(3, minmax(0, 1fr));
		gap: var(--space-4);
	}
	.expense-metrics > div {
		border: 1px solid var(--border);
		border-radius: var(--radius-card);
		background: var(--surface);
		padding: var(--space-6);
		min-width: 0;
	}
	.expense-metrics dt {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		color: var(--foreground-secondary);
		font-size: var(--text-small);
	}
	.expense-metrics dd {
		margin-block: var(--space-3) var(--space-2);
		font: 600 clamp(1.25rem, 2.5vw, 2rem) var(--font-mono);
		overflow-wrap: anywhere;
	}
	.expense-metrics p,
	.file-help {
		font-size: var(--text-caption);
		color: var(--foreground-secondary);
	}
	.workspace-tabs {
		display: flex;
		gap: var(--space-2);
		flex-wrap: wrap;
		border-bottom: 1px solid var(--border);
		padding-bottom: var(--space-4);
	}
	.workspace-tabs button {
		min-height: var(--control-height);
		padding: var(--space-2) var(--space-4);
		border-radius: var(--radius-control);
		color: var(--foreground-secondary);
		font-size: var(--text-small);
	}
	.workspace-tabs button.active {
		background: var(--primary);
		color: var(--primary-foreground);
		font-weight: 600;
	}
	.workspace-tabs span {
		margin-inline-start: var(--space-2);
	}
	.panel {
		display: grid;
		gap: var(--space-4);
		padding: var(--space-6);
		min-width: 0;
		align-content: start;
	}
	h2 {
		font-size: 1.25rem;
	}
	header p.muted {
		margin-top: var(--space-1);
		font-size: var(--text-small);
	}
	form,
	label {
		display: grid;
		gap: var(--space-2);
		min-width: 0;
	}
	form {
		gap: var(--space-4);
	}
	label {
		font-size: var(--text-small);
	}
	.expense-filters {
		grid-template-columns: repeat(4, minmax(0, 1fr)) auto;
		align-items: end;
		padding-block: var(--space-3) var(--space-4);
		border-bottom: 1px solid var(--border);
		gap: var(--space-3);
	}
	.form-fields,
	.mapping-fields {
		display: grid;
		grid-template-columns: repeat(2, minmax(0, 1fr));
		gap: var(--space-4);
	}
	.expense-item {
		display: flex;
		align-items: center;
		gap: var(--space-4);
		padding-block: var(--space-4);
		min-width: 0;
	}
	.expense-item + .expense-item {
		border-top: 1px solid var(--border-subtle);
	}
	.expense-icon,
	.upload-mark {
		display: grid;
		place-items: center;
		flex: none;
		width: 44px;
		height: 44px;
		background: var(--surface-secondary);
		border-radius: var(--radius-control);
	}
	.expense-description {
		flex: 1;
		min-width: 0;
		overflow-wrap: anywhere;
	}
	.expense-description h3 {
		font-size: var(--text-small);
		font-weight: 600;
	}
	.expense-description p {
		color: var(--foreground-secondary);
		font-size: var(--text-small);
		margin-block: var(--space-1);
	}
	.expense-description > span {
		color: var(--foreground-muted);
		font-size: var(--text-caption);
	}
	.expense-amount {
		display: grid;
		gap: var(--space-2);
		justify-items: end;
	}
	.expense-amount strong {
		font-family: var(--font-mono);
		font-variant-numeric: tabular-nums;
		overflow-wrap: anywhere;
	}
	.expense-amount a {
		display: flex;
		align-items: center;
		gap: var(--space-1);
		color: var(--foreground-secondary);
		font-size: var(--text-caption);
		min-height: 32px;
	}
	.empty-state {
		display: grid;
		gap: var(--space-3);
		justify-items: center;
		text-align: center;
		padding: var(--space-12) var(--space-4);
		color: var(--foreground-secondary);
	}
	.empty-state p {
		max-width: 480px;
		font-size: var(--text-small);
	}
	.empty-state h3 {
		font-size: 1.125rem;
	}
	.notice {
		padding: var(--space-3) var(--space-4);
		background: var(--surface-secondary);
		border-radius: var(--radius-control);
		display: flex;
		align-items: start;
		gap: var(--space-2);
		overflow-wrap: anywhere;
	}
	.notice a {
		text-decoration: underline;
	}
	.notice.success {
		background: var(--success-surface);
		color: var(--success-foreground);
	}
	.notice.error {
		background: var(--danger-surface);
		color: var(--danger-foreground);
	}
	.notice.warning {
		background: var(--warning-surface);
		color: var(--warning-foreground);
	}
	.notice :global(svg) {
		flex: none;
	}
	.import-content {
		display: grid;
		gap: var(--space-6);
	}
	.import-content[hidden] {
		display: none;
	}
	.import-layout {
		display: grid;
		grid-template-columns: minmax(0, 1.6fr) minmax(0, 1fr);
		gap: var(--space-6);
		align-items: start;
	}
	.upload-mark {
		background: var(--accent-subtle);
		color: var(--foreground);
	}
	.file-picker {
		padding: var(--space-4);
		border: 1px dashed var(--foreground-muted);
		border-radius: var(--radius-control);
		background: var(--surface-muted);
	}
	.file-picker input {
		max-width: 100%;
		min-width: 0;
		font-size: var(--text-small);
	}
	.file-picker input::file-selector-button {
		min-height: var(--control-height);
		border: 1px solid var(--border);
		border-radius: var(--radius-control);
		background: var(--surface);
		color: var(--foreground);
		padding: var(--space-2) var(--space-3);
		margin-inline-end: var(--space-2);
		cursor: pointer;
	}
	.mapping-fields {
		border: 1px solid var(--border);
		border-radius: var(--radius-control);
		padding: var(--space-4);
		min-inline-size: 0;
	}
	.mapping-fields legend {
		padding-inline: var(--space-2);
		font-size: var(--text-small);
		font-weight: 600;
	}
	.choice {
		display: flex;
		align-items: center;
		gap: var(--space-3);
		min-height: var(--control-height);
	}
	.choice input {
		flex: none;
		width: 20px;
		height: 20px;
		accent-color: var(--brand-dot);
	}
	.import-history ul {
		list-style: none;
	}
	.import-history li + li {
		border-top: 1px solid var(--border-subtle);
	}
	.import-history a {
		display: grid;
		gap: var(--space-1);
		padding-block: var(--space-4);
		overflow-wrap: anywhere;
	}
	.import-history strong {
		font-size: var(--text-small);
	}
	.import-history span {
		color: var(--foreground-secondary);
		font-size: var(--text-caption);
		text-transform: capitalize;
	}
	.history-empty {
		display: grid;
		gap: var(--space-3);
		padding-block: var(--space-6);
		color: var(--foreground-secondary);
	}
	.candidate {
		padding: var(--space-4);
		border: 1px solid var(--border);
		border-radius: var(--radius-control);
	}
	.candidate summary {
		min-height: var(--control-height);
		cursor: pointer;
		overflow-wrap: anywhere;
	}
	.candidate summary strong {
		margin-inline-start: var(--space-3);
		font-family: var(--font-mono);
	}
	.candidate > p,
	.candidate form {
		margin-top: var(--space-4);
	}
	.modal-actions {
		justify-content: flex-end;
	}
	@media (max-width: 1100px) {
		.expense-filters {
			grid-template-columns: repeat(2, minmax(0, 1fr));
		}
	}
	@media (max-width: 800px) {
		.import-layout {
			grid-template-columns: minmax(0, 1fr);
		}
		.expense-metrics {
			grid-template-columns: minmax(0, 1fr);
		}
		.expense-metrics > div {
			padding: var(--space-4);
		}
	}
	@media (max-width: 540px) {
		.expense-filters,
		.form-fields,
		.mapping-fields {
			grid-template-columns: minmax(0, 1fr);
		}
		.panel {
			padding: var(--space-4);
		}
		.expense-icon {
			display: none;
		}
		.expense-item {
			align-items: start;
			gap: var(--space-3);
		}
		.expense-amount {
			max-width: 45%;
		}
		.workspace-tabs button {
			padding-inline: var(--space-3);
		}
	}
</style>
