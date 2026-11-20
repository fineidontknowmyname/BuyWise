<script lang="ts">
	import { native } from './native';
	import { formatRupees, parseRupees } from '../src/lib/money';
	import { todayInIndia } from '../src/lib/dates';
	import type { Account, BudgetReport, Category, Decision, Goal } from '../src/lib/server/api';
	import { onMount } from 'svelte';

	type Candidate = {
		id: string;
		merchant?: string;
		amountPaise: number;
		occurredOn: string;
		sourceApp: string;
		possibleDuplicate: boolean;
		categorySuggestion?: { id: string; name: string; confidence: string };
	};
	type Capacity = {
		profile: { revision: number };
		capacity: { safeSavingsPaise: number };
		emergencySavingsPaise: number;
	};
	let signedIn = $state(false),
		ready = $state(false),
		busy = $state(false),
		cached = $state(false);
	let tab = $state<'overview' | 'expenses' | 'settings'>('overview');
	let email = $state(''),
		password = $state(''),
		message = $state(''),
		failure = $state('');
	let accounts = $state<Account[]>([]),
		categories = $state<Category[]>([]),
		goals = $state<Goal[]>([]);
	let candidates = $state<Candidate[]>([]),
		capacity = $state<Capacity | null>(null),
		budget = $state<BudgetReport | null>(null);
	let activeGoal = $state<Goal | null>(null),
		decision = $state<Decision | null>(null);
	let capture = $state<{
			granted: boolean;
			enabled: boolean;
			packages: string[];
			queued: number;
			rejected?: number;
		} | null>(null),
		packages = $state('');
	let candidate = $state<Candidate | null>(null),
		accountId = $state(''),
		categoryId = $state(''),
		amount = $state(''),
		occurredOn = $state(todayInIndia()),
		note = $state('');
	let allowDuplicate = $state(false),
		duplicate = $state(false),
		expenseKey = '';

	async function request<T>(
		path: string,
		method: 'GET' | 'POST' = 'GET',
		body?: object,
		key?: string
	) {
		const response = await native.request<T>({ path, method, body, key });
		if (response.status === 401) {
			reset();
			throw new Error('Your session ended. Sign in again.');
		}
		if (response.status >= 400) {
			const data = response.data as { error?: string | { message?: string; code?: string } };
			const problem = data?.error;
			if (typeof problem === 'object' && problem?.code === 'possible_duplicate') duplicate = true;
			throw new Error(
				(typeof problem === 'string' ? problem : problem?.message) ??
					'BuyWise could not complete this request.'
			);
		}
		cached ||= !!response.cached;
		return response.data;
	}

	function reset() {
		signedIn = false;
		cached = false;
		accounts = [];
		categories = [];
		goals = [];
		candidates = [];
		capacity = null;
		budget = null;
		activeGoal = null;
		decision = null;
		capture = null;
		packages = '';
		accountId = '';
		message = '';
		tab = 'overview';
		chooseCandidate(null);
	}

	async function run(action: () => Promise<void>) {
		if (busy) return;
		busy = true;
		failure = '';
		message = '';
		try {
			await action();
		} catch (err) {
			failure = err instanceof Error ? err.message : 'The request failed. Try again.';
		} finally {
			busy = false;
		}
	}

	async function load() {
		cached = false;
		const results = await Promise.allSettled([
			request<Account[]>('accounts'),
			request<Category[]>('categories'),
			request<Goal[]>('goals'),
			request<Capacity>('financial-profile'),
			request<BudgetReport>('budgets'),
			request<Candidate[]>('expenses/candidates')
		]);
		if (!signedIn) return;
		const [a, c, g, p, b, n] = results;
		accounts = a.status === 'fulfilled' ? a.value : [];
		categories =
			c.status === 'fulfilled'
				? c.value.filter((item) => item.kind === 'expense' && !item.archived)
				: [];
		goals = g.status === 'fulfilled' ? g.value : [];
		capacity = p.status === 'fulfilled' ? p.value : null;
		budget = b.status === 'fulfilled' ? b.value : null;
		candidates = n.status === 'fulfilled' ? n.value : [];
		if (results.some((result) => result.status === 'rejected'))
			failure = 'Some records could not load. Refresh when you are connected.';
	}

	onMount(() => {
		void run(async () => {
			try {
				signedIn = (await native.status()).signedIn;
				if (signedIn) await load();
			} finally {
				ready = true;
			}
		});
	});

	async function login() {
		try {
			await native.signIn({ email, password });
			signedIn = true;
			await load();
		} finally {
			password = '';
		}
	}

	async function switchTab(next: typeof tab) {
		tab = next;
		if (next === 'settings') {
			capture = await native.captureStatus();
			packages = capture.packages.join(', ');
		}
	}

	function chooseCandidate(item: Candidate | null) {
		candidate = item;
		categoryId = item?.categorySuggestion?.id ?? '';
		amount = item ? (item.amountPaise / 100).toFixed(2) : '';
		note = item?.merchant ?? '';
		occurredOn = item?.occurredOn ?? todayInIndia();
		duplicate = !!item?.possibleDuplicate;
		allowDuplicate = false;
		expenseKey = '';
	}

	async function saveExpense() {
		const amountPaise = parseRupees(amount);
		if (!amountPaise) throw new Error('Enter a positive amount with up to two decimal places.');
		const fields = { accountId, categoryId, amountPaise, occurredOn, note };
		if (candidate)
			await request('expenses/candidates/' + candidate.id + '/confirm', 'POST', {
				...fields,
				confirmed: true,
				allowDuplicate
			});
		else {
			expenseKey ||= crypto.randomUUID();
			await request('expenses', 'POST', fields, expenseKey);
		}
		chooseCandidate(null);
		await load();
		message = 'Expense recorded.';
	}

	async function saveCapture(enabled: boolean) {
		await native.configureCapture({
			enabled,
			packages: packages
				.split(',')
				.map((part) => part.trim())
				.filter(Boolean)
		});
		capture = await native.captureStatus();
		packages = capture.packages.join(', ');
	}
