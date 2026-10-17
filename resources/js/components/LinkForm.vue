<script setup lang="ts">
import { reactive, ref } from "vue";

import { useI18n } from "../composables/useI18n";
import { type BuyerForm, buyerSchema, maskPhone } from "../lib/buyerSchema";
import type { PaymentLink } from "../types";
import Field from "./Field.vue";
import PrimaryButton from "./PrimaryButton.vue";

const props = defineProps<{ raffleId: number; number: number }>();
const emit = defineEmits<{ created: [link: PaymentLink]; close: [] }>();

const { t } = useI18n();

type FieldKey = keyof BuyerForm;
const fieldKeys: FieldKey[] = ["buyer_name", "buyer_email", "buyer_phone"];
const invalid: Record<FieldKey, string> = {
	buyer_name: "link.form.invalid_name",
	buyer_email: "link.form.invalid_email",
	buyer_phone: "link.form.invalid_phone",
};

const form = reactive<BuyerForm>({ buyer_name: "", buyer_email: "", buyer_phone: "" });
const errors = reactive<Partial<Record<FieldKey, string>>>({});
const failure = ref("");
const busy = ref(false);

function isField(key: unknown): key is FieldKey {
	return typeof key === "string" && (fieldKeys as string[]).includes(key);
}

function onPhone(value: string): void {
	form.buyer_phone = maskPhone(value);
}

function isLink(v: unknown): v is PaymentLink {
	if (typeof v !== "object" || v === null) return false;
	const o = v as Record<string, unknown>;
	return typeof o.url === "string" && typeof o.expires_at === "string" && typeof o.number === "number";
}

// The link comes back once in the response body (fetch, not an Inertia
// visit) so the token never lands in the session or page history.
async function submit(): Promise<void> {
	for (const k of fieldKeys) delete errors[k];
	failure.value = "";

	const parsed = buyerSchema.safeParse(form);
	if (!parsed.success) {
		for (const issue of parsed.error.issues) {
			const key = issue.path[0];
			if (isField(key)) errors[key] = t(invalid[key]);
		}
		return;
	}

	busy.value = true;
	try {
		const res = await fetch(`/raffles/${props.raffleId}/numbers/${props.number}/link`, {
			method: "POST",
			headers: { "Content-Type": "application/json", Accept: "application/json" },
			body: JSON.stringify(parsed.data),
		});
		const body: unknown = await res.json().catch(() => null);

		if (res.status === 201 && isLink(body)) {
			emit("created", body);
			return;
		}
		if (res.status === 422 && typeof body === "object" && body !== null && "errors" in body) {
			for (const key of Object.keys((body as { errors: Record<string, unknown> }).errors)) {
				if (isField(key)) errors[key] = t(invalid[key]);
			}
			return;
		}
		if (res.status === 429) {
			failure.value = t("link.form.rate_limited");
			return;
		}
		const detail = typeof body === "object" && body !== null && "detail" in body ? String(body.detail) : "";
		failure.value = detail || t("link.form.error");
	} catch {
		failure.value = t("link.form.error");
	} finally {
		busy.value = false;
	}
}
</script>

<template>
	<form class="space-y-4" novalidate data-testid="link-form" @submit.prevent="submit">
		<div class="flex items-start justify-between gap-4">
			<h3 class="text-lg font-semibold">{{ t("link.form.title", { number }) }}</h3>
			<button
				type="button"
				class="rounded-md px-2 text-zinc-500 hover:bg-zinc-100 dark:hover:bg-zinc-800"
				:aria-label="t('link.form.close')"
				@click="emit('close')"
			>
				✕
			</button>
		</div>
		<Field
			v-model="form.buyer_name"
			:label="t('link.form.buyer_name')"
			:error="errors.buyer_name"
			autocomplete="off"
			maxlength="100"
			data-testid="buyer-name"
		/>
		<Field
			v-model="form.buyer_email"
			:label="t('link.form.buyer_email')"
			:error="errors.buyer_email"
			type="email"
			autocomplete="off"
			maxlength="255"
			data-testid="buyer-email"
		/>
		<div>
			<Field
				:model-value="form.buyer_phone"
				:label="t('link.form.buyer_phone')"
				:error="errors.buyer_phone"
				type="tel"
				inputmode="tel"
				autocomplete="off"
				placeholder="(11) 98765-4321"
				data-testid="buyer-phone"
				@update:model-value="onPhone"
			/>
			<p class="mt-1 text-xs text-zinc-500 dark:text-zinc-400">{{ t("link.form.phone_hint") }}</p>
		</div>
		<p v-if="failure" class="text-sm text-red-600 dark:text-red-400" role="alert" data-testid="link-error">{{ failure }}</p>
		<PrimaryButton :disabled="busy" data-testid="link-submit">{{ t("link.form.submit") }}</PrimaryButton>
	</form>
</template>
