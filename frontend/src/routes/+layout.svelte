<script lang="ts">
	import '../app.css';
	import { tick } from 'svelte';
	import { page } from '$app/state';
	import favicon from '#lib/assets/favicon.svg';
	import Sidebar from '#lib/components/Sidebar.svelte';
	import Header from '#lib/components/Header.svelte';
	import Toast from '#lib/components/Toast.svelte';
	import type { LayoutProps } from './$types';

	let { children, data }: LayoutProps = $props();
	let desktop = $state(false);
	let menuOpen = $state(false);

	$effect(() => {
		const viewport = window.matchMedia('(min-width: 1024px)');
		function update() {
			desktop = viewport.matches;
			menuOpen = desktop;
		}
		update();
		viewport.addEventListener('change', update);
		return () => viewport.removeEventListener('change', update);
	});

	function closeMenu(restoreFocus = false) {
		menuOpen = false;
		if (restoreFocus) document.getElementById('navigation-toggle')?.focus();
	}

	async function toggleMenu() {
		if (menuOpen) {
			closeMenu();
			return;
		}
		menuOpen = true;
		if (!desktop) {
			await tick();
			document.querySelector<HTMLElement>('#app-navigation a')?.focus();
		}
	}

	function handleKey(event: KeyboardEvent) {
		const focus = document.activeElement;
		if (
			event.key === 'Escape' &&
			menuOpen &&
			(!desktop ||
				focus?.id === 'navigation-toggle' ||
				document.getElementById('app-navigation')?.contains(focus))
		) {
			closeMenu(true);
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
		<Header
			{menuOpen}
			reducedMotion={data.workspacePreferences?.reducedMotion}
			ontoggle={toggleMenu}
		/>
		<div class="app-body">
			<Sidebar
				isOpen={menuOpen}
				onfollow={() => {
					if (!desktop) closeMenu();
				}}
			/>
			{#if menuOpen && !desktop}
				<button
					class="navigation-scrim"
					type="button"
					tabindex="-1"
					aria-label="Dismiss navigation menu"
					onclick={() => closeMenu(true)}
				></button>
			{/if}
			<main class="content-container" inert={menuOpen && !desktop}>
				{@render children()}
			</main>
		</div>
	</div>

	<Toast />
{/if}

<style>
	.navigation-scrim {
		display: none;
	}
	@media (width < 1024px) {
		.navigation-scrim {
			display: block;
			position: fixed;
			inset: var(--topbar-height) 0 0;
			z-index: 40;
			border: 0;
			background: rgb(0 0 0 / 42%);
		}
	}
	.reduced-motion :global(*) {
		animation-duration: 0.01ms !important;
		transition-duration: 0.01ms !important;
		scroll-behavior: auto !important;
	}
</style>
