<script setup lang="ts">
import { Head, Link, router } from "@inertiajs/vue3";
import { computed } from "vue";

import ExportLinks from "../../components/ExportLinks.vue";
import NumberGrid from "../../components/NumberGrid.vue";
import { useI18n } from "../../composables/useI18n";
import { formatDateTime, formatReais } from "../../lib/format";
import type { OwnedRaffle, RaffleNumber } from "../../types";

const props = defineProps<{ raffle: OwnedRaffle; numbers: RaffleNumber[] }>();

const { t, locale } = useI18n();
const statuses = computed(() => props.numbers.map((n) => n.status));
const finished = computed(() => props.raffle.status === "archived" || props.raffle.status === "drawn");
const button =
	"rounded-md border border-zinc-300 px-3 py-1 text-sm hover:bg-zinc-100 dark:border-zinc-700 dark:hover:bg-zinc-800";

function destroy(): void {
	if (window.confirm(t("raffle.actions.confirm_delete"))) router.delete(`/raffles/${props.raffle.id}`);
}

function archive(): void {
	if (window.confirm(t("raffle.actions.confirm_archive"))) router.post(`/raffles/${props.raffle.id}/archive`);
}
</script>

<template>
	<Head :title="raffle.title" />
	<div class="mb-6 flex flex-wrap items-start justify-between gap-4">
		<div>
			<h1 class="text-2xl font-semibold" data-testid="raffle-heading">{{ raffle.title }}</h1>
			<p class="mt-1 text-sm text-zinc-600 dark:text-zinc-400">
				<span data-testid="raffle-status">{{ t(`raffle.status.${raffle.status}`) }}</span>
				· {{ t("raffle.price", { price: formatReais(raffle.price_reais, locale) }) }}
				· {{ t("raffle.ends", { date: formatDateTime(raffle.ends_at, locale) }) }}
			</p>
		</div>
		<div class="flex flex-wrap gap-2">
			<a :href="`/r/${raffle.slug}`" :class="button" data-testid="public-link">{{ t("raffle.actions.public_page") }}</a>
			<ExportLinks :raffle-id="raffle.id" />
			<Link v-if="!finished" :href="`/raffles/${raffle.id}/edit`" :class="button">{{ t("raffle.actions.edit") }}</Link>
			<button v-if="!raffle.locked" type="button" :class="button" data-testid="raffle-delete" @click="destroy">
				{{ t("raffle.actions.delete") }}
			</button>
			<button
				v-else-if="raffle.status !== 'archived'"
				type="button"
				:class="button"
				data-testid="raffle-archive"
				@click="archive"
			>
				{{ t("raffle.actions.archive") }}
			</button>
		</div>
	</div>
	<p v-if="raffle.description" class="mb-6 whitespace-pre-line">{{ raffle.description }}</p>
	<h2 class="mb-2 text-lg font-semibold">{{ t("raffle.numbers") }}</h2>
	<NumberGrid :statuses="statuses" />
</template>
