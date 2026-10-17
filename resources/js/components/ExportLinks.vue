<script setup lang="ts">
import { useI18n } from "../composables/useI18n";

// Plain links (not Inertia visits): the server answers with a file download.
defineProps<{ raffleId: number }>();

const { t } = useI18n();
const button =
	"rounded-md border border-zinc-300 px-3 py-1 text-sm hover:bg-zinc-100 dark:border-zinc-700 dark:hover:bg-zinc-800";
</script>

<template>
	<a :href="`/raffles/${raffleId}/export.csv`" :class="button" download data-testid="export-csv">{{ t("export.csv") }}</a>
	<a :href="`/raffles/${raffleId}/export.json`" :class="button" download data-testid="export-json">
		{{ t("export.json") }}
	</a>
</template>
