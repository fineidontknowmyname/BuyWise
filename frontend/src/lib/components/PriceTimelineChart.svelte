<script lang="ts">
	import { formatRupees } from '#lib/money.js';
	import type { PriceObservation } from '#lib/server/api.js';
	import { Calendar } from '@lucide/svelte';

	let {
		prices = [],
		targetPricePaise = 0
	}: {
		prices?: PriceObservation[];
		targetPricePaise?: number;
	} = $props();

	let hoveredIndex = $state<number | null>(null);

	// Sort chronological
	const sortedPrices = $derived(
		[...prices].sort((a, b) => new Date(a.observedOn).getTime() - new Date(b.observedOn).getTime())
	);

	const minPrice = $derived.by(() => {
		if (sortedPrices.length === 0) return 0;
		return Math.min(...sortedPrices.map((p) => p.pricePaise));
	});

	const maxPrice = $derived.by(() => {
		if (sortedPrices.length === 0) return 0;
		const observedMax = Math.max(...sortedPrices.map((p) => p.pricePaise));
		return Math.max(observedMax, targetPricePaise || 0);
	});

	const avgPrice = $derived.by(() => {
		if (sortedPrices.length === 0) return 0;
		const sum = sortedPrices.reduce((acc, p) => acc + p.pricePaise, 0);
		return Math.round(sum / sortedPrices.length);
	});

	// SVG layout bounds
	const width = 680;
	const height = 260;
	const padding = { top: 30, right: 30, bottom: 40, left: 60 };
	const innerWidth = width - padding.left - padding.right;
	const innerHeight = height - padding.top - padding.bottom;

	// Range calculation with 10% breathing room
	const yMin = $derived(Math.max(0, Math.floor(minPrice * 0.9)));
	const yMax = $derived(Math.ceil(maxPrice * 1.1) || 10000);

	function getX(index: number): number {
		if (sortedPrices.length <= 1) return padding.left + innerWidth / 2;
		return padding.left + (index / (sortedPrices.length - 1)) * innerWidth;
	}

	function getY(paise: number): number {
		if (yMax === yMin) return padding.top + innerHeight / 2;
		const ratio = (paise - yMin) / (yMax - yMin);
		return padding.top + innerHeight - ratio * innerHeight;
	}

	const pointsString = $derived(
		sortedPrices.map((p, i) => `${getX(i)},${getY(p.pricePaise)}`).join(' ')
	);

	const areaString = $derived.by(() => {
		if (sortedPrices.length === 0) return '';
		const firstX = getX(0);
		const lastX = getX(sortedPrices.length - 1);
		const bottomY = padding.top + innerHeight;
		return `${pointsString} ${lastX},${bottomY} ${firstX},${bottomY}`;
	});
</script>

