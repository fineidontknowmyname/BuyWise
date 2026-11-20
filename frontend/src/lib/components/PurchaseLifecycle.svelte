<script lang="ts">
	import { enhance } from '$app/forms';
	import { resolve } from '$app/paths';
	import { formatRupees } from '#lib/money.js';
	import type {
		PurchaseInput,
		PurchaseLifecycle,
		PurchasePreview,
		PurchaseSummary
	} from '#lib/purchase.js';

	let {
		lifecycle,
		goalId,
		goalName,
		today,
		allocationKey,
		active,
		feedback
	}: {
		lifecycle: PurchaseLifecycle | null;
		goalId: string;
		goalName: string;
		today: string;
		allocationKey: string;
		active: boolean;
		feedback: {
			purchaseError?: string;
			purchaseDraft?: PurchaseInput;
			purchasePreview?: PurchasePreview;
			purchaseSaved?: boolean;
			allocationError?: string;
			allocationSaved?: boolean;
			allocationDraft?: { accountId: string; goalId: string; amount: string }[];
		} | null;
	} = $props();

	let extraRows = $state(2);
	const rows = $derived(
		Array.from(
			{ length: Math.max(extraRows, feedback?.allocationDraft?.length ?? 0) },
			(_, index) => index
		)
	);
	const paymentAccounts = $derived(
		lifecycle?.accounts.filter((account) => account.purpose !== 'emergency_reserve') ?? []
	);
	const hasDestinations = $derived(
		!!lifecycle?.goals.length ||
			lifecycle?.accounts.some(
				(account) =>
					account.purpose === 'general_savings' || account.purpose === 'emergency_reserve'
			)
	);
	const dateTime = new Intl.DateTimeFormat('en-IN', {
		dateStyle: 'medium',
		timeStyle: 'short',
		timeZone: 'Asia/Kolkata'
	});
</script>

