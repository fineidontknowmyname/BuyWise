<script lang="ts">
	import type { Snippet } from 'svelte';
	import { X } from '@lucide/svelte';

	let {
		open = false,
		isOpen,
		title = '',
		onclose,
		children
	}: {
		open?: boolean;
		isOpen?: boolean;
		title?: string;
		onclose?: () => void;
		children: Snippet;
	} = $props();

	const id = $props.id();
	const isVisible = $derived(isOpen ?? open);
	let dialog: HTMLDialogElement;

	$effect(() => {
		if (!isVisible) return;
		dialog.showModal();
		const previousOverflow = document.documentElement.style.overflow;
		document.documentElement.style.overflow = 'hidden';
		return () => {
			dialog.close();
			document.documentElement.style.overflow = previousOverflow;
		};
	});

	function dismissBackdrop(event: MouseEvent) {
		if (event.target !== dialog) return;
		const bounds = dialog.getBoundingClientRect();
		if (
			event.clientX < bounds.left ||
			event.clientX > bounds.right ||
			event.clientY < bounds.top ||
			event.clientY > bounds.bottom
		)
			onclose?.();
	}
</script>

<dialog
	bind:this={dialog}
	class="modal-card"
	aria-labelledby={title ? id : undefined}
	aria-label={title ? undefined : 'Dialog'}
	onclick={dismissBackdrop}
	oncancel={(event) => {
		event.preventDefault();
		onclose?.();
	}}
>
	<div class="modal-header">
		{#if title}<h2 {id} class="modal-title">{title}</h2>{/if}
		{#if onclose}
			<button type="button" class="btn-icon" onclick={onclose} aria-label="Close dialog"
				><X size={20} /></button
			>
		{/if}
	</div>
	<div class="modal-body">{@render children()}</div>
</dialog>

<style>
	dialog.modal-card {
		position: fixed;
		inset: 0;
		margin: auto;
		width: min(640px, calc(100% - 32px));
		max-height: calc(100dvh - 48px);
		overflow: auto;
		overscroll-behavior: contain;
		color: var(--foreground);
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-card);
		padding: var(--space-6);
		box-shadow: var(--shadow-lg);
	}
	dialog::backdrop {
		background: rgb(0 0 0 / 48%);
	}
	.modal-header {
		gap: var(--space-4);
	}
	.modal-title {
		font-size: 1.25rem;
	}
	.modal-header .btn-icon {
		flex: none;
	}
	.modal-body {
		display: flex;
		flex-direction: column;
		gap: var(--space-4);
	}
	@media (max-width: 640px) {
		dialog.modal-card {
			padding: var(--space-4);
			max-height: calc(100dvh - 32px);
		}
	}
</style>
