<script setup lang="ts">
import { Head, Link, useForm } from "@inertiajs/vue3";

import PrimaryButton from "../../components/PrimaryButton.vue";
import { useI18n } from "../../composables/useI18n";
import { formatCents, formatDateTime } from "../../lib/format";
import type { RefundDue } from "../../types";

const props = defineProps<{ pending: RefundDue[]; done: RefundDue[] }>();

const { t, locale } = useI18n();

const forms = Object.fromEntries(props.pending.map((r) => [r.id, useForm({ reference: "" })]));

function submit(r: RefundDue): void {
	const form = forms[r.id];
	if (!form) return;
	form.clearErrors();
	if (form.reference.trim() === "") {
		form.setError("reference", t("admin.payouts.invalid"));
		return;
	}
	form.post(`/admin/refunds/${r.id}`, { preserveScroll: true });
}

const input =
	"block w-full rounded-md border border-zinc-300 bg-transparent px-2 py-1 focus:border-brand-500 focus:outline-none dark:border-zinc-700";
</script>

<template>
	<Head :title="t('admin.refunds.title')" />
	<h1 class="text-2xl font-semibold">{{ t("admin.refunds.title") }}</h1>
	<p class="mt-1 text-sm text-zinc-600 dark:text-zinc-400">
		{{ t("admin.refunds.intro") }}
	</p>
	<p class="mt-1 mb-6 text-sm">
		<Link href="/admin/payouts" class="text-brand-700 underline dark:text-brand-500">{{
			t("admin.payouts.title")
		}}</Link>
	</p>

	<h2 class="mb-3 text-lg font-semibold">{{ t("admin.refunds.pending") }}</h2>
	<p v-if="pending.length === 0" class="text-zinc-600 dark:text-zinc-400" data-testid="refunds-empty">
		{{ t("admin.refunds.empty") }}
	</p>
	<ul v-else class="space-y-4" data-testid="refunds-pending">
		<li
			v-for="r in pending"
			:key="r.id"
			class="rounded-lg border border-zinc-200 p-4 dark:border-zinc-800"
			:data-testid="`refund-${r.id}`"
		>
			<dl class="grid gap-x-4 gap-y-1 text-sm sm:grid-cols-2">
				<div>
					<dt class="inline font-medium">{{ t("admin.refunds.raffle") }}:</dt>
					<dd class="inline">
						{{ r.raffle_title }} · {{ t("admin.refunds.number") }}
						{{ r.number }}
					</dd>
				</div>
				<div>
					<dt class="inline font-medium">{{ t("admin.refunds.amount") }}:</dt>
					<dd class="inline font-semibold">
						{{ formatCents(r.amount_cents, locale) }}
					</dd>
				</div>
				<div>
					<dt class="inline font-medium">{{ t("admin.refunds.buyer") }}:</dt>
					<dd class="inline break-all">{{ r.buyer_email }}</dd>
				</div>
				<div>
					<dt class="inline font-medium">{{ t("admin.refunds.created_at") }}:</dt>
					<dd class="inline">
						{{ formatDateTime(r.created_at, locale) }}
					</dd>
				</div>
				<div class="sm:col-span-2">
					<dd class="break-all text-zinc-600 dark:text-zinc-400">
						{{ r.note }}
					</dd>
				</div>
			</dl>
			<form class="mt-3 grid gap-3 sm:grid-cols-[1fr_auto] sm:items-end" novalidate @submit.prevent="submit(r)">
				<label class="block text-sm">
					<span class="font-medium">{{ t("admin.refunds.reference") }}</span>
					<input
						v-model="forms[r.id]!.reference"
						data-testid="refund-reference"
						maxlength="255"
						:class="input"
						required
					/>
				</label>
				<PrimaryButton :disabled="forms[r.id]?.processing" data-testid="refund-submit">
					{{ t("admin.refunds.submit") }}
				</PrimaryButton>
				<p v-if="forms[r.id]?.errors.reference" class="text-sm text-red-600 sm:col-span-2 dark:text-red-400">
					{{ forms[r.id]?.errors.reference }}
				</p>
			</form>
		</li>
	</ul>

	<template v-if="done.length > 0">
		<h2 class="mt-8 mb-3 text-lg font-semibold">
			{{ t("admin.refunds.done") }}
		</h2>
		<ul class="space-y-2 text-sm" data-testid="refunds-done">
			<li v-for="r in done" :key="r.id" class="rounded-lg border border-zinc-200 p-3 dark:border-zinc-800">
				{{ r.raffle_title }} · {{ t("admin.refunds.number") }} {{ r.number }} ·
				{{ formatCents(r.amount_cents, locale) }} ·
				{{ t("admin.refunds.refunded_at") }}
				{{ r.refunded_at ? formatDateTime(r.refunded_at, locale) : "" }}
				· {{ r.reference }}
			</li>
		</ul>
	</template>
</template>
