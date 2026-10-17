<script setup lang="ts">
import { Head, Link, useForm } from "@inertiajs/vue3";

import Field from "../../components/Field.vue";
import GoogleButton from "../../components/GoogleButton.vue";
import PrimaryButton from "../../components/PrimaryButton.vue";
import { useI18n } from "../../composables/useI18n";

const { t } = useI18n();
const form = useForm({ name: "", email: "", password: "", password_confirmation: "" });

function submit(): void {
	form.post("/register", { onFinish: () => form.reset("password", "password_confirmation") });
}
</script>

<template>
	<Head :title="t('auth.register.title')" />
	<div class="mx-auto max-w-sm">
		<h1 class="mb-4 text-2xl font-semibold">{{ t("auth.register.title") }}</h1>
		<GoogleButton />
		<p class="my-4 text-center text-sm text-zinc-600 dark:text-zinc-400">{{ t("auth.or") }}</p>
		<form class="space-y-4" @submit.prevent="submit">
			<Field v-model="form.name" :label="t('auth.name')" autocomplete="name" required :error="form.errors.name" />
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
				:label="t('auth.password_hint')"
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
			<PrimaryButton :disabled="form.processing">{{ t("auth.register.submit") }}</PrimaryButton>
		</form>
		<p class="mt-6 text-sm">
			<Link href="/login" class="text-brand-700 hover:underline dark:text-brand-500">
				{{ t("auth.register.login") }}
			</Link>
		</p>
	</div>
</template>
