<script lang="ts">
	import { page } from '$app/state';
	import { afterNavigate } from '$app/navigation';
	import { Grid2X2, LogOut } from '@lucide/svelte';
	import Modal from './Modal.svelte';
	import { primaryNavigation, supportingNavigation, isNavigationActive } from '#lib/navigation.js';

	let moreOpen = $state(false);
	const mainItems = primaryNavigation.slice(0, 3);
	const moreItems = [...primaryNavigation.slice(3), ...supportingNavigation];
	const moreActive = $derived(
		moreItems.some((item) => isNavigationActive(page.url.pathname, item.href))
	);

	afterNavigate(() => {
		moreOpen = false;
	});

	$effect(() => {
		const viewport = window.matchMedia('(min-width: 1024px)');
		const close = () => {
			if (viewport.matches) moreOpen = false;
		};
		viewport.addEventListener('change', close);
		return () => viewport.removeEventListener('change', close);
	});
</script>

<nav class="mobile-navigation" aria-label="Mobile navigation">
	{#each mainItems as item (item.href)}
		<a
			href={item.href}
			class:active={isNavigationActive(page.url.pathname, item.href)}
			aria-current={isNavigationActive(page.url.pathname, item.href) ? 'page' : undefined}
		>
			<item.icon size={21} aria-hidden="true" /><span>{item.shortLabel}</span>
		</a>
	{/each}
	<button
		type="button"
		class:active={moreActive || moreOpen}
		aria-haspopup="dialog"
		aria-expanded={moreOpen}
		onclick={() => (moreOpen = true)}
	>
		<Grid2X2 size={21} aria-hidden="true" /><span>More</span>
	</button>
</nav>

<Modal isOpen={moreOpen} title="More from BuyWise" onclose={() => (moreOpen = false)}>
	<nav class="more-links" aria-label="More destinations">
		{#each moreItems as item (item.href)}
			<a
				href={item.href}
				aria-current={isNavigationActive(page.url.pathname, item.href) ? 'page' : undefined}
				onclick={() => (moreOpen = false)}><item.icon size={20} aria-hidden="true" />{item.label}</a
			>
		{/each}
	</nav>
	<form method="POST" action="/sign-out" class="more-links sign-out">
		<button type="submit"><LogOut size={20} aria-hidden="true" />Sign out</button>
	</form>
</Modal>

<style>
	.mobile-navigation {
		display: none;
	}
	.more-links {
		display: grid;
		gap: var(--space-2);
	}
	.more-links a,
	.more-links button {
		display: flex;
		align-items: center;
		gap: var(--space-3);
		min-height: var(--control-height);
		padding: var(--space-3);
		border-radius: var(--radius-control);
		text-align: left;
	}
	.more-links a:hover,
	.more-links button:hover,
	.more-links a[aria-current='page'] {
		background: var(--surface-secondary);
	}
	.sign-out {
		margin-top: var(--space-4);
		border-top: 1px solid var(--border);
		padding-top: var(--space-4);
	}
	@media (width < 1024px) {
		.mobile-navigation {
			position: fixed;
			inset: auto 0 0;
			z-index: 45;
			display: grid;
			grid-template-columns: repeat(4, minmax(0, 1fr));
			padding: var(--space-2) var(--space-2) calc(var(--space-2) + env(safe-area-inset-bottom));
			border-top: 1px solid var(--border);
			background: var(--surface);
		}
		.mobile-navigation a,
		.mobile-navigation button {
			display: grid;
			justify-items: center;
			align-content: center;
			gap: var(--space-1);
			min-height: 52px;
			padding: var(--space-1);
			border-radius: var(--radius-control);
			color: var(--foreground-secondary);
			font-size: var(--text-caption);
			line-height: 1.25;
		}
		.mobile-navigation .active {
			background: var(--accent-subtle);
			color: var(--foreground);
			font-weight: 600;
		}
	}
</style>
