<script setup lang="ts">
import { useI18n } from "../composables/useI18n";
import type { NumberStatus } from "../types";

// numbers[i] is number i+1. With interactive, every number is a button
// that emits select (the owner's manage page); without it, a plain list
// (the public page).
// onlyAvailable (public page): only free numbers are buttons; reserved and
// paid ones stay plain, so a buyer can never pick them.
withDefaults(
	defineProps<{
		statuses: NumberStatus[];
		interactive?: boolean;
		selected?: number | null;
		onlyAvailable?: boolean;
	}>(),
	{ interactive: false, selected: null, onlyAvailable: false },
);
const emit = defineEmits<{ select: [number: number] }>();

const { t } = useI18n();

const colors: Record<NumberStatus, string> = {
	available: "border-zinc-300 bg-white text-zinc-800 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-100",
	reserved:
		"border-amber-400 bg-amber-100 text-amber-900 dark:border-amber-600 dark:bg-amber-950 dark:text-amber-100",
	paid: "border-brand-600 bg-brand-600 text-white dark:border-brand-500 dark:bg-brand-700",
};
const hover: Record<NumberStatus, string> = {
	available: "hover:border-brand-500 hover:bg-brand-50 dark:hover:border-brand-500 dark:hover:bg-zinc-800",
	reserved: "hover:bg-amber-200 dark:hover:bg-amber-900",
	paid: "hover:bg-brand-700 dark:hover:bg-brand-600",
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
				:data-number="i + 1"
				:title="`${i + 1}: ${t(`raffle.number_status.${status}`)}`"
				class="text-center text-sm tabular-nums"
				:class="interactive && (!onlyAvailable || status === 'available') ? '' : ['rounded-md border py-1', colors[status]]"
			>
				<button
					v-if="interactive && (!onlyAvailable || status === 'available')"
					type="button"
					class="w-full rounded-md border py-1 focus:outline-none focus-visible:ring-2 focus-visible:ring-brand-500"
					:class="[colors[status], hover[status], selected === i + 1 ? 'ring-2 ring-brand-500 dark:ring-brand-500' : '']"
					:aria-pressed="selected === i + 1"
					:data-testid="`number-${i + 1}`"
					@click="emit('select', i + 1)"
				>
					<span class="sr-only">{{ t(`raffle.number_status.${status}`) }}: </span>{{ i + 1 }}
				</button>
				<template v-else>
					<span class="sr-only">{{ t(`raffle.number_status.${status}`) }}: </span>{{ i + 1 }}
				</template>
			</li>
		</ol>
	</div>
</template>
