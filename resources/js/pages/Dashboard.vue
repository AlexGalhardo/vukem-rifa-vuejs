<script setup lang="ts">
import { Head, usePage } from "@inertiajs/vue3";
import { computed } from "vue";

import { useI18n } from "../composables/useI18n";
import type { SharedProps } from "../types";

const { t } = useI18n();
const page = usePage<SharedProps>();
const name = computed(() => page.props.auth?.user?.name ?? "");
</script>

<template>
	<Head :title="t('dashboard.title')" />
	<h1 class="text-2xl font-semibold" data-testid="dashboard-hello">{{ t("dashboard.hello", { name }) }}</h1>
</template>
