<script setup lang="ts">
import { Head } from "@inertiajs/vue3";

import { useI18n } from "../composables/useI18n";

const { t } = useI18n();
</script>

<template>
	<Head :title="t('home.title')" />
	<section class="py-16 text-center">
		<h1 class="text-4xl font-semibold tracking-tight" data-testid="home-title">{{ t("home.title") }}</h1>
		<p class="mx-auto mt-4 max-w-xl text-zinc-600 dark:text-zinc-400">{{ t("home.lead") }}</p>
	</section>
</template>
