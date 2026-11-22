<script lang="ts">
	import { resolve } from '$app/paths';
	import Modal from '#lib/components/Modal.svelte';
	import { toast } from '#lib/stores/toast.svelte.js';
	import { todayInIndia } from '#lib/dates.js';
	import { parseRupees } from '#lib/money.js';
	import type { Account, Category } from '#lib/server/api.js';
	import { Sparkles, CheckCircle2 } from '@lucide/svelte';

	let {
		isOpen = false,
		onclose,
		accounts = [],
		categories = [],
		onsuccess
	}: {
		isOpen?: boolean;
		onclose?: () => void;
		accounts?: Account[];
		categories?: Category[];
		onsuccess?: () => void;
	} = $props();

	let promptText = $state('');
	let isParsing = $state(false);
	let isSubmitting = $state(false);
	let parseError = $state('');
	let submissionKey = '';
	const expenseCategories = $derived(
		categories.filter((category) => category.kind === 'expense' && !category.archived)
	);

	// Structured form fields
	let selectedAccountId = $state('');
	let selectedCategoryId = $state('');
	let amountRupees = $state('');
	let occurredOn = $state(todayInIndia());
	let note = $state('');

	// When accounts or categories change, initialize defaults
	$effect(() => {
		if (accounts.length > 0 && !selectedAccountId) {
			selectedAccountId = accounts[0].id;
		}
		if (!expenseCategories.some((category) => category.id === selectedCategoryId)) {
			selectedCategoryId = expenseCategories[0]?.id ?? '';
		}
	});

	async function parsePrompt() {
		if (!promptText.trim()) return;
		isParsing = true;
		parseError = '';

		try {
			const res = await fetch(resolve('/api/ai/parse-expense'), {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ text: promptText.trim() })
			});

			if (!res.ok) {
				const err = await res.json().catch(() => ({}));
				throw new Error(err.error || 'Failed to parse expense description.');
			}

			const data = await res.json();
			if (data.amountPaise) {
				amountRupees = (data.amountPaise / 100).toString();
			}
			if (data.categoryId) {
				selectedCategoryId = data.categoryId;
			}
			if (data.occurredOn) {
				occurredOn = data.occurredOn;
			}
			if (data.note) {
				note = data.note;
			}
		} catch (err: unknown) {
			parseError = err instanceof Error ? err.message : 'Could not parse with AI.';
		} finally {
			isParsing = false;
		}
	}

	async function handleSubmit(e: SubmitEvent) {
		e.preventDefault();
		if (isSubmitting || isParsing) return;
		const amountPaise = parseRupees(amountRupees);
		if (!amountPaise) {
			toast.error('Please enter a valid positive amount.');
			return;
		}
		if (!selectedAccountId) {
			toast.error('Please select an account.');
			return;
		}
		if (!selectedCategoryId) {
			toast.error('Please select a category.');
			return;
		}

		isSubmitting = true;
		submissionKey ||= crypto.randomUUID();
		try {
			const res = await fetch(resolve('/api/transactions/quick'), {
				method: 'POST',
				headers: { 'Content-Type': 'application/json', 'Idempotency-Key': submissionKey },
				body: JSON.stringify({
					accountId: selectedAccountId,
					categoryId: selectedCategoryId,
					amountPaise,
					occurredOn,
					note
				})
			});

			if (!res.ok) {
				const err = await res.json().catch(() => ({}));
				if (err.code === 'idempotency_conflict') submissionKey = '';
				throw new Error(err.error || 'Failed to record expense.');
			}

			toast.success('Expense recorded successfully!');
			// Reset form
			promptText = '';
			amountRupees = '';
			submissionKey = '';
			note = '';
			if (onsuccess) onsuccess();
			if (onclose) onclose();
		} catch (err: unknown) {
			toast.error(err instanceof Error ? err.message : 'Error creating transaction.');
		} finally {
			isSubmitting = false;
		}
	}
</script>

