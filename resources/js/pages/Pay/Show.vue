<script setup lang="ts">
import { Head, Link, router, usePage } from "@inertiajs/vue3";
import { computed, onBeforeUnmount, ref, watch } from "vue";

import PrimaryButton from "../../components/PrimaryButton.vue";
import { useI18n } from "../../composables/useI18n";
import { formatCents, formatCountdown } from "../../lib/format";
import { payStatusSchema } from "../../lib/paySchema";
import type { Checkout, CheckoutState, Pix } from "../../types";

const props = defineProps<{
	state: CheckoutState;
	checkout: Checkout | null;
	pix: Pix | null;
	error: string | null;
}>();

const { t, locale } = useI18n();
const page = usePage();

// The page URL is /pay/{token}; the token is the only credential.
const base = computed(() => page.url.split("?")[0] ?? "");

const secondsLeft = ref(props.checkout?.seconds_left ?? 0);
const modalOpen = ref(props.pix !== null);
const copied = ref(false);
const confirming = ref(false);

watch(
	() => props.checkout?.seconds_left,
	(s) => {
		secondsLeft.value = s ?? 0;
	},
);
watch(
	() => props.pix,
	(p) => {
		modalOpen.value = p !== null;
	},
);

let tick: ReturnType<typeof setInterval> | undefined;
let poll: ReturnType<typeof setInterval> | undefined;

function stop(): void {
	clearInterval(tick);
	clearInterval(poll);
	tick = undefined;
	poll = undefined;
}

async function checkStatus(): Promise<void> {
	try {
		const res = await fetch(`${base.value}/status`, {
			headers: { Accept: "application/json" },
			credentials: "same-origin",
			cache: "no-store",
		});
		if (!res.ok) return;
		const parsed = payStatusSchema.safeParse(await res.json());
		if (parsed.success && (parsed.data.status === "paid" || parsed.data.status === "expired")) {
			stop();
			router.reload();
		}
	} catch {
		// Network hiccup: the next poll retries.
	}
}

watch(
	() => [props.state, props.pix] as const,
	([state, pix]) => {
		stop();
		if (state !== "live") return;
		tick = setInterval(() => {
			secondsLeft.value = Math.max(0, secondsLeft.value - 1);
			if (secondsLeft.value === 0 && !pix) {
				stop();
				router.reload();
			}
		}, 1000);
		if (pix) poll = setInterval(checkStatus, 3000);
	},
	{ immediate: true },
);

onBeforeUnmount(stop);

function confirm(): void {
	router.post(
		`${base.value}/confirm`,
		{},
		{
			preserveScroll: true,
			onStart: () => {
				confirming.value = true;
			},
			onFinish: () => {
				confirming.value = false;
			},
		},
	);
}

async function copy(): Promise<void> {
	if (!props.pix) return;
	try {
		await navigator.clipboard.writeText(props.pix.br_code);
		copied.value = true;
		setTimeout(() => {
			copied.value = false;
		}, 2000);
	} catch {
		copied.value = false;
	}
}

const card = "rounded-lg border border-zinc-200 bg-white p-6 shadow-sm dark:border-zinc-800 dark:bg-zinc-900";
</script>

