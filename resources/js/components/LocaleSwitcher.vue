<script setup lang="ts">
import { usePage } from "@inertiajs/vue3";
import { computed } from "vue";

import { useI18n } from "../composables/useI18n";
import type { Locale, SharedProps } from "../types";

const page = usePage<SharedProps>();
const { locale, t, setLocale } = useI18n();
const locales = computed<Locale[]>(() => page.props.locales ?? ["pt-BR", "en"]);

function onChange(event: Event): void {
	setLocale((event.target as HTMLSelectElement).value as Locale);
}
</script>

<template>
	<label class="flex items-center gap-2 text-sm">
		<span class="sr-only">{{ t("locale.switch") }}</span>
		<select
			data-testid="locale-switcher"
			:value="locale"
			class="rounded-md border border-zinc-300 bg-transparent px-2 py-1 dark:border-zinc-700"
			@change="onChange"
		>
			<option v-for="l in locales" :key="l" :value="l" class="text-zinc-900">{{ t(`locale.${l}`) }}</option>
		</select>
	</label>
</template>
