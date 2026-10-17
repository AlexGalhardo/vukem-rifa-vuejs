<script setup lang="ts">
import { Head } from "@inertiajs/vue3";
import { computed } from "vue";

import DrawResult from "../../components/DrawResult.vue";
import NumberGrid from "../../components/NumberGrid.vue";
import { useI18n } from "../../composables/useI18n";
import { formatDateTime, formatReais } from "../../lib/format";
import type { DrawResult as Draw, NumberStatus, PublicRaffle } from "../../types";

const props = defineProps<{ raffle: PublicRaffle; statuses: NumberStatus[]; draw: Draw | null }>();

const { t, locale } = useI18n();
// After the draw no number can change, so the paid ones are the participants.
const paid = computed(() => props.statuses.flatMap((s, i) => (s === "paid" ? [i + 1] : [])));
</script>

<template>
	<Head :title="raffle.title" />
	<article>
		<h1 class="text-3xl font-semibold" data-testid="public-title">{{ raffle.title }}</h1>
		<p class="mt-2 text-zinc-600 dark:text-zinc-400">
			{{ t(`raffle.status.${raffle.status}`) }}
			· {{ t("raffle.price", { price: formatReais(raffle.price_reais, locale) }) }}
			· {{ t("raffle.ends", { date: formatDateTime(raffle.ends_at, locale) }) }}
		</p>
		<p v-if="raffle.description" class="mt-4 whitespace-pre-line">{{ raffle.description }}</p>
		<DrawResult v-if="draw" class="mt-8" :draw="draw" :paid="paid" />
		<p v-else-if="raffle.status === 'archived'" class="mt-6 text-sm text-zinc-600 dark:text-zinc-400">
			{{ t("draw.archived") }}
		</p>
		<p v-else class="mt-6 text-sm text-zinc-600 dark:text-zinc-400" data-testid="draw-pending">{{ t("draw.pending") }}</p>
		<h2 class="mt-8 mb-2 text-lg font-semibold">{{ t("raffle.numbers") }}</h2>
		<NumberGrid :statuses="statuses" />
		<p class="mt-6 text-xs text-zinc-600 dark:text-zinc-400">{{ t("raffle.public_footer") }}</p>
	</article>
</template>
