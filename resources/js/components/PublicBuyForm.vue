<script setup lang="ts">
import { useForm } from "@inertiajs/vue3";
import { computed } from "vue";

import { useI18n } from "../composables/useI18n";
import { buyerSchema, maskPhone } from "../lib/buyerSchema";
import { formatReais } from "../lib/format";
import Field from "./Field.vue";
import PrimaryButton from "./PrimaryButton.vue";

// The public purchase of one free number: the buyer's contact, the LGPD
// consent and a honeypot ("website", hidden from people) — then the server
// redirects to the same /pay/{token} page as owner-generated links.
const props = defineProps<{
	slug: string;
	number: number;
	priceReais: number;
	prefill?: { name: string; email: string };
}>();
const emit = defineEmits<{ close: [] }>();

const { t, locale } = useI18n();
const form = useForm({
	buyer_name: props.prefill?.name ?? "",
	buyer_email: props.prefill?.email ?? "",
	buyer_phone: "",
	consent: false,
	website: "",
});

const clientErrors = computed<Record<string, string>>(() => {
	const parsed = buyerSchema.safeParse(form.data());
	const out: Record<string, string> = {};
	if (!parsed.success) {
		for (const issue of parsed.error.issues) {
			const key = String(issue.path[0]);
			out[key] = t(`link.form.invalid_${key.replace("buyer_", "")}`);
		}
	}
	if (!form.consent) out.consent = t("link.public.consent_required");
	return out;
});

// "number" errors come from the server (taken number, caps): not a form field.
const numberError = computed(() => (form.errors as Partial<Record<string, string>>).number);

function submit(): void {
	if (Object.keys(clientErrors.value).length > 0) {
		form.setError(clientErrors.value);
		return;
	}
	form.post(`/r/${props.slug}/numbers/${props.number}/reserve`, { preserveScroll: true });
}

function onPhone(v: string): void {
	form.buyer_phone = maskPhone(v);
}
</script>

<template>
	<form class="space-y-4" novalidate data-testid="public-buy-form" @submit.prevent="submit">
		<div class="flex items-start justify-between gap-4">
			<h3 class="text-lg font-semibold">
				{{ t("link.public.title", { number }) }}
				<span class="block text-sm font-normal text-zinc-600 dark:text-zinc-400">
					{{ t("raffle.price", { price: formatReais(priceReais, locale) }) }}
				</span>
			</h3>
			<button
				type="button"
				class="rounded-md px-2 text-zinc-600 hover:bg-zinc-100 dark:text-zinc-400 dark:hover:bg-zinc-800"
				:aria-label="t('link.form.close')"
				@click="emit('close')"
			>
				✕
			</button>
		</div>
		<p
			v-if="numberError"
			role="alert"
			data-testid="public-buy-error"
			class="rounded-md bg-red-50 px-3 py-2 text-sm text-red-800 dark:bg-red-950 dark:text-red-200"
		>
			{{ numberError }}
		</p>
		<Field
			v-model="form.buyer_name"
			:label="t('link.public.name')"
			autocomplete="name"
			required
			:error="form.errors.buyer_name"
			data-testid="public-buyer-name"
		/>
		<Field
			v-model="form.buyer_email"
			:label="t('link.public.email')"
			type="email"
			autocomplete="email"
			required
			:error="form.errors.buyer_email"
			data-testid="public-buyer-email"
		/>
		<Field
			:model-value="form.buyer_phone"
			:label="t('link.public.phone')"
			type="tel"
			autocomplete="tel"
			inputmode="tel"
			required
			:error="form.errors.buyer_phone"
			data-testid="public-buyer-phone"
			@update:model-value="onPhone"
		/>
		<!-- Honeypot: hidden from people and assistive tech; bots fill it. -->
		<div class="hidden" aria-hidden="true">
			<label>Website <input v-model="form.website" type="text" tabindex="-1" autocomplete="off" name="website" /></label>
		</div>
		<label class="flex items-start gap-2 text-sm">
			<input v-model="form.consent" type="checkbox" class="mt-1" data-testid="public-buyer-consent" />
			<span>{{ t("link.public.consent") }}</span>
		</label>
		<p v-if="form.errors.consent" class="text-sm text-red-600 dark:text-red-400">{{ form.errors.consent }}</p>
		<PrimaryButton :disabled="form.processing" data-testid="public-buy-submit">
			{{ t("link.public.submit") }}
		</PrimaryButton>
	</form>
</template>