</script>

<header>
	<strong>Buy<span>Wise.</span></strong>{#if signedIn}<button
			onclick={() => run(load)}
			disabled={busy}>Refresh</button
		>{/if}
</header>

<main>
	{#if failure}<p class="alert" role="alert">{failure}</p>{/if}
	{#if message}<p role="status">{message}</p>{/if}
	{#if cached}<p class="alert" role="status">
			Some figures are saved on this device. Connect to refresh them.
		</p>{/if}
	{#if !ready}<p role="status">Opening BuyWise…</p>
	{:else if !signedIn}
		<section>
			<h1>Sign in to BuyWise</h1>
			<form
				onsubmit={(e) => {
					e.preventDefault();
					void run(login);
				}}
			>
				<label
					>Email<input type="email" bind:value={email} required autocomplete="username" /></label
				>
				<label
					>Password<input
						type="password"
						bind:value={password}
						required
						autocomplete="current-password"
					/></label
				>
				<button class="primary" disabled={busy}>Sign in</button>
			</form>
			<button onclick={() => run(() => native.openWeb({ path: '/sign-in' }))}
				>Create an account on BuyWise web</button
			>
		</section>
	{:else if tab === 'overview'}
		<h1>Overview</h1>
		<section>
			<h2>Financial position</h2>
			<dl>
				<div>
					<dt>Monthly saving capacity</dt>
					<dd>
						{capacity?.profile.revision
							? formatRupees(capacity.capacity.safeSavingsPaise)
							: 'Unavailable'}
					</dd>
				</div>
				<div>
					<dt>Emergency savings</dt>
					<dd>{capacity ? formatRupees(capacity.emergencySavingsPaise) : 'Unavailable'}</dd>
				</div>
				<div>
					<dt>Remaining in budgeted categories</dt>
					<dd>{budget?.lines.length ? formatRupees(budget.totalRemainingPaise) : 'Unavailable'}</dd>
				</div>
			</dl>
		</section>
		<section>
			<h2>Active purchase goals</h2>
			{#each goals.filter((goal) => goal.status === 'active') as goal (goal.id)}
				<button
					disabled={busy}
					onclick={() =>
						run(async () => {
							activeGoal = goal;
							decision = null;
							decision = await request<Decision>('goals/' + goal.id + '/decision');
						})}>{goal.name} · {formatRupees(goal.savedAmountPaise ?? 0)} saved</button
				>
			{:else}<p>No active goal data to show.</p>{/each}
			{#if activeGoal}<article>
					<h3>{activeGoal.name}</h3>
					{#if decision}<p>{decision.state.replaceAll('_', ' ')} · {decision.explanation}</p>{/if}
					<button onclick={() => run(() => native.openWeb({ path: '/goals/' + activeGoal?.id }))}
						>Open goal on BuyWise web</button
					>
				</article>{/if}
		</section>
	{:else if tab === 'expenses'}
		<h1>Expenses</h1>
		<section>
			<h2>Suggestions</h2>
			{#each candidates as item (item.id)}<button
					disabled={busy}
					onclick={() => chooseCandidate(item)}
					>{item.merchant ?? item.sourceApp} · {formatRupees(item.amountPaise)}</button
				>
			{:else}<p>No suggestions to review.</p>{/each}
		</section>
		<section>
			<h2>{candidate ? 'Review suggestion' : 'Record expense'}</h2>
			{#if candidate}<p>
					{candidate.sourceApp}{candidate.categorySuggestion
						? ' · ' +
							candidate.categorySuggestion.name +
							' (' +
							candidate.categorySuggestion.confidence +
							')'
						: ''}
				</p>{/if}
			<form
				onsubmit={(e) => {
					e.preventDefault();
					void run(saveExpense);
				}}
			>
				<label>Amount (₹)<input inputmode="decimal" bind:value={amount} required /></label>
				<label
					>Date<input type="date" bind:value={occurredOn} max={todayInIndia()} required /></label
				>
				<label
					>Account<select bind:value={accountId} required
						><option value="">Choose an account</option>{#each accounts as item (item.id)}<option
								value={item.id}>{item.name}</option
							>{/each}</select
					></label
				>
				<label
					>Category<select bind:value={categoryId} required
						><option value="">Choose a category</option>{#each categories as item (item.id)}<option
								value={item.id}>{item.name}</option
							>{/each}</select
					></label
				>
				<label>Note<input bind:value={note} maxlength="500" /></label>
				{#if candidate && duplicate}<label class="check"
						><input type="checkbox" bind:checked={allowDuplicate} /> This is a separate expense</label
					>{/if}
				<button class="primary" disabled={busy}
					>{candidate ? 'Confirm and record' : 'Record expense'}</button
				>
			</form>
			{#if candidate}<button
					disabled={busy}
					onclick={() =>
						run(async () => {
							await request('expenses/candidates/' + candidate?.id + '/reject', 'POST');
							chooseCandidate(null);
							await load();
							message = 'Suggestion dismissed.';
						})}>Dismiss suggestion</button
				>
				<button disabled={busy} onclick={() => chooseCandidate(null)}>Back to manual entry</button
				>{:else}<button disabled={busy} onclick={() => chooseCandidate(null)}
					>Clear expense draft</button
				>{/if}
		</section>
	{:else}
		<h1>Settings</h1>
		<section>
			<h2>Notification capture</h2>
			<p>
				Android notification access covers device notifications. BuyWise only checks apps you name
				for INR payment messages. It keeps structured suggestions on this device and uploads them
				when you tap Sync. Review each suggestion before posting it.
			</p>
			{#if capture}<p>
					Android access {capture.granted ? 'granted' : 'off'} · BuyWise capture {capture.enabled
						? 'on'
						: 'off'} · {capture.queued} queued
				</p>{/if}
			<button disabled={busy} onclick={() => run(() => native.openNotificationSettings())}
				>Open Android notification settings</button
			>
			<label>Allowed Android app package IDs (comma-separated)<input bind:value={packages} /></label
			>
			<button disabled={busy} onclick={() => run(() => saveCapture(true))}
				>Enable and save allowed apps</button
			>
			<button disabled={busy} onclick={() => run(() => saveCapture(false))}
				>Disable and clear unsent suggestions</button
			>
			<button
				class="primary"
				disabled={busy}
				onclick={() =>
					run(async () => {
						const result = await native.sync();
						capture = result;
						await load();
						message = result.rejected
							? String(result.rejected) + ' invalid suggestions were discarded.'
							: 'Sync finished.';
					})}>Sync suggestions</button
			>
		</section>
		<section>
			<h2>Account</h2>
			<button
				disabled={busy}
				onclick={() =>
					run(async () => {
						await native.signOut();
						reset();
					})}>Sign out</button
			>
		</section>
	{/if}
</main>
{#if signedIn}<nav aria-label="Main navigation">
		<button
			aria-current={tab === 'overview' ? 'page' : undefined}
			onclick={() => run(() => switchTab('overview'))}>Overview</button
		>
		<button
			aria-current={tab === 'expenses' ? 'page' : undefined}
			onclick={() => run(() => switchTab('expenses'))}>Expenses</button
		>
		<button
			aria-current={tab === 'settings' ? 'page' : undefined}
			onclick={() => run(() => switchTab('settings'))}>Settings</button
		>
	</nav>{/if}

<style>
	:global(*) {
		box-sizing: border-box;
	}
	:global(body) {
		margin: 0;
		font:
			1rem system-ui,
			sans-serif;
		background: #111;
		color: #f5f5f5;
	}
	header,
	nav {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 0.5rem;
		padding: 1rem;
		background: #171717;
	}
	header {
		padding-top: calc(1rem + env(safe-area-inset-top));
	}
	header strong {
		font-size: 1.35rem;
	}
	header span {
		color: #ffe818;
	}
	main {
		max-width: 42rem;
		margin: auto;
		padding: 1rem 1rem calc(6rem + env(safe-area-inset-bottom));
	}
	section,
	form {
		display: grid;
		gap: 0.8rem;
	}
	section {
		padding: 1rem;
		margin-bottom: 1rem;
		border: 1px solid #363636;
		background: #1c1c1c;
		border-radius: 0.85rem;
	}
	h1 {
		margin: 0.4rem 0 1rem;
	}
	h2,
	h3,
	p,
	dl {
		margin: 0;
	}
	dd {
		font-weight: 650;
		margin: 0.2rem 0 0.6rem;
	}
	dt {
		color: #d5d5d5;
	}
	label {
		display: grid;
		gap: 0.35rem;
	}
	.check {
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}
	button,
	input,
	select {
		color: inherit;
		font: inherit;
		min-height: 2.8rem;
		background: #252525;
		border: 1px solid #575757;
		border-radius: 0.55rem;
		padding: 0.55rem;
	}
	button:disabled {
		opacity: 0.5;
	}
	.primary {
		background: #ffe818;
		color: #111;
		font-weight: 700;
	}
	.alert {
		margin: 0.5rem 0;
		color: #ffdb8e;
	}
	nav {
		position: fixed;
		inset: auto 0 0;
		padding-bottom: calc(0.6rem + env(safe-area-inset-bottom));
	}
	nav button {
		flex: 1;
	}
	nav button[aria-current] {
		color: #ffe818;
	}
</style>
