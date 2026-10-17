<script setup lang="ts">
import { Head, usePage } from "@inertiajs/vue3";
import { computed, ref } from "vue";

import DrawResult from "../../components/DrawResult.vue";
import NumberGrid from "../../components/NumberGrid.vue";
import OrganizerCard from "../../components/OrganizerCard.vue";
import PublicBuyForm from "../../components/PublicBuyForm.vue";
import { useI18n } from "../../composables/useI18n";
import { formatDateTime, formatReais } from "../../lib/format";
import type { DrawResult as Draw, NumberStatus, Organizer, PublicRaffle, SharedProps } from "../../types";

const props = defineProps<{
	raffle: PublicRaffle;
	statuses: NumberStatus[];
	draw: Draw | null;
	can_buy: boolean;
	organizer: Organizer;
}>();

const { t, locale } = useI18n();
// After the draw no number can change, so the paid ones are the participants.
const paid = computed(() => props.statuses.flatMap((s, i) => (s === "paid" ? [i + 1] : [])));

const page = usePage<SharedProps>();
const user = computed(() => page.props.auth?.user ?? null);
const selected = ref<number | null>(null);

function select(n: number): void {
	selected.value = selected.value === n ? null : n;
}
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
		<OrganizerCard class="mt-6" :organizer="organizer" :slug="raffle.slug" :created-at="raffle.created_at" />
		<DrawResult v-if="draw" class="mt-8" :draw="draw" :paid="paid" />
		<p v-else-if="raffle.status === 'archived'" class="mt-6 text-sm text-zinc-600 dark:text-zinc-400">
			{{ t("draw.archived") }}
		</p>
		<p v-else class="mt-6 text-sm text-zinc-600 dark:text-zinc-400" data-testid="draw-pending">{{ t("draw.pending") }}</p>
		<h2 class="mt-8 mb-2 text-lg font-semibold">{{ t("raffle.numbers") }}</h2>
		<p v-if="can_buy" class="mb-3 text-sm text-zinc-600 dark:text-zinc-400">{{ t("link.public.hint") }}</p>
		<div class="grid gap-6 lg:grid-cols-[1fr_22rem]">
			<NumberGrid
				:statuses="statuses"
				:interactive="can_buy"
				only-available
				:selected="selected"
				@select="select"
			/>
			<aside
				v-if="can_buy && selected"
				class="h-fit rounded-lg border border-zinc-200 bg-white p-4 shadow-sm lg:sticky lg:top-20 dark:border-zinc-800 dark:bg-zinc-950"
				data-testid="public-buy-panel"
			>
				<PublicBuyForm
					:key="selected"
					:slug="raffle.slug"
					:number="selected"
					:price-reais="raffle.price_reais"
					:prefill="user ? { name: user.name, email: user.email } : undefined"
					@close="selected = null"
				/>
			</aside>
		</div>
		<p class="mt-6 text-xs text-zinc-600 dark:text-zinc-400">{{ t("raffle.public_footer") }}</p>
	</article>
</template>
