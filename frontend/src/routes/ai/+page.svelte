<script lang="ts">
	import { resolve } from '$app/paths';
	import AssistantEvidence from '#lib/components/AssistantEvidence.svelte';
	import {
		askAssistant,
		assistantQuestions,
		type AssistantTopic,
		type AssistantAnswer
	} from '#lib/assistant.js';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	const topics = Object.keys(assistantQuestions) as AssistantTopic[];
	let topic = $state<AssistantTopic>('capacity');
	let recordId = $state('');
	let pending = $state(false);
	let failure = $state('');
	let history = $state<{ id: string; topic: AssistantTopic; answer: AssistantAnswer }[]>([]);
	let current: AbortController | undefined;
	const needsRecord = $derived(topic === 'goal' || topic === 'decision' || topic === 'price');
	const options = $derived(topic === 'price' ? data.products : data.goals);

	function choose(next: AssistantTopic) {
		if ((next === 'price') !== (topic === 'price') || !needsRecord) recordId = '';
		topic = next;
		failure = '';
	}

	function clear() {
		current?.abort();
		current = undefined;
		pending = false;
		failure = '';
		history = [];
	}

	async function explain(event: SubmitEvent) {
		event.preventDefault();
		if (pending || (needsRecord && !recordId)) return;
		const controller = new AbortController();
		current = controller;
		const selected = topic;
		pending = true;
		failure = '';
		try {
			const answer = await askAssistant(
				selected === 'price'
					? { topic: selected, productId: recordId }
					: needsRecord
						? { topic: selected, goalId: recordId }
						: { topic: selected },
				controller.signal
			);
			if (!controller.signal.aborted)
				history = [{ id: crypto.randomUUID(), topic: selected, answer }, ...history].slice(0, 8);
		} catch (err) {
			if (!controller.signal.aborted)
				failure = err instanceof Error ? err.message : 'The assistant is temporarily unavailable.';
		} finally {
			if (current === controller) {
				current = undefined;
				pending = false;
			}
		}
	}
</script>

<svelte:head><title>Financial Assistant | BuyWise</title></svelte:head>

<div class="assistant-page">
	<header>
		<h1>Financial Assistant</h1>
		<p>Ask about your saved goals, monthly capacity, prices, and purchase decisions.</p>
	</header>
	<section class="card assistant-card" aria-labelledby="questions-heading">
		<h2 id="questions-heading">Choose a question</h2>
		<div class="questions">
			{#each topics as item (item)}<button
					type="button"
					class="btn"
					class:btn-primary={topic === item}
					class:btn-secondary={topic !== item}
					aria-pressed={topic === item}
					disabled={pending}
					onclick={() => choose(item)}>{assistantQuestions[item]}</button
				>{/each}
		</div>
		<form onsubmit={explain}>
			{#if needsRecord}
				{#if options === null}<p role="alert">
						The record list couldn’t load. Reload to try again.
					</p>
				{:else if !options.length}
					{#if topic === 'price'}<p>
							<a href={resolve('/products')}>Add a tracked product</a> to ask about its prices.
						</p>
					{:else}<p><a href={resolve('/goals')}>Create a purchase goal</a> to ask about it.</p>{/if}
				{:else}
					<label
						>{topic === 'price' ? 'Tracked product' : 'Purchase goal'}
						<select class="input" bind:value={recordId} required disabled={pending}>
							<option value="">Choose a record</option>
							{#each options as item (item.id)}<option value={item.id}>{item.name}</option>{/each}
						</select>
					</label>
				{/if}
			{/if}
			<div>
				<button class="btn btn-primary" disabled={pending || (needsRecord && !recordId)}
					>{pending ? 'Reading records…' : 'Explain'}</button
				>
			</div>
		</form>
		{#if failure}<p role="alert">{failure}</p>{/if}
		<p class="muted">
			This assistant reads saved records. This page keeps up to eight answers until you reload or
			clear them.
		</p>
	</section>
	{#if history.length}
		<section class="answers" aria-label="Answers">
			<div class="answers-header">
				<h2>Answers</h2>
				<button type="button" class="btn btn-secondary" onclick={clear}>Clear answers</button>
			</div>
			{#each history as item (item.id)}
				<article class="card assistant-card">
					<p><strong>{assistantQuestions[item.topic]}</strong></p>
					<AssistantEvidence answer={item.answer} />
				</article>
			{/each}
		</section>
	{/if}
</div>

<style>
	.assistant-page,
	.answers {
		display: grid;
		gap: 1.15rem;
		max-width: 66rem;
	}
	.assistant-card {
		display: grid;
		gap: 0.9rem;
		padding: clamp(1rem, 3vw, 1.5rem);
	}
	.questions,
	.answers-header {
		display: flex;
		flex-wrap: wrap;
		gap: 0.65rem;
		align-items: center;
	}
	.answers-header {
		justify-content: space-between;
	}
	.questions button {
		white-space: normal;
		text-align: left;
	}
	form,
	label {
		display: grid;
		gap: 0.65rem;
	}
	select {
		max-width: 28rem;
	}
	h1,
	h2,
	p {
		margin: 0;
	}
	header p {
		margin-top: 0.5rem;
	}
	.muted {
		color: var(--text-secondary);
	}
	[role='alert'] {
		color: var(--danger);
	}
</style>
