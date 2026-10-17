<script setup lang="ts">
import { useI18n } from "../composables/useI18n";
import type { NumberStatus } from "../types";

defineProps<{ statuses: NumberStatus[] }>();

const { t } = useI18n();

const colors: Record<NumberStatus, string> = {
	available: "border-zinc-300 bg-white text-zinc-800 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-100",
	reserved:
		"border-amber-400 bg-amber-100 text-amber-900 dark:border-amber-600 dark:bg-amber-950 dark:text-amber-100",
	paid: "border-brand-600 bg-brand-600 text-white dark:border-brand-500 dark:bg-brand-700",
};
</script>

<template>
	<div>
		<ul class="mb-3 flex flex-wrap gap-4 text-sm" aria-hidden="true">
			<li v-for="s in ['available', 'reserved', 'paid'] as NumberStatus[]" :key="s" class="flex items-center gap-2">
				<span class="inline-block size-3 rounded-sm border" :class="colors[s]" />
				{{ t(`raffle.number_status.${s}`) }}
			</li>
		</ul>
		<ol class="grid grid-cols-5 gap-1 sm:grid-cols-10" data-testid="number-grid">
			<li
				v-for="(status, i) in statuses"
				:key="i"
				:data-status="status"
				:title="`${i + 1}: ${t(`raffle.number_status.${status}`)}`"
				class="rounded-md border py-1 text-center text-sm tabular-nums"
				:class="colors[status]"
			>
				<span class="sr-only">{{ t(`raffle.number_status.${status}`) }}: </span>{{ i + 1 }}
			</li>
		</ol>
	</div>
</template>
