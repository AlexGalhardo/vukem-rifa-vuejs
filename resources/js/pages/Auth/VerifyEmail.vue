<script setup lang="ts">
import { Head, Link, usePage } from "@inertiajs/vue3";
import { computed } from "vue";

import { useI18n } from "../../composables/useI18n";
import type { SharedProps } from "../../types";

const { t } = useI18n();
const page = usePage<SharedProps>();
const email = computed(() => page.props.auth?.user?.email ?? "");
</script>

<template>
	<Head :title="t('auth.verify.title')" />
	<div class="mx-auto max-w-md">
		<h1 class="mb-2 text-2xl font-semibold">{{ t("auth.verify.title") }}</h1>
		<p class="mb-4">{{ t("auth.verify.lead", { email }) }}</p>
		<Link
			href="/email/verification-notification"
			method="post"
			as="button"
			class="rounded-md bg-brand-600 px-4 py-2 font-medium text-white hover:bg-brand-700"
		>
			{{ t("auth.verify.resend") }}
		</Link>
	</div>
</template>
