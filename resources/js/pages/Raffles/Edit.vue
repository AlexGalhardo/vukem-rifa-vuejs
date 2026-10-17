<script setup lang="ts">
import { Head, useForm } from "@inertiajs/vue3";

import RaffleForm from "../../components/RaffleForm.vue";
import { useI18n } from "../../composables/useI18n";
import type { RaffleForm as RaffleFields } from "../../lib/raffleSchema";
import type { OwnedRaffle, RaffleLimits } from "../../types";

const props = defineProps<{ raffle: OwnedRaffle; limits: RaffleLimits }>();

const { t } = useI18n();
const form = useForm<RaffleFields>({
	title: props.raffle.title,
	description: props.raffle.description,
	number_count: props.raffle.number_count,
	price_reais: props.raffle.price_reais,
	duration_days: props.raffle.duration_days,
});

function submit(): void {
	form.put(`/raffles/${props.raffle.id}`);
}
</script>

<template>
	<Head :title="t('raffle.edit_title')" />
	<h1 class="mb-4 text-2xl font-semibold">{{ t("raffle.edit_title") }}</h1>
	<p
		class="mb-4 rounded-md bg-amber-50 px-3 py-2 text-sm text-amber-900 dark:bg-amber-950 dark:text-amber-100"
		data-testid="edit-cancels-links"
	>
		{{ t("raffle.edit_cancels_links") }}
	</p>
	<RaffleForm
		:form="form"
		:limits="props.limits"
		:locked="props.raffle.locked"
		:submit-label="t('raffle.actions.save')"
		@submit="submit"
	/>
</template>