<Modal {isOpen} title="AI Quick-Add Expense" {onclose}>
	<div class="quick-expense-container">
		<!-- Natural Language Prompt Box -->
		<div class="nl-box">
			<label for="nl-input" class="nl-label">
				<Sparkles size={16} class="sparkle-icon" />
				<span>Type naturally (e.g. "Dinner at Olive Garden ₹850 yesterday")</span>
			</label>
			<div class="nl-input-row">
				<input
					id="nl-input"
					type="text"
					placeholder="Spent 350 on petrol this morning..."
					bind:value={promptText}
					onkeydown={(e) => e.key === 'Enter' && parsePrompt()}
					class="form-input"
					disabled={isParsing}
				/>
				<button
					type="button"
					class="btn btn-primary parse-btn"
					onclick={parsePrompt}
					disabled={isParsing || !promptText.trim()}
				>
					{#if isParsing}
						<span class="spinner-small"></span>
					{:else}
						<Sparkles size={15} />
						<span>AI Parse</span>
					{/if}
				</button>
			</div>
			{#if parseError}
				<span class="error-text">{parseError}</span>
			{/if}
		</div>

		<!-- Review & Submit Form -->
		<form onsubmit={handleSubmit} class="expense-form">
			<div class="form-grid-2">
				<div class="form-group">
					<label for="exp-amount" class="form-label">Amount (₹) *</label>
					<input
						id="exp-amount"
						inputmode="decimal"
						required
						placeholder="e.g. 450"
						bind:value={amountRupees}
						class="form-input font-bold"
					/>
				</div>

				<div class="form-group">
					<label for="exp-date" class="form-label">Date *</label>
					<input
						id="exp-date"
						type="date"
						max={todayInIndia()}
						required
						bind:value={occurredOn}
						class="form-input"
					/>
				</div>
			</div>

			<div class="form-grid-2">
				<div class="form-group">
					<label for="exp-category" class="form-label">Category *</label>
					<select id="exp-category" bind:value={selectedCategoryId} class="form-input" required>
						{#each expenseCategories as cat (cat.id)}
							<option value={cat.id}>{cat.name}</option>
						{/each}
					</select>
				</div>

				<div class="form-group">
					<label for="exp-account" class="form-label">Paid From Account *</label>
					<select id="exp-account" bind:value={selectedAccountId} class="form-input" required>
						{#each accounts as acc (acc.id)}
							<option value={acc.id}>{acc.name}</option>
						{/each}
					</select>
				</div>
			</div>

			<div class="form-group">
				<label for="exp-note" class="form-label">Description / Note</label>
				<input
					id="exp-note"
					type="text"
					placeholder="Optional note"
					maxlength="500"
					bind:value={note}
					class="form-input"
				/>
			</div>

			<div class="modal-footer">
				{#if onclose}
					<button type="button" class="btn btn-secondary" onclick={onclose}> Cancel </button>
				{/if}
				<button type="submit" class="btn btn-primary" disabled={isSubmitting || isParsing}>
					{#if isSubmitting}
						<span class="spinner-small"></span>
						<span>Recording...</span>
					{:else}
						<CheckCircle2 size={16} />
						<span>Record Expense</span>
					{/if}
				</button>
			</div>
		</form>
	</div>
</Modal>

<style>
	.quick-expense-container {
		display: flex;
		flex-direction: column;
		gap: 16px;
	}

	.nl-box {
		display: flex;
		flex-direction: column;
		gap: 8px;
		background-color: var(--surface-secondary);
		padding: 12px;
		border-radius: var(--radius-md);
		border: 1px solid var(--border-color);
	}

	.nl-label {
		display: flex;
		align-items: center;
		gap: 6px;
		font-size: 12px;
		font-weight: 600;
		color: var(--primary-blue);
	}

	:global(.sparkle-icon) {
		color: var(--primary-blue);
	}

	.nl-input-row {
		display: flex;
		gap: 8px;
	}

	.parse-btn {
		white-space: nowrap;
	}

	.error-text {
		font-size: 12px;
		color: var(--danger);
	}

	.expense-form {
		display: flex;
		flex-direction: column;
		gap: 14px;
	}

	.form-grid-2 {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 12px;
	}

	.form-group {
		display: flex;
		flex-direction: column;
		gap: 6px;
	}

	.form-label {
		font-size: 13px;
		font-weight: 500;
		color: var(--text-primary);
	}

	.form-input {
		width: 100%;
		padding: 8px 12px;
		border-radius: var(--radius-md);
		border: 1px solid var(--border-color);
		background-color: var(--card-bg);
		color: var(--text-primary);
		font-size: 14px;
		outline: none;
	}

	.form-input:focus {
		border-color: var(--primary-blue);
	}

	.font-bold {
		font-weight: 700;
	}

	.modal-footer {
		display: flex;
		justify-content: flex-end;
		gap: 12px;
		margin-top: 10px;
		padding-top: 12px;
		border-top: 1px solid var(--border-color);
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
