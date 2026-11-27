<script lang="ts">
	import { Sun, Moon } from '@lucide/svelte';
	import { themeManager } from '#lib/stores/theme.svelte.js';
</script>

<div class="theme-toggle" role="group" aria-label="Color theme">
	<button
		type="button"
		aria-label="Dark theme"
		aria-pressed={themeManager.isDark}
		onclick={() => themeManager.set('dark')}
	>
		<Moon size={16} />
	</button>
	<button
		type="button"
		aria-label="Light theme"
		aria-pressed={!themeManager.isDark}
		onclick={() => themeManager.set('light')}
	>
		<Sun size={16} />
	</button>
</div>

<style>
	.theme-toggle {
		position: relative;
		display: flex;
		flex: none;
		padding: 3px;
		border: 1px solid var(--border);
		border-radius: 999px;
		background: var(--surface);
	}

	button {
		display: grid;
		width: var(--control-height);
		height: var(--control-height);
		place-items: center;
		border-radius: 999px;
		color: var(--foreground-secondary);
		transition:
			background-color 0.2s,
			color 0.2s;
	}

	button[aria-pressed='true'] {
		background: var(--primary);
		color: var(--primary-foreground);
	}
</style>
