<script setup lang="ts">
import { Head } from "@inertiajs/vue3";

import NumberGrid from "../../components/NumberGrid.vue";
import { useI18n } from "../../composables/useI18n";
import { formatDateTime, formatReais } from "../../lib/format";
import type { NumberStatus, PublicRaffle } from "../../types";

defineProps<{ raffle: PublicRaffle; statuses: NumberStatus[] }>();

const { t, locale } = useI18n();
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
		<h2 class="mt-8 mb-2 text-lg font-semibold">{{ t("raffle.numbers") }}</h2>
		<NumberGrid :statuses="statuses" />
		<p class="mt-6 text-xs text-zinc-500">{{ t("raffle.public_footer") }}</p>
	</article>
</template>
