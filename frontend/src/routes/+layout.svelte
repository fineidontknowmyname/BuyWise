<script lang="ts">
	import '../app.css';
	import { page } from '$app/state';
	import favicon from '#lib/assets/favicon.svg';
	import Sidebar from '#lib/components/Sidebar.svelte';
	import Header from '#lib/components/Header.svelte';
	import MobileNavigation from '#lib/components/MobileNavigation.svelte';
	import MoneyNavigation from '#lib/components/MoneyNavigation.svelte';
	import Toast from '#lib/components/Toast.svelte';
	import { isMoneyPath } from '#lib/navigation.js';
	import type { LayoutProps } from './$types';

	let { children, data }: LayoutProps = $props();
	let desktop = $state(false);
	let menuOpen = $state(true);

	$effect(() => {
		const viewport = window.matchMedia('(min-width: 1024px)');
		function update() {
			desktop = viewport.matches;
		}
		update();
		viewport.addEventListener('change', update);
		return () => viewport.removeEventListener('change', update);
	});

	function handleKey(event: KeyboardEvent) {
		if (
			event.key === 'Escape' &&
			desktop &&
			menuOpen &&
			(document.activeElement?.id === 'navigation-toggle' ||
				document.getElementById('app-navigation')?.contains(document.activeElement))
		) {
			menuOpen = false;
			document.getElementById('navigation-toggle')?.focus();
		}
	}
</script>

<svelte:window onkeydown={handleKey} />

<svelte:head>
	<link rel="icon" href={favicon} />
	<link rel="manifest" href="/manifest.webmanifest" />
	<meta name="theme-color" content="#0b0b0b" media="(prefers-color-scheme: dark)" />
	<meta name="theme-color" content="#f4f4f2" media="(prefers-color-scheme: light)" />
	<title>BuyWise - Plan smarter. Buy wiser.</title>
</svelte:head>

{#if page.url.pathname === '/sign-in'}
	{@render children()}
{:else}
	<div class="app-shell" class:reduced-motion={data.workspacePreferences?.reducedMotion}>
		<a class="skip-link" href="#main-content">Skip to content</a>
		<Header
			{menuOpen}
			reducedMotion={data.workspacePreferences?.reducedMotion}
			ontoggle={() => (menuOpen = !menuOpen)}
		/>
		<div class="app-body" class:sidebar-open={menuOpen}>
			<Sidebar isOpen={desktop && menuOpen} />
			<main id="main-content" class="content-container" tabindex="-1">
				{#if isMoneyPath(page.url.pathname)}<MoneyNavigation />{/if}
				{@render children()}
			</main>
		</div>
		<MobileNavigation />
	</div>
	<Toast />
{/if}

<style>
	.reduced-motion :global(*) {
		animation-duration: 0.01ms !important;
		transition-duration: 0.01ms !important;
		scroll-behavior: auto !important;
	}
</style>
