<script lang="ts">
	import '../app.css';
	import { page } from '$app/state';
	import favicon from '#lib/assets/favicon.svg';
	import Sidebar from '#lib/components/Sidebar.svelte';
	import Header from '#lib/components/Header.svelte';
	import Toast from '#lib/components/Toast.svelte';
	import type { LayoutProps } from './$types';

	let { children, data }: LayoutProps = $props();

	let mobileMenuOpen = $state(false);

	function toggleMobileMenu() {
		mobileMenuOpen = !mobileMenuOpen;
	}

	function closeMobileMenu() {
		mobileMenuOpen = false;
	}
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
	<link rel="manifest" href="/manifest.webmanifest" />
	<meta name="theme-color" content="#24507B" />
	<title>BuyWise — Plan smarter. Buy wiser.</title>
</svelte:head>

{#if page.url.pathname === '/sign-in'}
	{@render children()}
{:else}
	<div class="app-shell" class:reduced-motion={data.workspacePreferences?.reducedMotion}>
		<Sidebar isOpen={mobileMenuOpen} onclose={closeMobileMenu} />

		<div class="main-wrapper">
			<Header ontoggleMobileMenu={toggleMobileMenu} />
			<main class="content-container">
				{@render children()}
			</main>
		</div>
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