<div class="timeline-container">
	{#if sortedPrices.length === 0}
		<div class="chart-empty">
			<Calendar size={36} class="empty-icon" />
			<p>No price timeline points yet.</p>
			<span>Add manual observations or upload a CSV to visualize price changes over time.</span>
		</div>
	{:else}
		<div class="chart-summary-bar">
			<div class="summary-metric">
				<span class="metric-label">Lowest Ever</span>
				<span class="metric-val min">{formatRupees(minPrice)}</span>
			</div>
			<div class="summary-metric">
				<span class="metric-label">Average</span>
				<span class="metric-val">{formatRupees(avgPrice)}</span>
			</div>
			<div class="summary-metric">
				<span class="metric-label">Highest</span>
				<span class="metric-val">{formatRupees(maxPrice)}</span>
			</div>
			{#if targetPricePaise > 0}
				<div class="summary-metric">
					<span class="metric-label">Target Alert</span>
					<span class="metric-val target">{formatRupees(targetPricePaise)}</span>
				</div>
			{/if}
		</div>

		<div class="svg-wrapper">
			<svg viewBox="0 0 {width} {height}" class="chart-svg" preserveAspectRatio="none">
				<defs>
					<linearGradient id="priceGradient" x1="0" y1="0" x2="0" y2="1">
						<stop offset="0%" stop-color="var(--primary-blue)" stop-opacity="0.3" />
						<stop offset="100%" stop-color="var(--primary-blue)" stop-opacity="0.0" />
					</linearGradient>
				</defs>

				<!-- Grid lines & Y Axis labels -->
				{#each [0, 0.25, 0.5, 0.75, 1] as tick (tick)}
					{@const yVal = Math.round(yMin + tick * (yMax - yMin))}
					{@const yPos = getY(yVal)}
					<line
						x1={padding.left}
						y1={yPos}
						x2={width - padding.right}
						y2={yPos}
						stroke="var(--border-color)"
						stroke-dasharray="3,3"
						opacity="0.6"
					/>
					<text
						x={padding.left - 8}
						y={yPos + 4}
						fill="var(--text-muted)"
						font-size="10"
						text-anchor="end"
					>
						₹{Math.round(yVal / 100).toLocaleString('en-IN')}
					</text>
				{/each}

				<!-- Target Price Line (if set) -->
				{#if targetPricePaise > 0 && targetPricePaise >= yMin && targetPricePaise <= yMax}
					{@const targetY = getY(targetPricePaise)}
					<line
						x1={padding.left}
						y1={targetY}
						x2={width - padding.right}
						y2={targetY}
						stroke="var(--warning)"
						stroke-width="1.5"
						stroke-dasharray="4,4"
					/>
					<text
						x={width - padding.right - 4}
						y={targetY - 6}
						fill="var(--warning)"
						font-size="10"
						font-weight="600"
						text-anchor="end"
					>
						Target: {formatRupees(targetPricePaise)}
					</text>
				{/if}

				<!-- Area Fill -->
				{#if sortedPrices.length > 1}
					<polygon points={areaString} fill="url(#priceGradient)" />
				{/if}

				<!-- Price Line -->
				{#if sortedPrices.length > 1}
					<polyline
						points={pointsString}
						fill="none"
						stroke="var(--primary-blue)"
						stroke-width="2.5"
						stroke-linecap="round"
						stroke-linejoin="round"
					/>
				{/if}

				<!-- Observation Points -->
				{#each sortedPrices as price, i (price.id || i)}
					{@const cx = getX(i)}
					{@const cy = getY(price.pricePaise)}
					{@const isLowest = price.pricePaise === minPrice && sortedPrices.length > 1}
					{@const isHovered = hoveredIndex === i}

					<circle
						{cx}
						{cy}
						r={isHovered ? 6 : isLowest ? 5 : 3.5}
						fill={isLowest ? 'var(--success)' : 'var(--primary-blue)'}
						stroke="#ffffff"
						stroke-width={isHovered ? 2.5 : 1.5}
						class="point-circle"
						role="presentation"
						onmouseenter={() => (hoveredIndex = i)}
						onmouseleave={() => (hoveredIndex = null)}
					/>
				{/each}
			</svg>

			<!-- Hover Tooltip -->
			{#if hoveredIndex !== null && sortedPrices[hoveredIndex]}
				{@const p = sortedPrices[hoveredIndex]}
				{@const leftPos = (getX(hoveredIndex) / width) * 100}
				{@const topPos = (getY(p.pricePaise) / height) * 100}

				<div
					class="chart-tooltip"
					style="left: {leftPos}%; top: {topPos}%; transform: translate(-50%, -120%);"
				>
					<div class="tooltip-price">{formatRupees(p.pricePaise)}</div>
					<div class="tooltip-date">{p.observedOn}</div>
					{#if p.source}
						<div class="tooltip-source">{p.source}</div>
					{/if}
				</div>
			{/if}
		</div>

		<!-- Date labels below chart -->
		<div class="date-labels">
			<span>{sortedPrices[0]?.observedOn}</span>
			{#if sortedPrices.length > 2}
				<span>{sortedPrices[Math.floor(sortedPrices.length / 2)]?.observedOn}</span>
			{/if}
			{#if sortedPrices.length > 1}
				<span>{sortedPrices[sortedPrices.length - 1]?.observedOn}</span>
			{/if}
		</div>
	{/if}
</div>

<style>
	.timeline-container {
		background-color: var(--card-bg);
		border: 1px solid var(--card-border);
		border-radius: var(--radius-lg);
		padding: 20px;
		display: flex;
		flex-direction: column;
		gap: 16px;
		position: relative;
	}

	.chart-empty {
		padding: 40px 20px;
		text-align: center;
		color: var(--text-muted);
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 8px;
	}

	.chart-empty p {
		font-weight: 600;
		color: var(--text-secondary);
	}

	.chart-empty span {
		font-size: 13px;
	}

	:global(.empty-icon) {
		color: var(--text-muted);
		opacity: 0.7;
	}

	.chart-summary-bar {
		display: flex;
		align-items: center;
		gap: 24px;
		flex-wrap: wrap;
		padding-bottom: 12px;
		border-bottom: 1px solid var(--border-color);
	}

	.summary-metric {
		display: flex;
		flex-direction: column;
	}

	.metric-label {
		font-size: 11px;
		text-transform: uppercase;
		letter-spacing: 0.05em;
		color: var(--text-muted);
		font-weight: 600;
	}

	.metric-val {
		font-size: 16px;
		font-weight: 700;
		color: var(--text-primary);
		font-variant-numeric: tabular-nums;
	}

	.metric-val.min {
		color: var(--success);
	}

	.metric-val.target {
		color: var(--warning);
	}

	.svg-wrapper {
		position: relative;
		width: 100%;
		height: 260px;
	}

	.chart-svg {
		width: 100%;
		height: 100%;
		overflow: visible;
	}

	.point-circle {
		cursor: pointer;
		transition:
			r 0.15s ease,
			stroke-width 0.15s ease;
	}

	.chart-tooltip {
		position: absolute;
		pointer-events: none;
		background-color: var(--card-bg);
		border: 1px solid var(--card-border);
		box-shadow: var(--shadow-md);
		border-radius: var(--radius-md);
		padding: 8px 12px;
		font-size: 12px;
		white-space: nowrap;
		z-index: 10;
		display: flex;
		flex-direction: column;
		gap: 2px;
	}

	.tooltip-price {
		font-size: 14px;
		font-weight: 700;
		color: var(--text-primary);
		font-variant-numeric: tabular-nums;
	}

	.tooltip-date {
		color: var(--text-secondary);
		font-size: 11px;
	}

	.tooltip-source {
		color: var(--text-muted);
		font-size: 10px;
		text-transform: uppercase;
		letter-spacing: 0.04em;
	}

	.date-labels {
		display: flex;
		justify-content: space-between;
		font-size: 11px;
		color: var(--text-muted);
		padding: 0 10px;
	}
</style>
