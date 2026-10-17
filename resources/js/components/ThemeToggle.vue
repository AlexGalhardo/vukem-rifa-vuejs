<script setup lang="ts">
import { useI18n } from "../composables/useI18n";
import { useTheme } from "../composables/useTheme";

const { theme, toggle } = useTheme();
const { t } = useI18n();
</script>

<template>
	<button
		type="button"
		data-testid="theme-toggle"
		:aria-label="t('theme.toggle')"
		:aria-pressed="theme === 'dark'"
		class="rounded-md border border-zinc-300 px-3 py-1 text-sm hover:bg-zinc-100 dark:border-zinc-700 dark:hover:bg-zinc-800"
		@click="toggle"
	>
		{{ theme === "dark" ? t("theme.light") : t("theme.dark") }}
	</button>
</template>
