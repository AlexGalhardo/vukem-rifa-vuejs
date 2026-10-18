<script setup lang="ts">
import { Head, Link } from "@inertiajs/vue3";
import { computed } from "vue";

import { useI18n } from "../composables/useI18n";

// Rendered by vukem for errors on Inertia visits (404, 403, 500…). message
// is safe to show: 5xx only ever carry the status text. When the server only
// sent the status text (message === title), the catalog says it in the
// visitor's language.
const props = defineProps<{ status: number; title: string; message: string }>();

const { t } = useI18n();

const known = [403, 404, 405, 409, 410, 419, 429, 500, 503];
const heading = computed(() =>
	props.message === props.title && known.includes(props.status) ? t(`error_page.s${props.status}`) : props.message,
);
</script>

<template>
	<Head :title="`${status} · ${heading}`" />
	<section class="py-12 text-center" data-testid="error-page">
		<p class="text-sm font-semibold text-brand-700 dark:text-brand-500" data-testid="error-status">{{ status }}</p>
		<h1 class="mt-2 text-3xl font-semibold tracking-tight">{{ heading }}</h1>
		<Link href="/" class="mt-6 inline-block text-brand-700 underline hover:no-underline dark:text-brand-500">
			{{ t("error_page.home") }}
		</Link>
	</section>
</template>
