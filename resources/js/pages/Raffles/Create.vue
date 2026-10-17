<script setup lang="ts">
import { Head, useForm } from "@inertiajs/vue3";

import RaffleForm from "../../components/RaffleForm.vue";
import { useI18n } from "../../composables/useI18n";
import type { RaffleForm as RaffleFields } from "../../lib/raffleSchema";
import type { RaffleLimits } from "../../types";

const props = defineProps<{ limits: RaffleLimits }>();

const { t } = useI18n();
const form = useForm<RaffleFields>({
	title: "",
	description: "",
	number_count: 100,
	price_reais: 10,
	duration_days: 30,
});

function submit(): void {
	form.post("/raffles");
}
</script>

<template>
	<Head :title="t('raffle.create_title')" />
	<h1 class="mb-4 text-2xl font-semibold">{{ t("raffle.create_title") }}</h1>
	<RaffleForm :form="form" :limits="props.limits" :submit-label="t('raffle.actions.create')" @submit="submit" />
</template>
