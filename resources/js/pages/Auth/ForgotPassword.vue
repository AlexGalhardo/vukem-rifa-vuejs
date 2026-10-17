<script setup lang="ts">
import { Head, useForm } from "@inertiajs/vue3";

import Field from "../../components/Field.vue";
import PrimaryButton from "../../components/PrimaryButton.vue";
import { useI18n } from "../../composables/useI18n";

const { t } = useI18n();
const form = useForm({ email: "" });

function submit(): void {
	form.post("/forgot-password");
}
</script>

<template>
	<Head :title="t('auth.forgot.title')" />
	<div class="mx-auto max-w-sm">
		<h1 class="mb-2 text-2xl font-semibold">{{ t("auth.forgot.title") }}</h1>
		<p class="mb-4 text-zinc-600 dark:text-zinc-400">{{ t("auth.forgot.lead") }}</p>
		<form class="space-y-4" @submit.prevent="submit">
			<Field
				v-model="form.email"
				:label="t('auth.email')"
				type="email"
				autocomplete="email"
				required
				:error="form.errors.email"
			/>
			<PrimaryButton :disabled="form.processing">{{ t("auth.forgot.submit") }}</PrimaryButton>
		</form>
	</div>
</template>
