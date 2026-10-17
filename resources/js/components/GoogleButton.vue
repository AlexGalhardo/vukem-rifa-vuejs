<script setup lang="ts">
import { useI18n } from "../composables/useI18n";

const { t } = useI18n();
</script>

<template>
	<!-- A full-page navigation: the OAuth redirect cannot be an Inertia visit. -->
	<a
		href="/auth/google/redirect"
		data-testid="google-login"
		class="flex w-full items-center justify-center gap-2 rounded-md border border-zinc-300 px-4 py-2 font-medium hover:bg-zinc-100 dark:border-zinc-700 dark:hover:bg-zinc-800"
	>
		{{ t("auth.google") }}
	</a>
</template>
