<script setup lang="ts">
import { Head, useForm } from "@inertiajs/vue3";

import Field from "../../components/Field.vue";
import PrimaryButton from "../../components/PrimaryButton.vue";
import { useI18n } from "../../composables/useI18n";

const props = defineProps<{ token: string; email: string }>();

const { t } = useI18n();
const form = useForm({ token: props.token, email: props.email, password: "", password_confirmation: "" });

function submit(): void {
	form.post("/reset-password", { onFinish: () => form.reset("password", "password_confirmation") });
}
</script>

<template>
	<Head :title="t('auth.reset.title')" />
	<div class="mx-auto max-w-sm">
		<h1 class="mb-4 text-2xl font-semibold">{{ t("auth.reset.title") }}</h1>
		<form class="space-y-4" @submit.prevent="submit">
			<Field
				v-model="form.email"
				:label="t('auth.email')"
				type="email"
				autocomplete="email"
				required
				:error="form.errors.email"
			/>
			<Field
				v-model="form.password"
				:label="t('auth.new_password_hint')"
				type="password"
				autocomplete="new-password"
				minlength="12"
				required
				:error="form.errors.password"
			/>
			<Field
				v-model="form.password_confirmation"
				:label="t('auth.confirm_password')"
				type="password"
				autocomplete="new-password"
				required
				:error="form.errors.password_confirmation"
			/>
			<PrimaryButton :disabled="form.processing">{{ t("auth.reset.submit") }}</PrimaryButton>
		</form>
	</div>
</template>
