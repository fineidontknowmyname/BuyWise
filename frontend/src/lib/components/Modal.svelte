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

	const isVisible = $derived(isOpen ?? open);

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape' && isVisible) {
			if (onclose) onclose();
		}
	}
</script>

<svelte:window onkeydown={handleKeydown} />

{#if isVisible}
	<div
		class="modal-backdrop"
		role="presentation"
		onclick={(e) => {
			if (e.target === e.currentTarget && onclose) onclose();
		}}
	>
		<div
			class="modal-card"
			role="dialog"
			aria-modal="true"
			aria-labelledby={title ? 'modal-title' : undefined}
		>
			<div class="modal-header">
				{#if title}
					<h3 id="modal-title" class="modal-title">{title}</h3>
				{/if}
				{#if onclose}
					<button type="button" class="btn-icon btn-sm" onclick={onclose} aria-label="Close modal">
						<X size={18} />
					</button>
				{/if}
			</div>
			<div class="modal-body">
				{@render children()}
			</div>
		</div>
	</div>
{/if}

<style>
	.modal-body {
		display: flex;
		flex-direction: column;
	}
</style>
