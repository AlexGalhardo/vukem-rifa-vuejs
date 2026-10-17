<script setup lang="ts">
import { Head, Link } from "@inertiajs/vue3";

import { useI18n } from "../../composables/useI18n";

const { t } = useI18n();
</script>

<template>
	<Head :title="t('auth.check_email.title')" />
	<div class="mx-auto max-w-md" data-testid="check-email">
		<h1 class="mb-2 text-2xl font-semibold">{{ t("auth.check_email.title") }}</h1>
		<p class="mb-2">{{ t("auth.check_email.lead") }}</p>
		<p class="mb-4 text-sm text-zinc-600 dark:text-zinc-400">{{ t("auth.check_email.spam") }}</p>
		<Link href="/login" class="text-brand-700 underline dark:text-brand-500">{{ t("auth.check_email.login") }}</Link>
	</div>
</template>
