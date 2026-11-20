<script lang="ts">
	import { resolve } from '$app/paths';
	import { enhance } from '$app/forms';
	import { themeManager, type ThemePreference } from '#lib/stores/theme.svelte.js';
	import type { PageProps } from './$types';
	let { data, form }: PageProps = $props();
</script>

<svelte:head><title>Profile | BuyWise</title></svelte:head>
<h1>Profile</h1>
{#if form?.message}<p role="status">{form.message}</p>{/if}
{#if form?.error}<p role="alert">{form.error}</p>{/if}
<div class="sections">
	<section class="card">
		<h2>Personal details and preferences</h2>
		<p>Signed in as {data.email ?? 'Unavailable'}</p>
		{#if data.workspacePreferences}
			<form method="POST" action="?/preferences" use:enhance>
				<label
					>Display name <input
						class="input"
						name="displayName"
						maxlength="100"
						value={data.workspacePreferences.displayName}
					/></label
				>
				<label
					><input
						type="checkbox"
						name="priceUpdates"
						value="yes"
						checked={data.workspacePreferences.priceUpdates}
					/> Show price updates on the overview</label
				>
				<label
					><input
						type="checkbox"
						name="reducedMotion"
						value="yes"
						checked={data.workspacePreferences.reducedMotion}
					/> Reduce motion</label
				>
				<div><button class="btn btn-primary">Save preferences</button></div>
			</form>
		{:else}<p>Preferences are temporarily unavailable.</p>{/if}
		<label
			>Theme on this device <select
				class="input"
				value={themeManager.preference}
				onchange={(event) => themeManager.set(event.currentTarget.value as ThemePreference)}
				><option value="system">System</option><option value="light">Light</option><option
					value="dark">Dark</option
				></select
			></label
		>
	</section>
	<section class="card">
		<h2>Financial assumptions</h2>
		<p>Manage your variable income confidence, safety buffer and emergency reserve target.</p>
		<div><a class="btn btn-secondary" href={resolve('/finances')}>Edit financial profile</a></div>
	</section>
	<section class="card">
		<h2>Account and privacy</h2>
		<form method="POST" action="?/revoke" use:enhance>
			<label
				><input type="checkbox" name="confirm" value="yes" required /> Sign out all other sessions</label
			>
			<div><button class="btn btn-secondary">Confirm sign-out</button></div>
		</form>
		<div>
			<a class="btn btn-secondary" href={resolve('/profile/export')} download
				>Download workspace snapshot</a
			>
		</div>
		<p>
			The JSON snapshot includes profile settings, account balances, goals, products and the current
			budget.
		</p>
	</section>
</div>

<style>
	.sections,
	section,
	form {
		display: grid;
		gap: 1rem;
	}
	.sections {
		max-width: 60rem;
	}
	section {
		padding: clamp(1rem, 3vw, 1.5rem);
	}
	h2,
	p {
		margin: 0;
	}
	.input {
		display: block;
		max-width: 24rem;
		margin-top: 0.4rem;
	}
</style>
