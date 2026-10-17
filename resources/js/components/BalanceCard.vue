<script setup lang="ts">
import { useI18n } from "../composables/useI18n";
import { formatCents } from "../lib/format";

defineProps<{ balanceCents: number }>();

const { t, locale } = useI18n();
</script>

<template>
	<section
		class="mb-6 rounded-lg border border-zinc-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-900"
		aria-labelledby="balance-title"
	>
		<h2 id="balance-title" class="text-sm text-zinc-600 dark:text-zinc-400">{{ t("ledger.balance") }}</h2>
		<p class="mt-1 text-2xl font-semibold text-brand-700 dark:text-brand-500" data-testid="owner-balance">
			{{ formatCents(balanceCents, locale) }}
		</p>
		<p class="mt-1 text-xs text-zinc-500">{{ t("ledger.balance_hint") }}</p>
	</section>
</template>
