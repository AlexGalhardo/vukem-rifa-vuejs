<script setup lang="ts">
import { Link, usePage } from "@inertiajs/vue3";
import { computed } from "vue";

import LocaleSwitcher from "../components/LocaleSwitcher.vue";
import ThemeToggle from "../components/ThemeToggle.vue";
import { useI18n } from "../composables/useI18n";
import type { SharedProps } from "../types";

const { t } = useI18n();
const page = usePage<SharedProps>();
const user = computed(() => page.props.auth?.user ?? null);
const message = computed(() => page.flash?.success ?? page.flash?.status);
const link = "text-sm text-brand-700 hover:underline dark:text-brand-500";
</script>

<template>
	<header class="flex items-center justify-between border-b border-zinc-200 px-4 py-3 dark:border-zinc-800">
		<Link href="/" class="font-bold text-brand-700 dark:text-brand-500">{{ page.props.appName }}</Link>
		<nav class="flex items-center gap-3">
			<template v-if="user">
				<Link href="/dashboard" :class="link">{{ t("nav.dashboard") }}</Link>
				<Link href="/settings/api-keys" :class="link" data-testid="nav-api-keys">{{ t("apikeys.nav") }}</Link>
				<Link href="/logout" method="post" as="button" :class="link" data-testid="logout">
					{{ t("nav.logout") }}
				</Link>
			</template>
			<template v-else>
				<Link href="/login" :class="link">{{ t("nav.login") }}</Link>
				<Link href="/register" :class="link">{{ t("nav.register") }}</Link>
			</template>
			<LocaleSwitcher />
			<ThemeToggle />
		</nav>
	</header>
	<main class="mx-auto max-w-4xl p-4">
		<p
			v-if="typeof message === 'string'"
			role="status"
			class="mb-4 rounded-md bg-brand-50 px-4 py-3 text-brand-700 dark:bg-zinc-900 dark:text-brand-500"
		>
			{{ message }}
		</p>
		<slot />
	</main>
</template>
