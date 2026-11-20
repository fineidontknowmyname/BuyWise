<script lang="ts">
	import { Menu, X } from '@lucide/svelte';
	import { resolve } from '$app/paths';
	import ThemeToggle from './ThemeToggle.svelte';
	import HeaderWeave from './HeaderWeave.svelte';

	let {
		menuOpen,
		reducedMotion = false,
		ontoggle
	}: { menuOpen: boolean; reducedMotion?: boolean; ontoggle: () => void } = $props();
</script>

<header class="topbar">
	<HeaderWeave {reducedMotion} />
	<button
		id="navigation-toggle"
		type="button"
		class="btn-icon menu-toggle"
		onclick={ontoggle}
		aria-label={menuOpen ? 'Close navigation menu' : 'Open navigation menu'}
		aria-expanded={menuOpen}
		aria-controls="app-navigation"
	>
		{#if menuOpen}<X size={21} />{:else}<Menu size={21} />{/if}
	</button>
	<a class="wordmark" href={resolve('/')}>
		<strong>BuyWise<span>.</span></strong>
		<small>Buy what you want. Without borrowing for it.</small>
	</a>
	<ThemeToggle />
</header>

<style>
	.menu-toggle,
	.wordmark {
		position: relative;
	}

	.menu-toggle {
		flex: none;
		width: 42px;
		height: 42px;
		background: var(--surface);
		color: var(--foreground);
	}

	.wordmark {
		display: flex;
		flex: 1;
		min-width: 0;
		align-items: baseline;
		justify-content: center;
		gap: 12px;
		padding: 7px 12px;
		border: 1px solid var(--border);
		border-radius: 9px;
		background: color-mix(in srgb, var(--surface) 92%, transparent);
		color: var(--foreground);
		font-family: var(--font-mono);
	}

	.wordmark strong {
		font-size: 1.2rem;
		white-space: nowrap;
	}
	.wordmark strong span {
		color: var(--brand-dot);
	}
	.wordmark small {
		color: var(--foreground-secondary);
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	@media (max-width: 720px) {
		.wordmark small {
			display: none;
		}
	}
</style>