<template>
	<Head :title="t('pay.title')">
		<meta name="referrer" content="no-referrer" />
		<meta name="robots" content="noindex" />
	</Head>

	<section class="mx-auto max-w-lg">
		<div v-if="state === 'expired' || !checkout" :class="card" data-testid="pay-expired" role="alert">
			<p class="text-lg font-medium">{{ t("pay.expired") }}</p>
		</div>

		<div v-else-if="state === 'paid'" :class="card" data-testid="pay-confirmed" role="status">
			<h1 class="text-2xl font-semibold text-brand-700 dark:text-brand-500">{{ t("pay.paid") }}</h1>
			<p class="mt-2 text-zinc-700 dark:text-zinc-300">
				{{ t("pay.paid_detail", { number: checkout.number, title: checkout.raffle_title }) }}
			</p>
			<Link
				:href="`/r/${checkout.raffle_slug}`"
				class="mt-4 inline-block text-brand-700 hover:underline dark:text-brand-500"
			>
				{{ t("pay.view_raffle") }}
			</Link>
		</div>

		<div v-else :class="card">
			<p class="text-sm text-zinc-600 dark:text-zinc-400">{{ t("pay.title") }}</p>
			<h1 class="mt-1 text-2xl font-semibold" data-testid="pay-raffle">{{ checkout.raffle_title }}</h1>
			<p class="mt-1 text-lg" data-testid="pay-number">{{ t("pay.number", { number: checkout.number }) }}</p>

			<dl class="mt-4 grid grid-cols-[auto_1fr] gap-x-4 gap-y-1 text-sm">
				<dt class="text-zinc-600 dark:text-zinc-400">{{ t("pay.amount") }}</dt>
				<dd class="font-medium" data-testid="pay-amount">{{ formatCents(checkout.amount_cents, locale) }}</dd>
				<dt class="text-zinc-600 dark:text-zinc-400">{{ t("pay.buyer") }}</dt>
				<dd>{{ checkout.buyer_name }}</dd>
				<dt class="text-zinc-600 dark:text-zinc-400">{{ t("pay.email") }}</dt>
				<dd>{{ checkout.buyer_email }}</dd>
				<dt class="text-zinc-600 dark:text-zinc-400">{{ t("pay.phone") }}</dt>
				<dd>{{ checkout.buyer_phone }}</dd>
			</dl>

			<p class="mt-4 text-sm text-amber-800 dark:text-amber-300" data-testid="pay-countdown" aria-live="polite">
				{{ t("pay.expires_in", { time: formatCountdown(secondsLeft) }) }}
			</p>

			<p
				v-if="error"
				role="alert"
				data-testid="pay-error"
				class="mt-4 rounded-md bg-red-50 px-3 py-2 text-sm text-red-800 dark:bg-red-950 dark:text-red-200"
			>
				{{ error }}
			</p>

			<div class="mt-6">
				<PrimaryButton v-if="!pix" type="button" data-testid="pay-confirm" :disabled="confirming" @click="confirm">
					{{ confirming ? t("pay.confirming") : t("pay.confirm") }}
				</PrimaryButton>
				<template v-else>
					<p class="text-sm text-zinc-600 dark:text-zinc-400" data-testid="pay-waiting">{{ t("pay.waiting") }}</p>
					<button
						v-if="!modalOpen"
						type="button"
						class="mt-2 text-brand-700 hover:underline dark:text-brand-500"
						data-testid="pay-show-qr"
						@click="modalOpen = true"
					>
						{{ t("pay.show_qr") }}
					</button>
				</template>
			</div>
		</div>
	</section>

	<div
		v-if="state === 'live' && pix && modalOpen"
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4"
		@click.self="modalOpen = false"
	>
		<div
			role="dialog"
			aria-modal="true"
			aria-labelledby="pix-title"
			data-testid="pix-modal"
			class="w-full max-w-sm rounded-lg bg-white p-6 text-zinc-900 shadow-xl dark:bg-zinc-900 dark:text-zinc-100"
		>
			<h2 id="pix-title" class="text-xl font-semibold">{{ t("pay.qr_title") }}</h2>
			<p class="mt-1 text-sm text-zinc-600 dark:text-zinc-400">{{ t("pay.qr_hint") }}</p>
			<img
				:src="pix.br_code_base64"
				:alt="t('pay.qr_alt')"
				data-testid="pix-qr"
				class="mx-auto mt-4 h-56 w-56 rounded bg-white p-2"
			/>
			<label class="mt-4 block text-sm">
				<span class="font-medium">{{ t("pay.copy_label") }}</span>
				<textarea
					:value="pix.br_code"
					readonly
					rows="3"
					data-testid="pix-code"
					class="mt-1 block w-full resize-none rounded-md border border-zinc-300 bg-zinc-50 px-2 py-1 font-mono text-xs break-all dark:border-zinc-700 dark:bg-zinc-800"
				/>
			</label>
			<div class="mt-4 flex items-center justify-between gap-2">
				<PrimaryButton type="button" data-testid="pix-copy" @click="copy">
					{{ copied ? t("pay.copied") : t("pay.copy") }}
				</PrimaryButton>
				<button
					type="button"
					class="text-sm text-zinc-600 hover:underline dark:text-zinc-400"
					data-testid="pix-close"
					@click="modalOpen = false"
				>
					{{ t("pay.close") }}
				</button>
			</div>
			<p class="mt-4 text-center text-sm text-zinc-600 dark:text-zinc-400" aria-live="polite">
				{{ t("pay.waiting") }} · {{ formatCountdown(secondsLeft) }}
			</p>
		</div>
	</div>
</template>
