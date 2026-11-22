<script lang="ts">
	import { themeManager } from '#lib/stores/theme.svelte.js';

	let { reducedMotion = false }: { reducedMotion?: boolean } = $props();
	let canvas: HTMLCanvasElement;

	$effect(() => {
		const dark = themeManager.isDark;
		const reduce = reducedMotion;
		const preference = window.matchMedia('(prefers-reduced-motion: reduce)');
		const context = canvas.getContext('2d');
		if (!context) return;

		let width = 0;
		let height = 0;
		let pointer: { x: number; y: number } | undefined;
		let frame = 0;
		let lastPaint = -Infinity;

		function draw(time: number) {
			if (!context || !width || !height) return;
			context.fillStyle = dark ? '#080808' : '#a5a5a1';
			context.fillRect(0, 0, width, height);
			const stepX = 25;
			const stepY = 9;
			for (let row = 0; row * stepY < height + stepY; row++) {
				for (let col = -1; col * stepX < width + stepX; col++) {
					const x = col * stepX + ((row % 2) * stepX) / 2;
					const y = row * stepY + 4;
					const light = pointer
						? Math.max(0, 1 - Math.hypot(x - pointer.x, y - pointer.y) / 90)
						: 0;
					const wave = 0.45 + 0.4 * Math.sin(x * 0.012 + row * 0.7 - time * 0.0006);
					const level = Math.min(1, wave + light * 0.35);
					const shade = Math.round(dark ? 20 + level * 128 : 167 + level * 83);
					context.fillStyle = `rgb(${shade} ${shade} ${shade})`;
					context.beginPath();
					context.ellipse(
						x,
						y - light * 2,
						stepX * 0.57,
						stepY * 0.43,
						(x / width - 0.5) * 0.5,
						0,
						Math.PI * 2
					);
					context.fill();
				}
			}
		}

		function animate(time: number) {
			if (time - lastPaint >= 1000 / 30) {
				draw(time);
				lastPaint = time;
			}
			frame = requestAnimationFrame(animate);
		}

		function synchronize() {
			cancelAnimationFrame(frame);
			if (reduce || preference.matches || document.hidden) {
				pointer = undefined;
				draw(0);
			} else {
				frame = requestAnimationFrame(animate);
			}
		}

		const observer = new ResizeObserver(() => {
			const bounds = canvas.getBoundingClientRect();
			const scale = Math.min(2, window.devicePixelRatio || 1);
			width = bounds.width;
			height = bounds.height;
			canvas.width = Math.round(width * scale);
			canvas.height = Math.round(height * scale);
			context.setTransform(scale, 0, 0, scale, 0, 0);
			draw(0);
		});
		const header = canvas.parentElement;
		function track(event: PointerEvent) {
			if (reduce || preference.matches) return;
			const bounds = canvas.getBoundingClientRect();
			pointer = { x: event.clientX - bounds.left, y: event.clientY - bounds.top };
		}
		function reset() {
			pointer = undefined;
		}

		observer.observe(canvas);
		header?.addEventListener('pointermove', track);
		header?.addEventListener('pointerleave', reset);
		document.addEventListener('visibilitychange', synchronize);
		preference.addEventListener('change', synchronize);
		synchronize();
		return () => {
			cancelAnimationFrame(frame);
			observer.disconnect();
			header?.removeEventListener('pointermove', track);
			header?.removeEventListener('pointerleave', reset);
			document.removeEventListener('visibilitychange', synchronize);
			preference.removeEventListener('change', synchronize);
		};
	});
</script>

<canvas bind:this={canvas} aria-hidden="true"></canvas>

<style>
	canvas {
		position: absolute;
		inset: 0;
		width: 100%;
		height: 100%;
		pointer-events: none;
	}
</style>
