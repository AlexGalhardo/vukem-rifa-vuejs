<script setup lang="ts">
import { Head, Link } from "@inertiajs/vue3";

import { useI18n } from "../../composables/useI18n";
import { formatCents, formatDateTime } from "../../lib/format";
import type { Purchase } from "../../types";

defineProps<{ paid: Purchase[]; unpaid: Purchase[] }>();

const { t, locale } = useI18n();
const cell = "px-3 py-2 text-left";
</script>

<template>
	<Head :title="t('purchases.title')" />
	<h1 class="mb-6 text-2xl font-semibold">{{ t("purchases.title") }}</h1>

	<section class="mb-10" data-testid="purchases-paid">
		<h2 class="mb-2 text-lg font-semibold">{{ t("purchases.paid") }}</h2>
		<p v-if="paid.length === 0" class="text-sm text-zinc-600 dark:text-zinc-400">{{ t("purchases.none_paid") }}</p>
		<div v-else class="overflow-x-auto rounded-lg border border-zinc-200 dark:border-zinc-800">
			<table class="w-full text-sm">
				<thead class="bg-zinc-50 text-zinc-600 dark:bg-zinc-900 dark:text-zinc-400">
					<tr>
						<th :class="cell" scope="col">{{ t("purchases.raffle") }}</th>
						<th :class="cell" scope="col">{{ t("purchases.number") }}</th>
						<th :class="cell" scope="col">{{ t("purchases.amount") }}</th>
						<th :class="cell" scope="col">{{ t("purchases.paid_at") }}</th>
					</tr>
				</thead>
				<tbody>
					<tr v-for="p in paid" :key="`${p.raffle_slug}-${p.number}`" class="border-t border-zinc-200 dark:border-zinc-800">
						<td :class="cell">
							<Link :href="`/r/${p.raffle_slug}`" class="text-brand-700 hover:underline dark:text-brand-500">
								{{ p.raffle_title }}
							</Link>
						</td>
						<td :class="cell" class="tabular-nums">{{ p.number }}</td>
						<td :class="cell" class="tabular-nums">{{ formatCents(p.amount_cents, locale) }}</td>
						<td :class="cell">{{ p.paid_at ? formatDateTime(p.paid_at, locale) : "" }}</td>
					</tr>
				</tbody>
			</table>
		</div>
	</section>

	<section data-testid="purchases-unpaid">
		<h2 class="mb-2 text-lg font-semibold">{{ t("purchases.unpaid") }}</h2>
		<p v-if="unpaid.length === 0" class="text-sm text-zinc-600 dark:text-zinc-400">{{ t("purchases.none_unpaid") }}</p>
		<div v-else class="overflow-x-auto rounded-lg border border-zinc-200 dark:border-zinc-800">
			<table class="w-full text-sm">
				<thead class="bg-zinc-50 text-zinc-600 dark:bg-zinc-900 dark:text-zinc-400">
					<tr>
						<th :class="cell" scope="col">{{ t("purchases.raffle") }}</th>
						<th :class="cell" scope="col">{{ t("purchases.number") }}</th>
						<th :class="cell" scope="col">{{ t("purchases.status") }}</th>
						<th :class="cell" scope="col">{{ t("purchases.created_at") }}</th>
						<th :class="cell" scope="col">{{ t("purchases.expires_at") }}</th>
					</tr>
				</thead>
				<tbody>
					<tr v-for="p in unpaid" :key="`${p.raffle_slug}-${p.number}-${p.created_at}`" class="border-t border-zinc-200 dark:border-zinc-800">
						<td :class="cell">
							<Link :href="`/r/${p.raffle_slug}`" class="text-brand-700 hover:underline dark:text-brand-500">
								{{ p.raffle_title }}
							</Link>
						</td>
						<td :class="cell" class="tabular-nums">{{ p.number }}</td>
						<td :class="cell">{{ t(`purchases.statuses.${p.status}`) }}</td>
						<td :class="cell">{{ formatDateTime(p.created_at, locale) }}</td>
						<td :class="cell">{{ formatDateTime(p.expires_at, locale) }}</td>
					</tr>
				</tbody>
			</table>
		</div>
	</section>
</template>
