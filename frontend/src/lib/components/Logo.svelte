<script lang="ts">
	let {
		size = 'md',
		showWordmark = true
	}: {
		size?: 'sm' | 'md' | 'lg';
		showWordmark?: boolean;
	} = $props();

	const iconSizes = {
		sm: 26,
		md: 32,
		lg: 40
	};
</script>

<div class="logo-container logo-{size}">
	<svg
		viewBox="0 0 32 32"
		width={iconSizes[size]}
		height={iconSizes[size]}
		fill="none"
		xmlns="http://www.w3.org/2000/svg"
		class="logo-icon"
		aria-hidden="true"
	>
		<rect width="32" height="32" rx="9" fill="url(#bw-grad)" />
		<path
			d="M7 11C7 9.89543 7.89543 9 9 9H23C24.1046 9 25 9.89543 25 11V21C25 22.1046 24.1046 23 23 23H9C7.89543 23 7 22.1046 7 21V11Z"
			fill="#193A5A"
			fill-opacity="0.3"
		/>
		<!-- Upward growth arrow through wallet -->
		<path
			d="M9 18L14 13L18 17L23 10.5"
			stroke="#C4EE1B"
			stroke-width="2.6"
			stroke-linecap="round"
			stroke-linejoin="round"
		/>
		<path
			d="M18.5 10.5H23V15"
			stroke="#C4EE1B"
			stroke-width="2.6"
			stroke-linecap="round"
			stroke-linejoin="round"
		/>
		<defs>
			<linearGradient id="bw-grad" x1="0" y1="0" x2="32" y2="32" gradientUnits="userSpaceOnUse">
				<stop stop-color="#24507B" />
				<stop offset="1" stop-color="#193A5A" />
			</linearGradient>
		</defs>
	</svg>

	{#if showWordmark}
		<div class="wordmark-group">
			<span class="brand-name">Buy<span class="brand-accent">Wise</span></span>
			<span class="brand-tagline">Plan smarter. Buy wiser.</span>
		</div>
	{/if}
</div>

<style>
	.logo-container {
		display: inline-flex;
		align-items: center;
		gap: 12px;
		user-select: none;
	}

	.logo-icon {
		flex-shrink: 0;
		filter: drop-shadow(0 2px 4px rgba(22, 139, 219, 0.25));
	}

	.wordmark-group {
		display: flex;
		flex-direction: column;
		line-height: 1;
	}

	.brand-name {
		font-size: 1.25rem;
		font-weight: 700;
		letter-spacing: -0.03em;
		color: var(--foreground);
	}

	.brand-accent {
		color: var(--primary);
	}

	.brand-tagline {
		font-size: 0.65rem;
		font-weight: 500;
		color: var(--foreground-muted);
		letter-spacing: 0.02em;
		margin-top: 3px;
	}

	.logo-sm .brand-name {
		font-size: 1.05rem;
	}

	.logo-lg .brand-name {
		font-size: 1.45rem;
	}
</style>
