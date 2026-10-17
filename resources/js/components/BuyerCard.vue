<script setup lang="ts">
import { router } from "@inertiajs/vue3";

import { useI18n } from "../composables/useI18n";
import { formatDateTime } from "../lib/format";
import type { Buyer } from "../types";

// The owner's view of a reserved or paid number. Never used on public pages.
const props = defineProps<{ raffleId: number; buyer: Buyer; cancellable: boolean }>();
const emit = defineEmits<{ close: [] }>();

const { t, locale } = useI18n();

function cancel(): void {
	if (!window.confirm(t("link.details.confirm_cancel"))) return;
	router.post(
		`/raffles/${props.raffleId}/numbers/${props.buyer.number}/cancel`,
		{},
		{ preserveScroll: true, onSuccess: () => emit("close") },
	);
}
</script>

<template>
	<section class="space-y-3" data-testid="buyer-card">
		<div class="flex items-start justify-between gap-4">
			<h3 class="text-lg font-semibold">{{ t("link.details.title", { number: buyer.number }) }}</h3>
			<button
				type="button"
				class="rounded-md px-2 text-zinc-500 hover:bg-zinc-100 dark:hover:bg-zinc-800"
				:aria-label="t('link.form.close')"
				@click="emit('close')"
			>
				✕
			</button>
		</div>
		<p class="text-sm font-medium" data-testid="buyer-status">{{ t(`link.details.status.${buyer.status}`) }}</p>
		<dl class="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1 text-sm">
			<dt class="text-zinc-500 dark:text-zinc-400">{{ t("link.form.buyer_name") }}</dt>
			<dd data-testid="buyer-card-name">{{ buyer.buyer_name }}</dd>
			<dt class="text-zinc-500 dark:text-zinc-400">{{ t("link.form.buyer_email") }}</dt>
			<dd class="break-all">{{ buyer.buyer_email }}</dd>
			<dt class="text-zinc-500 dark:text-zinc-400">{{ t("link.form.buyer_phone") }}</dt>
			<dd>{{ buyer.buyer_phone }}</dd>
		</dl>
		<p v-if="buyer.status === 'paid' && buyer.paid_at" class="text-sm text-zinc-600 dark:text-zinc-400">
			{{ t("link.details.paid_at", { date: formatDateTime(buyer.paid_at, locale) }) }}
		</p>
		<p v-else-if="buyer.status !== 'paid'" class="text-sm text-zinc-600 dark:text-zinc-400" data-testid="buyer-expires">
			{{ t("link.details.expires_at", { date: formatDateTime(buyer.expires_at, locale) }) }}
		</p>
		<button
			v-if="cancellable && buyer.status !== 'paid'"
			type="button"
			class="rounded-md border border-red-300 px-3 py-1 text-sm text-red-700 hover:bg-red-50 dark:border-red-800 dark:text-red-300 dark:hover:bg-red-950"
			data-testid="link-cancel"
			@click="cancel"
		>
			{{ t("link.details.cancel") }}
		</button>
	</section>
</template>
