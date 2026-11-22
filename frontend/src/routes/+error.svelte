<script lang="ts">
	import { page } from '$app/state';
	import { resolve } from '$app/paths';
	import { AlertTriangle, Home, RotateCcw } from '@lucide/svelte';
</script>

<div class="error-container">
	<div class="error-card">
		<div class="error-icon-box">
			<AlertTriangle size={36} />
		</div>

		<h1 class="error-title">
			{page.status === 404 ? 'Page Not Found' : 'Something Went Wrong'}
		</h1>

		<p class="error-message">
			{page.error?.message || 'An unexpected error occurred while loading this page.'}
		</p>

		<div class="error-actions">
			<button type="button" class="btn btn-secondary" onclick={() => window.location.reload()}>
				<RotateCcw size={16} />
				<span>Reload Page</span>
			</button>
			<a href={resolve('/')} class="btn btn-primary">
				<Home size={16} />
				<span>Go to Overview</span>
			</a>
		</div>
	</div>
</div>

<style>
	.error-container {
		display: flex;
		align-items: center;
		justify-content: center;
		min-height: calc(100vh - 200px);
		padding: 32px 16px;
	}

	.error-card {
		background: var(--surface);
		border: 1px solid var(--border-color);
		border-radius: var(--radius-lg);
		padding: 40px;
		max-width: 520px;
		width: 100%;
		text-align: center;
		box-shadow: var(--shadow-md);
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 16px;
	}

	.error-icon-box {
		width: 64px;
		height: 64px;
		border-radius: 50%;
		background: rgba(214, 69, 69, 0.1);
		color: var(--danger);
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.error-title {
		font-size: 22px;
		font-weight: 700;
		color: var(--text-primary);
		margin: 0;
	}

	.error-message {
		font-size: 14px;
		color: var(--text-secondary);
		line-height: 1.5;
		margin: 0;
	}

	.error-actions {
		display: flex;
		align-items: center;
		gap: 12px;
		margin-top: 12px;
	}
</style>
