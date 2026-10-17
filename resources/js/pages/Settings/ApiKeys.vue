<script setup lang="ts">
import { Head, router, useForm } from "@inertiajs/vue3";
import { ref } from "vue";

import PrimaryButton from "../../components/PrimaryButton.vue";
import { useI18n } from "../../composables/useI18n";
import { type ApiKeyForm, apiKeySchema } from "../../lib/apiKeySchema";
import { formatDateTime } from "../../lib/format";
import type { ApiAbility, ApiKey } from "../../types";

// plain_token is only set on the response that created the key.
defineProps<{ keys: ApiKey[]; abilities: ApiAbility[]; plain_token: string | null }>();

const { t, locale } = useI18n();
const form = useForm<ApiKeyForm>({ name: "", abilities: ["raffles:read"] });
const copied = ref(false);
const input =
	"mt-1 block w-full rounded-md border border-zinc-300 bg-transparent px-3 py-2 focus:border-brand-500 focus:outline-none dark:border-zinc-700";
const button =
	"rounded-md border border-zinc-300 px-3 py-1 text-sm hover:bg-zinc-100 dark:border-zinc-700 dark:hover:bg-zinc-800";

function abilityLabel(a: ApiAbility): string {
	return t(`apikeys.ability.${a.replace(":", "_")}`);
}

// Server errors for list items come back as "abilities.0".
function abilitiesError(): string | undefined {
	const errors = form.errors as Record<string, string | undefined>;
	return Object.entries(errors).find(([k]) => k === "abilities" || k.startsWith("abilities."))?.[1];
}

function submit(): void {
	form.clearErrors();
	const result = apiKeySchema.safeParse(form.data());
	if (!result.success) {
		for (const issue of result.error.issues) {
			const key = issue.path[0];
			if (key === "name" || key === "abilities") form.setError(key, t("apikeys.invalid_field"));
		}
		return;
	}
	copied.value = false;
	form.post("/settings/api-keys", { onSuccess: () => form.reset("name") });
}

async function copy(token: string): Promise<void> {
	await navigator.clipboard.writeText(token);
	copied.value = true;
}

function revoke(key: ApiKey): void {
	if (window.confirm(t("apikeys.confirm_revoke", { name: key.name }))) {
		router.delete(`/settings/api-keys/${encodeURIComponent(key.id)}`, { preserveScroll: true });
	}
}
</script>

<template>
	<Head :title="t('apikeys.title')" />
	<div class="mb-6 flex flex-wrap items-start justify-between gap-4">
		<div>
			<h1 class="text-2xl font-semibold">{{ t("apikeys.title") }}</h1>
			<p class="mt-1 text-sm text-zinc-600 dark:text-zinc-400">{{ t("apikeys.lead") }}</p>
		</div>
		<a href="/api/docs" :class="button" data-testid="api-docs-link">{{ t("apikeys.docs") }}</a>
	</div>

	<div
		v-if="plain_token"
		class="mb-6 rounded-md border border-amber-300 bg-amber-50 p-4 text-amber-950 dark:border-amber-700 dark:bg-amber-950 dark:text-amber-50"
		role="alert"
	>
		<p class="mb-2 font-medium">{{ t("apikeys.created") }}</p>
		<div class="flex flex-wrap items-center gap-2">
			<code
				class="break-all rounded bg-white px-2 py-1 font-mono text-sm text-zinc-900 dark:bg-zinc-900 dark:text-zinc-100"
				data-testid="api-key-token"
				>{{ plain_token }}</code
			>
			<button type="button" :class="button" data-testid="api-key-copy" @click="copy(plain_token)">
				{{ copied ? t("apikeys.copied") : t("apikeys.copy") }}
			</button>
		</div>
	</div>

	<form class="mb-8 space-y-4" novalidate data-testid="api-key-form" @submit.prevent="submit">
		<label class="block">
			<span class="font-medium">{{ t("apikeys.name") }}</span>
			<input v-model="form.name" data-testid="api-key-name" :class="input" maxlength="60" required />
			<p v-if="form.errors.name" class="mt-1 text-sm text-red-600 dark:text-red-400">{{ form.errors.name }}</p>
		</label>
		<fieldset>
			<legend class="font-medium">{{ t("apikeys.abilities") }}</legend>
			<label v-for="a in abilities" :key="a" class="mt-1 flex items-center gap-2">
				<input
					v-model="form.abilities"
					type="checkbox"
					:value="a"
					:data-testid="`api-key-ability-${a.replace(':', '-')}`"
					class="accent-brand-600"
				/>
				<span>{{ abilityLabel(a) }}</span>
			</label>
			<p v-if="abilitiesError()" class="mt-1 text-sm text-red-600 dark:text-red-400">{{ abilitiesError() }}</p>
		</fieldset>
		<PrimaryButton :disabled="form.processing" data-testid="api-key-submit">{{ t("apikeys.create") }}</PrimaryButton>
	</form>

	<p v-if="keys.length === 0" class="text-zinc-600 dark:text-zinc-400" data-testid="api-key-empty">
		{{ t("apikeys.empty") }}
	</p>
	<ul v-else class="divide-y divide-zinc-200 dark:divide-zinc-800" data-testid="api-key-list">
		<li v-for="k in keys" :key="k.id" class="flex flex-wrap items-center justify-between gap-2 py-3">
			<div>
				<p class="font-medium" data-testid="api-key-row-name">{{ k.name }}</p>
				<p class="text-sm text-zinc-600 dark:text-zinc-400">
					{{ k.abilities.map(abilityLabel).join(" · ") }} ·
					{{
						k.last_used_at
							? t("apikeys.last_used", { date: formatDateTime(k.last_used_at, locale) })
							: t("apikeys.never_used")
					}}
				</p>
			</div>
			<button type="button" :class="button" data-testid="api-key-revoke" @click="revoke(k)">
				{{ t("apikeys.revoke") }}
			</button>
		</li>
	</ul>
</template>
