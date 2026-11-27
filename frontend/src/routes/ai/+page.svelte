<script lang="ts">
	import { onDestroy } from 'svelte';
	import {
		Sparkles,
		ArrowUp,
		ArrowUpRight,
		Wallet,
		Target,
		ShoppingBag,
		ChartNoAxesCombined,
		CircleCheck,
		Trash2,
		LoaderCircle
	} from '@lucide/svelte';
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
	const prompts = {
		capacity: { title: 'Monthly savings', detail: 'What can I set aside?', icon: Wallet },
		goal: { title: 'My goals', detail: 'Am I on track?', icon: Target },
		decision: {
			title: 'Buying decisions',
			detail: 'Is this purchase within reach?',
			icon: CircleCheck
		},
		price: { title: 'Product prices', detail: 'How does this price compare?', icon: ShoppingBag },
		budget: { title: 'My spending', detail: 'Where is my money going?', icon: ChartNoAxesCombined }
	};
	let topic = $state<AssistantTopic>('capacity');
	let recordId = $state('');
	let message = $state('');
	let pending = $state(false);
	let failure = $state('');
	let history = $state<{ id: string; question: string; answer: AssistantAnswer }[]>([]);
	let current: AbortController | undefined;
	const needsRecord = $derived(topic === 'goal' || topic === 'decision' || topic === 'price');
	const options = $derived(topic === 'price' ? data.products : data.goals);
	onDestroy(() => current?.abort());

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

	async function explain(event: SubmitEvent, freeText = false) {
		event.preventDefault();
		const text = message.trim();
		if (pending || (freeText ? !text : needsRecord && !recordId)) return;
		const sources = history[0]?.answer.sources ?? [];
		const goal = sources.find((source) => source.kind === 'goal');
		const product = sources.find((source) => source.kind === 'product');
		const controller = new AbortController();
		current = controller;
		const selected = topic;
		pending = true;
		failure = '';
		try {
			const answer = await askAssistant(
				freeText
					? {
							message: text,
							previous: history[0]?.question,
							goalId: goal && 'id' in goal ? goal.id : undefined,
							productId: product && 'id' in product ? product.id : undefined
						}
					: selected === 'price'
						? { topic: selected, productId: recordId }
						: needsRecord
							? { topic: selected, goalId: recordId }
							: { topic: selected },
				controller.signal
			);
			if (!controller.signal.aborted) {
				history = [
					{
						id: crypto.randomUUID(),
						question: freeText ? text : assistantQuestions[selected],
						answer
					},
					...history
				].slice(0, 8);
				if (freeText) message = '';
			}
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
	<header class="page-heading">
		<div class="heading-copy">
			<div class="eyebrow"><Sparkles size={15} aria-hidden="true" /> BuyWise Assist</div>
			<h1>Financial Assistant</h1>
			<p>A clearer view of your money, and your next move.</p>
		</div>
		{#if history.length}<button class="btn btn-secondary clear-button" type="button" onclick={clear}
				><Trash2 size={15} aria-hidden="true" /> Clear conversation</button
			>{/if}
	</header>

	<section class="composer-card" aria-labelledby="question-heading">
		<div class="composer-heading">
			<span class="assistant-icon"><Sparkles size={21} aria-hidden="true" /></span>
			<div>
				<h2 id="question-heading">What would you like to work out?</h2>
				<p>Ask about your savings, spending or next purchase.</p>
			</div>
		</div>
		<form class="question-form" onsubmit={(event) => explain(event, true)}>
			<label class="sr-only" for="assistant-message">Your question</label>
			<textarea
				id="assistant-message"
				bind:value={message}
				maxlength="1000"
				rows="3"
				placeholder="How can I reach my next goal sooner?"
				disabled={pending}
				onkeydown={(event) => {
					if (event.key === 'Enter' && (event.ctrlKey || event.metaKey)) {
						event.preventDefault();
						event.currentTarget.form?.requestSubmit();
					}
				}}></textarea>
			<div class="composer-footer">
				<span
					>{message.length ? `${message.length}/1,000` : 'Your question, in your own words'}</span
				><button
					class="send-button"
					aria-label={pending ? 'Preparing your answer' : 'Send question'}
					disabled={pending || !message.trim()}
					>{#if pending}<LoaderCircle class="spin" size={18} aria-hidden="true" />{:else}<ArrowUp
							size={19}
							aria-hidden="true"
						/>{/if}</button
				>
			</div>
		</form>
	</section>

	<section class="suggestions" aria-labelledby="suggestions-heading">
		<div class="section-heading">
			<h2 id="suggestions-heading">Start with a question</h2>
			<span>Based on your BuyWise records</span>
		</div>
		<div class="prompt-grid">
			{#each topics as item (item)}{@const Icon = prompts[item].icon}<button
					class="prompt-card"
					class:selected={topic === item}
					type="button"
					aria-pressed={topic === item}
					disabled={pending}
					onclick={() => choose(item)}
					><span class="prompt-top"
						><Icon size={18} aria-hidden="true" /><ArrowUpRight
							size={14}
							aria-hidden="true"
						/></span
					><strong>{prompts[item].title}</strong><span>{prompts[item].detail}</span></button
				>{/each}
		</div>
		<form class="record-picker" onsubmit={(event) => explain(event)}>
			<div class="selected-question">
				<span class="selection-label">Explore</span>
				<p>{assistantQuestions[topic]}</p>
			</div>
			{#if needsRecord}
				{#if options === null}<p class="inline-status" role="alert">
						Your records couldn't load. Refresh to try again.
					</p>
				{:else if !options.length}<p class="inline-status">
						{#if topic === 'price'}<a href={resolve('/products')}>Add a product →</a> to explore its price.{:else}<a
								href={resolve('/goals')}>Create a goal →</a
							> to explore your plan.{/if}
					</p>
				{:else}<label class="record-label"
						><span>{topic === 'price' ? 'Product' : 'Purchase goal'}</span><select
							bind:value={recordId}
							required
							disabled={pending}
							><option value="">{topic === 'price' ? 'Choose a product' : 'Choose a goal'}</option
							>{#each options as item (item.id)}<option value={item.id}>{item.name}</option
								>{/each}</select
						></label
					>{/if}
			{/if}
			<button
				class="btn btn-primary explore-button"
				disabled={pending || (needsRecord && !recordId)}
				>Explore <ArrowUpRight size={16} aria-hidden="true" /></button
			>
		</form>
	</section>

	{#if failure}<div class="error-notice" role="alert">
			<strong>Couldn't get an answer</strong>
			<p>{failure}</p>
		</div>{/if}
	<div class="request-status" role="status" aria-live="polite">
		{#if pending}<LoaderCircle class="spin" size={16} aria-hidden="true" /> Reviewing your saved records…{/if}
	</div>
	{#if history.length}<section class="answers" aria-label="Conversation" aria-busy={pending}>
			<div class="section-heading">
				<h2>Your conversation</h2>
				<span>{history.length} {history.length === 1 ? 'answer' : 'answers'}</span>
			</div>
			{#each history as item (item.id)}<article class="conversation-turn">
					<div class="question-bubble">
						<span>You asked</span>
						<p>{item.question}</p>
					</div>
					<div class="answer-card">
						<div class="answer-label"><Sparkles size={14} aria-hidden="true" /> BuyWise</div>
						<AssistantEvidence answer={item.answer} />
					</div>
				</article>{/each}
		</section>{/if}
</div>

<style>
	.assistant-page {
		width: 100%;
		max-width: 1100px;
		margin-inline: auto;
		display: grid;
		gap: 26px;
		font-size: 14px;
		line-height: 1.6;
		min-width: 0;
	}
	.page-heading,
	.section-heading {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 16px;
		flex-wrap: wrap;
	}
	.eyebrow,
	.answer-label {
		display: flex;
		align-items: center;
		gap: 7px;
		font-size: 12px;
		font-weight: 600;
		color: var(--foreground-secondary);
	}
	h1,
	h2,
	p {
		margin: 0;
	}
	.heading-copy h1 {
		margin: 8px 0;
		font-size: clamp(28px, 3vw, 36px);
		line-height: 1.2;
		font-weight: 650;
		letter-spacing: -0.04em;
	}
	.heading-copy > p,
	.composer-heading p {
		color: var(--foreground-secondary);
		font-size: 14px;
	}
	.clear-button {
		font-size: 12px;
		min-height: 38px;
	}
	.composer-card {
		padding: 28px;
		border: 1px solid var(--border);
		border-radius: 22px;
		background: var(--surface);
		box-shadow: 0 8px 32px rgb(0 0 0 / 0.03);
	}
	.composer-heading {
		display: flex;
		align-items: center;
		gap: 14px;
		margin-bottom: 22px;
	}
	.assistant-icon {
		display: grid;
		place-items: center;
		width: 46px;
		height: 46px;
		flex-shrink: 0;
		border-radius: 14px;
		background: var(--background);
		border: 1px solid var(--border);
		color: var(--brand-dot);
	}
	.composer-heading h2 {
		font-size: 18px;
		font-weight: 600;
		letter-spacing: -0.025em;
		margin-bottom: 3px;
	}
	.question-form {
		border: 1px solid var(--border);
		border-radius: 16px;
		background: var(--background);
		overflow: hidden;
		transition: border-color 0.15s;
	}
	.question-form:focus-within {
		border-color: var(--foreground-secondary);
		outline: 2px solid color-mix(in srgb, var(--brand-dot) 28%, transparent);
		outline-offset: 3px;
	}
	textarea {
		display: block;
		width: 100%;
		min-height: 108px;
		max-height: 280px;
		resize: vertical;
		padding: 18px 18px 8px;
		border: 0;
		background: transparent;
		color: var(--foreground);
		font: inherit;
		line-height: 1.65;
		box-shadow: none;
		outline: none;
		border-radius: 0;
	}
	textarea::placeholder {
		color: var(--foreground-muted);
	}
	.composer-footer {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 12px;
		padding: 10px 14px 14px;
	}
	.composer-footer > span {
		font-size: 11px;
		color: var(--foreground-muted);
	}
	.send-button {
		width: 38px;
		height: 38px;
		display: grid;
		place-items: center;
		flex-shrink: 0;
		border: 0;
		border-radius: 11px;
		color: #111;
		background: var(--accent);
		cursor: pointer;
	}
	.send-button:disabled {
		opacity: 0.4;
		cursor: default;
	}
	.suggestions,
	.answers {
		display: grid;
		gap: 15px;
		min-width: 0;
	}
	.section-heading h2 {
		font-size: 14px;
		font-weight: 600;
	}
	.section-heading > span {
		font-size: 12px;
		color: var(--foreground-muted);
	}
	.prompt-grid {
		display: grid;
		grid-template-columns: repeat(5, minmax(0, 1fr));
		gap: 12px;
	}
	.prompt-card {
		display: grid;
		gap: 7px;
		align-content: start;
		text-align: left;
		padding: 17px 16px;
		min-width: 0;
		min-height: 142px;
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: 16px;
		color: var(--foreground);
		font: inherit;
		cursor: pointer;
		transition:
			border-color 0.15s,
			background 0.15s;
	}
	.prompt-card:hover:not(:disabled) {
		border-color: var(--foreground-secondary);
	}
	.prompt-card.selected {
		border-color: var(--foreground-secondary);
		background: color-mix(in srgb, var(--brand-dot) 5%, var(--surface));
	}
	.prompt-top {
		display: flex;
		align-items: center;
		justify-content: space-between;
		margin-bottom: 8px;
		color: var(--foreground-secondary);
	}
	.prompt-card.selected .prompt-top {
		color: var(--foreground);
	}
	.prompt-card strong {
		font-size: 13px;
		font-weight: 600;
	}
	.prompt-card > span:last-child {
		font-size: 12px;
		color: var(--foreground-secondary);
		line-height: 1.5;
	}
	.prompt-card:disabled {
		cursor: default;
		opacity: 0.65;
	}
	.record-picker {
		display: flex;
		flex-wrap: wrap;
		gap: 16px;
		align-items: center;
		padding: 18px 20px;
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: 16px;
	}
	.selected-question {
		flex: 1 1 260px;
	}
	.selection-label {
		font-size: 10px;
		font-weight: 650;
		letter-spacing: 0.09em;
		text-transform: uppercase;
		color: var(--foreground-muted);
	}
	.selected-question p {
		font-size: 13px;
		margin-top: 3px;
	}
	.record-label {
		display: grid;
		gap: 4px;
		flex: 0 1 250px;
		min-width: 0;
	}
	.record-label > span {
		font-size: 11px;
		color: var(--foreground-secondary);
	}
	select {
		width: 100%;
		min-width: 0;
		height: 42px;
		border: 1px solid var(--border);
		border-radius: 10px;
		background: var(--background);
		color: var(--foreground);
		padding: 8px 10px;
		font: inherit;
		font-size: 13px;
	}
	.explore-button {
		min-height: 40px;
		font-size: 13px;
	}
	.inline-status {
		font-size: 12px;
		color: var(--foreground-secondary);
		flex: 1 1 200px;
	}
	.request-status:empty {
		display: none;
	}
	.request-status {
		display: flex;
		align-items: center;
		gap: 9px;
		color: var(--foreground-secondary);
		font-size: 13px;
	}
	.error-notice {
		border: 1px solid var(--danger);
		background: var(--surface);
		color: var(--foreground);
		padding: 16px 20px;
		border-radius: 14px;
	}
	.error-notice strong {
		font-size: 13px;
	}
	.error-notice p {
		font-size: 12px;
		color: var(--foreground-secondary);
		margin-top: 4px;
	}
	.conversation-turn {
		display: grid;
		gap: 12px;
		min-width: 0;
	}
	.question-bubble {
		justify-self: end;
		max-width: min(90%, 700px);
		padding: 12px 17px;
		border: 1px solid var(--border);
		background: var(--surface-secondary);
		border-radius: 16px 16px 4px 16px;
		overflow-wrap: anywhere;
	}
	.question-bubble > span {
		font-size: 10px;
		color: var(--foreground-muted);
	}
	.question-bubble p {
		font-size: 13px;
	}
	.answer-card {
		padding: 24px;
		border-radius: 18px;
		border: 1px solid var(--border);
		background: var(--surface);
		display: grid;
		gap: 14px;
		min-width: 0;
	}
	.answer-label {
		font-size: 11px;
	}
	button:focus-visible,
	select:focus-visible {
		outline: 2px solid var(--brand-dot);
		outline-offset: 3px;
	}
	.sr-only {
		position: absolute;
		width: 1px;
		height: 1px;
		padding: 0;
		margin: -1px;
		overflow: hidden;
		clip-path: inset(50%);
		white-space: nowrap;
		border: 0;
	}
	.assistant-page :global(.spin) {
		animation: turn 1s linear infinite;
	}
	@keyframes turn {
		to {
			transform: rotate(360deg);
		}
	}
	@media (max-width: 900px) {
		.prompt-grid {
			grid-template-columns: repeat(3, minmax(0, 1fr));
		}
		.prompt-card {
			min-height: 126px;
		}
	}
	@media (max-width: 600px) {
		.assistant-page {
			gap: 20px;
		}
		.composer-card {
			padding: 20px;
		}
		.composer-heading {
			align-items: flex-start;
			gap: 10px;
		}
		.composer-heading h2 {
			font-size: 16px;
		}
		.composer-heading p {
			font-size: 12px;
		}
		.assistant-icon {
			width: 38px;
			height: 38px;
			border-radius: 11px;
		}
		.prompt-grid {
			grid-template-columns: repeat(2, minmax(0, 1fr));
			gap: 10px;
		}
		.prompt-card {
			padding: 14px;
			min-height: 126px;
		}
		.section-heading > span {
			font-size: 11px;
		}
		.record-picker {
			padding: 16px;
		}
		.selected-question,
		.record-label {
			flex-basis: 100%;
		}
		.explore-button {
			width: 100%;
		}
		.answer-card {
			padding: 18px;
		}
		.question-bubble {
			max-width: 95%;
		}
	}
	@media (prefers-reduced-motion: reduce) {
		.assistant-page :global(.spin) {
			animation: none;
		}
		.prompt-card,
		.question-form {
			transition: none;
		}
	}
</style>
