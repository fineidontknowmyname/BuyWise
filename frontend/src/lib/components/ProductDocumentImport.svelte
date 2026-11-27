<script lang="ts">
	import { deserialize } from '$app/forms';
	import { invalidateAll } from '$app/navigation';
	import { documentProductLabels, documentIdentityConflicts } from '#lib/documentProduct.js';
	import { parseRupees } from '#lib/money.js';
	import type { Product } from '#lib/server/api.js';
	import type {
		DocumentPreviewResult,
		DocumentProductDetails
	} from '#lib/server/documents/types.js';
	let { product }: { product: Product } = $props();
	let preview = $state<DocumentPreviewResult | null>(null);
	let selected = $state<Array<keyof DocumentProductDetails>>([]);
	let draft = $state<Record<string, string>>({});
	let busy = $state(false);
	let message = $state('');
	let failure = $state('');
	let confirmed = $state(false);
	const fields = $derived(
		Object.keys(preview?.productDetails ?? {}) as Array<keyof DocumentProductDetails>
	);
	const conflicts = $derived(
		documentIdentityConflicts(
			{ model: product.model, variant: product.variant, url: product.url },
			{
				model: selected.includes('model') ? draft.model : undefined,
				variant: selected.includes('variant') ? draft.variant : undefined,
				url: selected.includes('url') ? draft.url : undefined
			}
		)
	);
	const needsConfirmation = $derived(
		conflicts.length > 0 || preview?.identityMatch?.matches === false
	);

	async function upload(event: SubmitEvent) {
		event.preventDefault();
		const form = event.currentTarget as HTMLFormElement;
		busy = true;
		failure = '';
		message = '';
		preview = null;
		confirmed = false;
		try {
			const response = await fetch('?/uploadDocument', {
				method: 'POST',
				headers: { Accept: 'application/json' },
				body: new FormData(form)
			});
			const result = deserialize<{ preview: DocumentPreviewResult }, { error: string }>(
				await response.text()
			);
			if (result.type !== 'success' || !result.data?.preview)
				throw new Error(
					result.type === 'failure' ? result.data?.error : 'Unable to read this document.'
				);
			preview = result.data.preview;
			selected = Object.keys(preview.productDetails ?? {}) as Array<keyof DocumentProductDetails>;
			draft = Object.fromEntries(
				Object.entries(preview.productDetails ?? {}).map(([key, value]) => [
					key,
					key.endsWith('Paise')
						? String(Number(value) / 100)
						: typeof value === 'object'
							? JSON.stringify(value, null, 2)
							: String(value)
				])
			);
		} catch (err) {
			failure = err instanceof Error ? err.message : 'Upload failed.';
		} finally {
			busy = false;
		}
	}

	async function save() {
		busy = true;
		failure = '';
		message = '';
		try {
			const details: Record<string, unknown> = {};
			for (const key of selected) {
				if (key.endsWith('Paise')) {
					const amount = parseRupees(draft[key]);
					if (amount === null) throw new Error('Enter valid prices with up to two decimal places.');
					details[key] = amount;
				} else details[key] = key === 'specifications' ? JSON.parse(draft[key]) : draft[key];
			}
			const body = new FormData();
			body.set('details', JSON.stringify(details));
			body.set('identityConfirmed', confirmed ? 'yes' : 'no');
			const response = await fetch('?/saveDocumentDetails', {
				method: 'POST',
				headers: { Accept: 'application/json' },
				body
			});
			const result = deserialize<{ notice: string }, { error: string }>(await response.text());
			if (result.type !== 'success')
				throw new Error(
					result.type === 'failure' ? result.data?.error : 'Unable to save product details.'
				);
			await invalidateAll();
			preview = null;
			message = 'Product details saved.';
		} catch (err) {
			failure = err instanceof Error ? err.message : 'Unable to save product details.';
		} finally {
			busy = false;
		}
	}
</script>

<details class="wb-fold">
	<summary>Import from document</summary>
	<div class="wb-fold-body">
		<form
			method="POST"
			action="?/uploadDocument"
			enctype="multipart/form-data"
			onsubmit={upload}
			class="wb-stack"
		>
			<label
				>Product document<input
					type="file"
					name="docFile"
					accept=".pdf,.docx,.csv,.xlsx,.xls,.png,.jpg,.jpeg,.webp"
					required
					disabled={busy}
				/></label
			>
			<button class="btn btn-secondary" disabled={busy}
				>{busy ? 'Working…' : 'Read document'}</button
			>
		</form>
		{#if failure}<p class="wb-notice" role="alert">{failure}</p>{/if}
		{#if message}<p class="wb-notice" role="status">{message}</p>{/if}
		{#if preview}
			<p class="wb-muted">
				From {preview.filename}. Choose fields to save; other values stay unchanged. Document prices
				are not live quotes.
			</p>
			{#each preview.warnings ?? [] as warning (warning)}<p class="wb-muted">{warning}</p>{/each}
			<div class="wb-stack">
				{#each fields as key (key)}
					<div class="review-field">
						<label class="wb-check"
							><input
								type="checkbox"
								value={key}
								bind:group={selected}
								disabled={busy}
							/>{documentProductLabels[key]}</label
						>
						<label
							><span class="sr-only">{documentProductLabels[key]} value</span>
							{#if key === 'specifications' || key === 'notes'}
								<textarea
									rows="4"
									bind:value={draft[key]}
									disabled={busy || !selected.includes(key)}></textarea>
							{:else if key === 'availability'}
								<select bind:value={draft[key]} disabled={busy || !selected.includes(key)}
									><option value="unknown">Unconfirmed</option><option value="in_stock"
										>In stock</option
									><option value="out_of_stock">Out of stock</option><option value="pre_order"
										>Pre-order</option
									></select
								>
							{:else}
								<input
									type="text"
									inputmode={key.endsWith('Paise') ? 'decimal' : 'text'}
									bind:value={draft[key]}
									disabled={busy || !selected.includes(key)}
								/>
							{/if}
						</label>
					</div>
				{/each}
				{#if needsConfirmation}<p class="wb-muted">
						Check the product identity{conflicts.length ? ': ' + conflicts.join(', ') : ''}.
						Existing tracked models and variants remain protected.
					</p>
					<label class="wb-check"
						><input type="checkbox" bind:checked={confirmed} />I checked that these details belong
						to this product.</label
					>{/if}
				<button
					type="button"
					class="btn btn-primary"
					onclick={save}
					disabled={busy || !selected.length || (needsConfirmation && !confirmed)}
					>Save selected details</button
				>
			</div>
		{/if}
	</div>
</details>

<style>
	.wb-fold-body {
		min-width: 0;
		overflow-wrap: anywhere;
	}
	input[type='file'] {
		width: 100%;
		min-width: 0;
		font-size: 12px;
	}
	.review-field {
		display: grid;
		gap: 6px;
		min-width: 0;
	}
	textarea {
		width: 100%;
		min-width: 0;
		resize: vertical;
		font-family: inherit;
	}
	.sr-only {
		position: absolute;
		width: 1px;
		height: 1px;
		overflow: hidden;
		clip-path: inset(50%);
	}
</style>
