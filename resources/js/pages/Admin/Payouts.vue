<script setup lang="ts">
import { Head, Link, useForm } from "@inertiajs/vue3";
import { reactive } from "vue";

import PrimaryButton from "../../components/PrimaryButton.vue";
import { useI18n } from "../../composables/useI18n";
import { formatCents } from "../../lib/format";
import { payoutSchema } from "../../lib/paySchema";
import type { OwnerBalance } from "../../types";

const props = defineProps<{ owners: OwnerBalance[]; max_cents: number }>();

const { t, locale } = useI18n();

// One form per owner; the amount is typed in reais and sent in centavos.
const amounts = reactive<Record<number, string>>({});
const forms = Object.fromEntries(
	props.owners.map((o) => [o.user_id, useForm({ user_id: o.user_id, amount_cents: 0, reference: "" })]),
);

function toCents(reais: string): number {
	const n = Number(reais.replace(",", "."));
	return Number.isFinite(n) ? Math.round(n * 100) : Number.NaN;
}

function submit(owner: OwnerBalance): void {
	const form = forms[owner.user_id];
	if (!form) return;
	form.clearErrors();
	form.amount_cents = toCents(amounts[owner.user_id] ?? "");
	const result = payoutSchema(owner.balance_cents, props.max_cents).safeParse(form.data());
	if (!result.success) {
		for (const issue of result.error.issues) {
			const key = issue.path[0];
			if (key === "amount_cents" || key === "reference") {
				form.setError(
					key,
					key === "amount_cents" && form.amount_cents > owner.balance_cents
						? t("admin.payouts.exceeds_balance")
						: t("admin.payouts.invalid"),
				);
			}
		}
		return;
	}
	form.post("/admin/payouts", {
		preserveScroll: true,
		onSuccess: () => {
			amounts[owner.user_id] = "";
			form.reset();
		},
	});
}

const input =
	"block w-full rounded-md border border-zinc-300 bg-transparent px-2 py-1 focus:border-brand-500 focus:outline-none dark:border-zinc-700";
</script>

<template>
	<Head :title="t('admin.payouts.title')" />
	<h1 class="text-2xl font-semibold">{{ t("admin.payouts.title") }}</h1>
	<p class="mt-1 text-sm text-zinc-600 dark:text-zinc-400">{{ t("admin.payouts.intro") }}</p>
	<p class="mt-1 mb-6 text-sm">
		<Link href="/admin/refunds" class="text-brand-700 underline dark:text-brand-300" data-testid="link-refunds">
			{{ t("admin.refunds.title") }}
		</Link>
	</p>

	<p v-if="owners.length === 0" class="text-zinc-600 dark:text-zinc-400" data-testid="payouts-empty">
		{{ t("admin.payouts.empty") }}
	</p>
	<ul v-else class="space-y-4" data-testid="payouts-list">
		<li
			v-for="o in owners"
			:key="o.user_id"
			class="rounded-lg border border-zinc-200 p-4 dark:border-zinc-800"
			:data-testid="`payout-owner-${o.user_id}`"
		>
			<div class="flex flex-wrap items-baseline justify-between gap-2">
				<p>
					<span class="font-medium">{{ o.name }}</span>
					<span class="text-sm text-zinc-600 dark:text-zinc-400"> · {{ o.email }}</span>
				</p>
				<p class="text-sm">
					{{ t("admin.payouts.balance") }}:
					<span class="font-semibold" data-testid="payout-balance">{{ formatCents(o.balance_cents, locale) }}</span>
				</p>
			</div>
			<form class="mt-3 grid gap-3 sm:grid-cols-[10rem_1fr_auto] sm:items-end" novalidate @submit.prevent="submit(o)">
				<label class="block text-sm">
					<span class="font-medium">{{ t("admin.payouts.amount") }}</span>
					<input
						v-model="amounts[o.user_id]"
						inputmode="decimal"
						data-testid="payout-amount"
						:class="input"
						required
					/>
				</label>
				<label class="block text-sm">
					<span class="font-medium">{{ t("admin.payouts.reference") }}</span>
					<input
						v-model="forms[o.user_id]!.reference"
						data-testid="payout-reference"
						maxlength="255"
						:class="input"
						required
					/>
				</label>
				<PrimaryButton :disabled="forms[o.user_id]?.processing" data-testid="payout-submit">
					{{ t("admin.payouts.submit") }}
				</PrimaryButton>
				<p v-if="forms[o.user_id]?.errors.amount_cents" class="text-sm text-red-600 sm:col-span-3 dark:text-red-400">
					{{ forms[o.user_id]?.errors.amount_cents }}
				</p>
				<p v-if="forms[o.user_id]?.errors.reference" class="text-sm text-red-600 sm:col-span-3 dark:text-red-400">
					{{ forms[o.user_id]?.errors.reference }}
				</p>
			</form>
		</li>
	</ul>
</template>
