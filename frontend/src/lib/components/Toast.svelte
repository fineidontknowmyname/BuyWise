<script lang="ts">
	import { toast } from '#lib/stores/toast.svelte.js';
	import { CheckCircle2, AlertCircle, Info, X } from '@lucide/svelte';
</script>

{#if toast.items.length > 0}
	<div class="toast-container" role="region" aria-label="Notifications">
		{#each toast.items as item (item.id)}
			<div class="toast toast-{item.type}">
				<div class="toast-icon">
					{#if item.type === 'success'}
						<CheckCircle2 size={18} color="var(--success)" />
					{:else if item.type === 'error'}
						<AlertCircle size={18} color="var(--danger)" />
					{:else}
						<Info size={18} color="var(--primary)" />
					{/if}
				</div>
				<span class="toast-message">{item.message}</span>
				<button
					type="button"
					class="toast-close"
					onclick={() => toast.dismiss(item.id)}
					aria-label="Dismiss notification"
				>
					<X size={15} />
				</button>
			</div>
		{/each}
	</div>
{/if}

<style>
	.toast-icon {
		display: flex;
		align-items: center;
		flex-shrink: 0;
	}

	.toast-message {
		flex: 1;
		word-break: break-word;
	}

	.toast-close {
		display: flex;
		align-items: center;
		color: var(--foreground-muted);
		padding: 2px;
		border-radius: 4px;
		transition: color 0.15s ease;
	}

	.toast-close:hover {
		color: var(--foreground);
	}
</style>
