<script setup lang="ts">
import type { InertiaForm } from "@inertiajs/vue3";

import { useI18n } from "../composables/useI18n";
import { type RaffleForm, raffleSchema } from "../lib/raffleSchema";
import type { RaffleLimits } from "../types";
import PrimaryButton from "./PrimaryButton.vue";

const props = defineProps<{
	form: InertiaForm<RaffleForm>;
	limits: RaffleLimits;
	locked?: boolean;
	submitLabel: string;
}>();
const emit = defineEmits<{ submit: [] }>();

const { t } = useI18n();
const input =
	"mt-1 block w-full rounded-md border border-zinc-300 bg-transparent px-3 py-2 focus:border-brand-500 focus:outline-none disabled:opacity-60 dark:border-zinc-700";

const fieldKeys = ["title", "description", "number_count", "price_reais", "duration_days"] as const;

// Client-side check with the same rules as the Go schema; failures show
// next to the fields without a round trip.
function submit(): void {
	props.form.clearErrors();
	const result = raffleSchema(props.limits).safeParse(props.form.data());
	if (!result.success) {
		for (const issue of result.error.issues) {
			const key = issue.path[0];
			if (typeof key === "string" && (fieldKeys as readonly string[]).includes(key)) {
				props.form.setError(key as (typeof fieldKeys)[number], t("raffle.invalid_field"));
			}
		}
		return;
	}
	emit("submit");
}
</script>

<template>
	<form class="space-y-4" novalidate @submit.prevent="submit">
		<label class="block">
			<span class="font-medium">{{ t("raffle.fields.title") }}</span>
			<input v-model="form.title" data-testid="raffle-title" :class="input" maxlength="120" required />
			<p v-if="form.errors.title" class="mt-1 text-sm text-red-600 dark:text-red-400">{{ form.errors.title }}</p>
		</label>
		<label class="block">
			<span class="font-medium">{{ t("raffle.fields.description") }}</span>
			<textarea v-model="form.description" data-testid="raffle-description" :class="input" rows="4" maxlength="5000" />
			<p v-if="form.errors.description" class="mt-1 text-sm text-red-600 dark:text-red-400">
				{{ form.errors.description }}
			</p>
		</label>
		<p v-if="locked" class="rounded-md bg-amber-50 px-3 py-2 text-sm text-amber-900 dark:bg-amber-950 dark:text-amber-100">
			{{ t("raffle.hints.locked") }}
		</p>
		<div class="grid gap-4 sm:grid-cols-3">
			<label class="block">
				<span class="font-medium">{{ t("raffle.fields.number_count") }}</span>
				<input
					v-model.number="form.number_count"
					data-testid="raffle-number-count"
					type="number"
					:min="limits.min_numbers"
					:max="limits.max_numbers"
					step="1"
					:disabled="locked"
					:class="input"
				/>
				<span class="mt-1 block text-xs text-zinc-500">
					{{ t("raffle.hints.number_count", { min: limits.min_numbers, max: limits.max_numbers }) }}
				</span>
				<p v-if="form.errors.number_count" class="mt-1 text-sm text-red-600 dark:text-red-400">
					{{ form.errors.number_count }}
				</p>
			</label>
			<label class="block">
				<span class="font-medium">{{ t("raffle.fields.price_reais") }}</span>
				<input
					v-model.number="form.price_reais"
					data-testid="raffle-price"
					type="number"
					:min="limits.min_price"
					:max="limits.max_price"
					step="1"
					:disabled="locked"
					:class="input"
				/>
				<span class="mt-1 block text-xs text-zinc-500">
					{{ t("raffle.hints.price_reais", { min: limits.min_price, max: limits.max_price }) }}
				</span>
				<p v-if="form.errors.price_reais" class="mt-1 text-sm text-red-600 dark:text-red-400">
					{{ form.errors.price_reais }}
				</p>
			</label>
			<label class="block">
				<span class="font-medium">{{ t("raffle.fields.duration_days") }}</span>
				<input
					v-model.number="form.duration_days"
					data-testid="raffle-duration"
					type="number"
					:min="limits.min_duration"
					:max="limits.max_duration"
					step="1"
					:disabled="locked"
					:class="input"
				/>
				<span class="mt-1 block text-xs text-zinc-500">
					{{ t("raffle.hints.duration_days", { min: limits.min_duration, max: limits.max_duration }) }}
				</span>
				<p v-if="form.errors.duration_days" class="mt-1 text-sm text-red-600 dark:text-red-400">
					{{ form.errors.duration_days }}
				</p>
			</label>
		</div>
		<PrimaryButton :disabled="form.processing" data-testid="raffle-submit">{{ submitLabel }}</PrimaryButton>
	</form>
</template>
