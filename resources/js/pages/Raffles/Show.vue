<script setup lang="ts">
import { Head, Link, router } from "@inertiajs/vue3";
import { computed, ref } from "vue";

import BuyerCard from "../../components/BuyerCard.vue";
import DrawResult from "../../components/DrawResult.vue";
import LinkForm from "../../components/LinkForm.vue";
import LinkResult from "../../components/LinkResult.vue";
import NumberGrid from "../../components/NumberGrid.vue";
import { useI18n } from "../../composables/useI18n";
import { formatDateTime, formatReais } from "../../lib/format";
import type { Buyer, DrawResult as Draw, OwnedRaffle, PaymentLink, RaffleNumber } from "../../types";

const props = defineProps<{
	raffle: OwnedRaffle;
	numbers: RaffleNumber[];
	buyers: Buyer[];
	draw: Draw | null;
	winner: Buyer | null;
}>();

const { t, locale } = useI18n();
const statuses = computed(() => props.numbers.map((n) => n.status));
const paid = computed(() => props.numbers.filter((n) => n.status === "paid").map((n) => n.number));
const finished = computed(() => props.raffle.status !== "active");
const onSale = computed(() => props.raffle.status === "active" && Date.parse(props.raffle.ends_at) > Date.now());
const button =
	"rounded-md border border-zinc-300 px-3 py-1 text-sm hover:bg-zinc-100 dark:border-zinc-700 dark:hover:bg-zinc-800";

const selected = ref<number | null>(null);
// The new link lives only in this component's state: it is never stored
// server-side and disappears on reload, by design.
const created = ref<PaymentLink | null>(null);

const selectedStatus = computed(() => (selected.value ? props.numbers[selected.value - 1]?.status : undefined));
const selectedBuyer = computed(() => props.buyers.find((b) => b.number === selected.value) ?? null);

function select(n: number): void {
	created.value = null;
	selected.value = selected.value === n ? null : n;
}

function close(): void {
	selected.value = null;
	created.value = null;
}

function onCreated(link: PaymentLink): void {
	created.value = link;
	router.reload({ only: ["numbers", "buyers"] });
}

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
	<DrawResult v-if="draw" class="mb-6" :draw="draw" :paid="paid">
		<div v-if="winner" class="mb-4 rounded-md bg-brand-50 p-3 text-sm dark:bg-zinc-900" data-testid="draw-winner-contact">
			<p class="font-medium">{{ t("draw.winner") }}</p>
			<p>{{ winner.buyer_name }} · {{ winner.buyer_email }} · {{ winner.buyer_phone }}</p>
		</div>
		<p v-else-if="draw.winning_number !== null" class="mb-4 text-sm text-zinc-600 dark:text-zinc-400">
			{{ t("draw.winner_missing") }}
		</p>
	</DrawResult>
	<h2 class="mb-2 text-lg font-semibold">{{ t("raffle.numbers") }}</h2>
	<p v-if="onSale" class="mb-3 text-sm text-zinc-600 dark:text-zinc-400">{{ t("link.grid_hint") }}</p>
	<div class="grid gap-6 lg:grid-cols-[1fr_22rem]">
		<NumberGrid :statuses="statuses" interactive :selected="selected" @select="select" />
		<aside
			v-if="selected"
			class="h-fit rounded-lg border border-zinc-200 bg-white p-4 shadow-sm lg:sticky lg:top-4 dark:border-zinc-800 dark:bg-zinc-950"
			data-testid="number-panel"
		>
			<LinkResult v-if="created" :link="created" :raffle-title="raffle.title" @close="close" />
			<BuyerCard
				v-else-if="selectedBuyer"
				:raffle-id="raffle.id"
				:buyer="selectedBuyer"
				:cancellable="raffle.status === 'active'"
				@close="close"
			/>
			<LinkForm
				v-else-if="selectedStatus === 'available' && onSale"
				:key="selected"
				:raffle-id="raffle.id"
				:number="selected"
				@created="onCreated"
				@close="close"
			/>
			<p v-else class="text-sm text-zinc-600 dark:text-zinc-400">
				{{ t("link.details.title", { number: selected }) }} · {{ t(`raffle.number_status.${selectedStatus ?? "available"}`) }}
			</p>
		</aside>
	</div>
</template>
