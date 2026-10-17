<script setup lang="ts">
import { Head, Link, usePage } from "@inertiajs/vue3";
import { computed } from "vue";

import BalanceCard from "../components/BalanceCard.vue";
import { useI18n } from "../composables/useI18n";
import { formatDateTime, formatReais } from "../lib/format";
import type { RaffleSummary, SharedProps } from "../types";

defineProps<{ raffles: RaffleSummary[]; balance_cents: number }>();

const { t, locale } = useI18n();
const page = usePage<SharedProps>();
const name = computed(() => page.props.auth?.user?.name ?? "");
</script>

<template>
	<Head :title="t('dashboard.title')" />
	<div class="mb-6 flex items-center justify-between gap-4">
		<h1 class="text-2xl font-semibold" data-testid="dashboard-hello">{{ t("dashboard.hello", { name }) }}</h1>
		<Link
			href="/settings/contact"
			data-testid="contact-settings"
			class="ml-auto rounded-md border border-zinc-300 px-3 py-2 text-sm hover:bg-zinc-100 dark:border-zinc-700 dark:hover:bg-zinc-800"
		>
			{{ t("contact.nav") }}
		</Link>
		<Link
			href="/raffles/create"
			data-testid="new-raffle"
			class="rounded-md bg-brand-600 px-4 py-2 font-medium text-white hover:bg-brand-700"
		>
			{{ t("raffle.actions.new") }}
		</Link>
	</div>
	<BalanceCard :balance-cents="balance_cents" />
	<p v-if="raffles.length === 0" class="text-zinc-600 dark:text-zinc-400">{{ t("raffle.list.empty") }}</p>
	<ul v-else class="divide-y divide-zinc-200 dark:divide-zinc-800" data-testid="raffle-list">
		<li v-for="r in raffles" :key="r.id" class="py-3">
			<Link :href="`/raffles/${r.id}`" class="font-medium text-brand-700 hover:underline dark:text-brand-500">
				{{ r.title }}
			</Link>
			<p class="text-sm text-zinc-600 dark:text-zinc-400">
				{{ t(`raffle.status.${r.status}`) }} · {{ formatReais(r.price_reais, locale) }} ·
				{{ t("raffle.ends", { date: formatDateTime(r.ends_at, locale) }) }}
			</p>
			<p class="text-sm text-zinc-600 dark:text-zinc-400">
				{{ t("raffle.list.counts", { paid: r.paid, reserved: r.reserved, available: r.available }) }}
			</p>
		</li>
	</ul>
</template>
