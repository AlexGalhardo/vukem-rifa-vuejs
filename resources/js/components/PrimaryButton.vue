<script setup lang="ts">
defineProps<{ disabled?: boolean }>();
</script>

<template>
	<button
		type="submit"
		:disabled="disabled"
		class="rounded-md bg-brand-600 px-4 py-2 font-medium text-white hover:bg-brand-700 disabled:opacity-50"
	>
		<slot />
	</button>
</template>
