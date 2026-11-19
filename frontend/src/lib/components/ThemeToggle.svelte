<script lang="ts">
	import { Sun, Moon } from '@lucide/svelte';
	import { themeManager } from '#lib/stores/theme.svelte.js';
</script>

<button
	type="button"
	class="btn-icon theme-toggle"
	onclick={() => themeManager.toggle()}
	aria-label="Switch theme (currently {themeManager.isDark ? 'dark' : 'light'} mode)"
	title="Switch to {themeManager.isDark ? 'light' : 'dark'} mode"
>
	{#if themeManager.isDark}
		<Sun size={19} />
	{:else}
		<Moon size={19} />
	{/if}
</button>

<style>
	.theme-toggle {
		position: relative;
		transition:
			transform 0.2s ease,
			background-color 0.15s ease;
	}

	.theme-toggle:active {
		transform: scale(0.92);
	}
</style>
