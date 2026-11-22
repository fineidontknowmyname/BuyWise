<script lang="ts">
	import { resolve } from '$app/paths';
</script>

<svelte:head>
	<meta http-equiv="refresh" content="0;url=/budget" />
	<title>Redirecting · BuyWise</title>
</svelte:head>

<div class="card p-6 text-center">
	<p>
		Redirecting to <a href={resolve('/budget')} class="text-primary font-semibold"
			>Budget & Categories</a
		>...
	</p>
</div>