{#snippet amounts(summary: PurchaseSummary)}
	<dl class="summary">
		<div>
			<dt>Original target</dt>
			<dd>{formatRupees(summary.targetPaise)}</dd>
		</div>
		<div>
			<dt>Total contributed</dt>
			<dd>{formatRupees(summary.contributedPaise)}</dd>
		</div>
		<div>
			<dt>Backed savings</dt>
			<dd>{formatRupees(summary.backedPaise)}</dd>
		</div>
		<div>
			<dt>Actual landed cost</dt>
			<dd>{formatRupees(summary.costPaise)}</dd>
		</div>
		<div>
			<dt>Surplus</dt>
			<dd>{formatRupees(summary.surplusPaise)}</dd>
		</div>
	</dl>
	<p>{summary.date} · {summary.seller} · {summary.paymentAccountName} · {summary.categoryName}</p>
	{#if summary.reference}<p>Receipt or reference {summary.reference}</p>{/if}
	{#if summary.shortfallPaise > 0}
		<p>
			{formatRupees(summary.shortfallPaise)} comes from unallocated cash in {summary.paymentAccountName}.
		</p>
	{/if}
{/snippet}

{#snippet splitRow(index: number, options: PurchaseLifecycle)}
	{@const draft = feedback?.allocationDraft?.[index]}
	<fieldset>
		<legend>Allocation {index + 1}</legend>
		<div class="fields">
			<label>
				Destination account
				<select
					class="input"
					name="accountId"
					value={draft?.accountId ?? ''}
					required={index === 0}
				>
					<option value="">Choose an account</option>
					{#each options.accounts as account (account.id)}
						<option value={account.id}>
							{account.name}{account.purpose === 'emergency_reserve'
								? ' · Emergency reserve'
								: account.purpose === 'general_savings'
									? ' · General savings'
									: ' · Goal funding only'}
						</option>
					{/each}
				</select>
			</label>
			<label>
				Purchase goal
				<select class="input" name="goalId" value={draft?.goalId ?? ''}>
					<option value="">Keep in selected reserve account</option>
					{#each options.goals as goal (goal.id)}
						<option value={goal.id}>{goal.name}</option>
					{/each}
				</select>
			</label>
			<label>
				Amount in ₹
				<input
					class="input"
					name="amount"
					inputmode="decimal"
					value={draft?.amount ?? ''}
					required={index === 0}
				/>
			</label>
		</div>
	</fieldset>
{/snippet}

{#if lifecycle?.purchase}
	{@const purchase = lifecycle.purchase}
	<section class="card lifecycle" aria-labelledby="purchase-complete">
		<header>
			<h2 id="purchase-complete">Purchase completed</h2>
			<p>{goalName}</p>
		</header>
		{#if feedback?.purchaseSaved}<p role="status">Purchase recorded in your ledger.</p>{/if}
		{@render amounts(purchase)}
		{#if purchase.funding.length}
			<details>
				<summary>Purchase funding</summary>
				<ul>
					{#each purchase.funding as source (source.accountId)}
						<li>{source.accountName} · {formatRupees(source.amountPaise)}</li>
					{/each}
				</ul>
			</details>
		{/if}
		{#if purchase.surplusPaise > 0}
			<h3>Surplus allocation</h3>
			<p><strong>{formatRupees(lifecycle.remainingPaise)}</strong> left to allocate.</p>
			{#if feedback?.allocationSaved}<p role="status">Surplus allocation recorded.</p>{/if}
			{#if feedback?.allocationError}<p role="alert">{feedback.allocationError}</p>{/if}
			{#if lifecycle.remainingPaise > 0}
				<p>
					Unallocated surplus stays reserved in its source accounts. Allocate all or part of it
					whenever you’re ready.
				</p>
				{#if hasDestinations}
					<form method="POST" action="?/allocateSurplus" use:enhance>
						<input type="hidden" name="key" value={allocationKey} />
						{@render splitRow(0, lifecycle)}
						<details open={(feedback?.allocationDraft?.length ?? 0) > 1}>
							<summary>Split across destinations</summary>
							{#each rows.slice(1) as index (index)}
								{@render splitRow(index, lifecycle)}
							{/each}
							{#if rows.length < 10}
								<button
									type="button"
									class="btn btn-secondary"
									onclick={() => (extraRows = rows.length + 1)}>Add destination</button
								>
							{/if}
						</details>
						<label class="check"
							><input type="checkbox" name="confirmed" value="yes" required /> Confirm the amounts and
							destinations above.</label
						>
						<button class="btn btn-primary">Allocate surplus</button>
					</form>
				{:else}
					<p>
						<a href={resolve('/finances')}
							>Designate a general savings or emergency reserve account</a
						> to allocate the surplus.
					</p>
				{/if}
			{/if}
			{#if lifecycle.allocations.length}
				<h3>Allocation history</h3>
				{#each lifecycle.allocations as allocation, index (index)}
					<details>
						<summary
							>{dateTime.format(new Date(allocation.recordedAt))} · {formatRupees(
								allocation.lines.reduce((sum, line) => sum + line.amountPaise, 0)
							)}</summary
						>
						<ul>
							{#each allocation.lines as line, lineIndex (lineIndex)}
								<li>
									{formatRupees(line.amountPaise)} to {line.goalName
										? `${line.goalName} (${line.accountName})`
										: line.accountName}
									{#if !line.goalName && line.purpose === 'emergency_reserve'}
										· Emergency reserve{/if}
									<ul>
										{#each line.sources as source (source.accountId)}
											<li>{formatRupees(source.amountPaise)} from {source.accountName}</li>
										{/each}
									</ul>
								</li>
							{/each}
						</ul>
					</details>
				{/each}
			{/if}
		{/if}
	</section>
{:else if active || !lifecycle}
	<section class="card lifecycle" aria-labelledby="purchase-form">
		<h2 id="purchase-form">Mark purchased</h2>
		{#if feedback?.purchaseError}<p role="alert">{feedback.purchaseError}</p>{/if}
		{#if !lifecycle}
			<p>Purchase details are temporarily unavailable.</p>
		{:else if feedback?.purchasePreview}
			{@const preview = feedback.purchasePreview}
			{@const summary = preview.summary}
			<h3>Review purchase</h3>
			{@render amounts(summary)}
			<p>BuyWise will record one expense in {summary.paymentAccountName}.</p>
			{#each summary.funding.filter((source) => source.accountId !== summary.paymentAccountId) as source (source.accountId)}
				<p>
					{formatRupees(source.amountPaise)} of earmarked funds will be recorded as a transfer from {source.accountName}
					to {summary.paymentAccountName}.
				</p>
			{/each}
			{#if summary.surplusPaise > 0}<p>
					Your surplus will stay reserved until you allocate it.
				</p>{/if}
			<form method="POST" action="?/completePurchase" use:enhance>
				<input type="hidden" name="key" value={preview.key} />
				<input type="hidden" name="token" value={preview.token} />
				<input type="hidden" name="date" value={summary.date} />
				<input type="hidden" name="seller" value={summary.seller} />
				<input type="hidden" name="costPaise" value={summary.costPaise} />
				<input type="hidden" name="reference" value={summary.reference} />
				<input type="hidden" name="paymentAccountId" value={summary.paymentAccountId} />
				<input type="hidden" name="categoryId" value={summary.categoryId} />
				<label class="check"
					><input type="checkbox" name="confirmed" value="yes" required /> Confirm the purchase details
					and ledger entries.</label
				>
				<div class="actions">
					<button class="btn btn-primary">Confirm purchase</button>
					<a
						class="btn btn-secondary"
						href={resolve('/goals/[id]', { id: goalId })}
						data-sveltekit-reload>Start over</a
					>
				</div>
			</form>
		{:else if paymentAccounts.length && lifecycle.categories.length}
			{@const draft = feedback?.purchaseDraft}
			<details open={!!draft}>
				<summary>Enter purchase details</summary>
				<form method="POST" action="?/previewPurchase" use:enhance>
					<div class="fields">
						<label
							>Purchase date <input
								class="input"
								type="date"
								name="date"
								required
								max={today}
								value={draft?.date || today}
							/></label
						>
						<label
							>Seller <input
								class="input"
								name="seller"
								required
								maxlength="200"
								value={draft?.seller ?? ''}
							/></label
						>
						<label
							>Landed cost in ₹ <input
								class="input"
								name="cost"
								inputmode="decimal"
								required
								value={draft?.costPaise ? (draft.costPaise / 100).toFixed(2) : ''}
							/></label
						>
						<label
							>Payment account
							<select
								class="input"
								name="paymentAccountId"
								required
								value={draft?.paymentAccountId ?? ''}
							>
								<option value="">Choose an account</option>
								{#each paymentAccounts as account (account.id)}<option value={account.id}
										>{account.name}</option
									>{/each}
							</select>
						</label>
						<label
							>Expense category
							<select class="input" name="categoryId" required value={draft?.categoryId ?? ''}>
								<option value="">Choose a category</option>
								{#each lifecycle.categories as category (category.id)}<option value={category.id}
										>{category.name}</option
									>{/each}
							</select>
						</label>
						<label
							>Receipt or reference (optional) <input
								class="input"
								name="reference"
								maxlength="500"
								value={draft?.reference ?? ''}
							/></label
						>
					</div>
					<p>
						Include discounts, tax and delivery in the final amount paid. Select the account that
						paid the seller.
					</p>
					<button class="btn btn-primary">Review purchase</button>
				</form>
			</details>
		{:else}
			<p>
				Add a <a href={resolve('/accounts')}>payment account</a> and an active
				<a href={resolve('/categories')}>expense category</a> to record a purchase.
			</p>
		{/if}
	</section>
{/if}

<style>
	.lifecycle {
		display: grid;
		gap: 0.9rem;
		padding: clamp(1rem, 3vw, 1.5rem);
		overflow-wrap: anywhere;
	}
	h2,
	h3,
	p,
	dl {
		margin: 0;
	}
	.fields,
	.summary {
		display: grid;
		gap: 0.8rem;
		grid-template-columns: repeat(auto-fit, minmax(min(100%, 13rem), 1fr));
	}
	.summary > div {
		background: var(--surface-secondary);
		padding: 0.75rem;
		border-radius: var(--radius-md);
	}
	.summary dd {
		margin: 0.25rem 0 0;
		font-weight: 700;
	}
	form,
	label {
		display: grid;
		gap: 0.45rem;
	}
	form {
		gap: 0.8rem;
		justify-items: start;
		margin-top: 0.65rem;
	}
	form .fields,
	fieldset {
		width: 100%;
	}
	fieldset {
		margin: 0;
		padding: 0.7rem;
		border: 1px solid var(--border-subtle);
		border-radius: var(--radius-md);
	}
	form details,
	.input {
		width: 100%;
	}
	form details fieldset {
		margin-top: 0.65rem;
	}
	form details button {
		margin-top: 0.65rem;
	}
	.check,
	.actions {
		display: flex;
		align-items: center;
		gap: 0.6rem;
		flex-wrap: wrap;
	}
	summary {
		cursor: pointer;
		font-weight: 600;
	}
	[role='alert'] {
		color: var(--danger);
	}
</style>
