<script setup lang="ts">
import { useToast } from "../composables/useToast";

const { toasts, dismiss } = useToast();
</script>

<template>
	<div class="pointer-events-none fixed inset-x-0 bottom-4 z-50 flex flex-col items-center gap-2 px-4">
		<div
			v-for="toast in toasts"
			:key="toast.id"
			:role="toast.kind === 'error' ? 'alert' : 'status'"
			:aria-live="toast.kind === 'error' ? 'assertive' : 'polite'"
			data-testid="toast"
			:class="[
				'pointer-events-auto flex max-w-md items-center gap-3 rounded-lg px-4 py-3 text-sm font-medium shadow-lg',
				toast.kind === 'error'
					? 'bg-red-700 text-white dark:bg-red-800'
					: 'bg-zinc-900 text-white dark:bg-zinc-100 dark:text-zinc-900',
			]"
			@click="dismiss(toast.id)"
		>
			{{ toast.message }}
		</div>
	</div>
</template>
