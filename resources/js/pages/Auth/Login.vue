<script setup lang="ts">
import { Head, Link, useForm } from "@inertiajs/vue3";

import Field from "../../components/Field.vue";
import GoogleButton from "../../components/GoogleButton.vue";
import PrimaryButton from "../../components/PrimaryButton.vue";
import { useI18n } from "../../composables/useI18n";

const { t } = useI18n();
const form = useForm({ email: "", password: "", remember: false });

function submit(): void {
	form.post("/login", { onFinish: () => form.reset("password") });
}
</script>

<template>
	<Head :title="t('auth.login.title')" />
	<div class="mx-auto max-w-sm">
		<h1 class="mb-4 text-2xl font-semibold">{{ t("auth.login.title") }}</h1>
		<GoogleButton />
		<p class="my-4 text-center text-sm text-zinc-600 dark:text-zinc-400">{{ t("auth.or") }}</p>
		<form class="space-y-4" @submit.prevent="submit">
			<Field
				v-model="form.email"
				:label="t('auth.email')"
				type="email"
				autocomplete="username"
				required
				:error="form.errors.email"
			/>
			<Field
				v-model="form.password"
				:label="t('auth.password')"
				type="password"
				autocomplete="current-password"
				required
				:error="form.errors.password"
			/>
			<label class="flex items-center gap-2">
				<input v-model="form.remember" type="checkbox" />
				{{ t("auth.remember") }}
			</label>
			<PrimaryButton :disabled="form.processing">{{ t("auth.login.submit") }}</PrimaryButton>
		</form>
		<p class="mt-6 flex gap-4 text-sm">
			<Link href="/forgot-password" class="text-brand-700 hover:underline dark:text-brand-500">
				{{ t("auth.login.forgot") }}
			</Link>
			<Link href="/register" class="text-brand-700 hover:underline dark:text-brand-500">
				{{ t("auth.login.register") }}
			</Link>
		</p>
	</div>
</template>
