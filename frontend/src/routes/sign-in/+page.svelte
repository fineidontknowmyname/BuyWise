<script lang="ts">
	import type { PageProps } from './$types';
	let { form }: PageProps = $props();
	let registering = $state(false);
	const register = $derived(form?.register ?? registering);
</script>

<main class="auth">
	<section class="card">
		<p class="wordmark">BuyWise<span>.</span></p>
		<h1>{register ? 'Create your account' : 'Sign in to BuyWise'}</h1>
		{#if form?.message}<p role="alert">{form.message}</p>{/if}
		<form method="POST">
			<label
				>Email <input
					type="email"
					name="email"
					autocomplete="email"
					maxlength="254"
					required
				/></label
			>
			<label
				>Password <input
					type="password"
					name="password"
					autocomplete={register ? 'new-password' : 'current-password'}
					minlength={register ? 15 : undefined}
					maxlength="256"
					required
				/></label
			>
			<button class="primary" name="intent" value={register ? 'register' : 'login'}>
				{register ? 'Create account' : 'Sign in'}
			</button>
		</form>
		<button
			class="switch"
			type="button"
			onclick={() => {
				registering = !register;
				form = null;
			}}
		>
			{register ? 'Already have an account? Sign in' : 'Create an account'}
		</button>
	</section>
</main>

<style>
	.auth {
		min-height: 100dvh;
		display: grid;
		place-items: center;
		padding: 1.25rem;
		color: #0d0d0d;
		background: #f4f4f2;
		font-family: Geist, system-ui, sans-serif;
	}
	.card {
		width: min(100%, 26rem);
		padding: 2rem;
		border: 1px solid #dadad6;
		border-radius: 22px;
		background: #fff;
	}
	.wordmark {
		font:
			600 1.2rem Inconsolata,
			monospace;
	}
	.wordmark span {
		color: #aca000;
	}
	h1 {
		font-size: 1.75rem;
		letter-spacing: -0.04em;
		margin: 0.75rem 0 1.25rem;
	}
	form,
	label {
		display: grid;
		gap: 0.55rem;
	}
	form {
		gap: 1rem;
	}
	input,
	button {
		min-height: 2.75rem;
		font: inherit;
		border-radius: 12px;
	}
	input {
		width: 100%;
		border: 1px solid #dadad6;
		padding: 0.5rem 0.65rem;
		background: transparent;
		color: inherit;
	}
	.primary {
		border: 0;
		background: #fff01f;
		color: #0d0d0d;
		font-weight: 600;
	}
	.switch {
		margin-top: 0.75rem;
		border: 0;
		background: transparent;
		color: inherit;
		text-decoration: underline;
	}
	button {
		cursor: pointer;
		transition:
			translate 0.3s cubic-bezier(0.34, 1.5, 0.64, 1),
			opacity 0.2s;
	}
	button:hover {
		opacity: 0.88;
	}
	:focus-visible {
		outline: 2px solid currentColor;
		outline-offset: 3px;
	}
	@media (prefers-color-scheme: dark) {
		.auth {
			background: #0b0b0b;
			color: #f1f1ee;
		}
		.card {
			background: #141414;
		}
		.card,
		input {
			border-color: #2a2a28;
		}
	}
	@media (prefers-reduced-motion: reduce) {
		button {
			transition: none;
		}
	}
</style>
