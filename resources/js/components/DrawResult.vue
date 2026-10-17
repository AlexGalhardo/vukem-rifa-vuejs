<script setup lang="ts">
import { computed, ref } from "vue";

import { useI18n } from "../composables/useI18n";
import { participantsHash, participantsList, pickWinner } from "../lib/draw";
import { formatDateTime } from "../lib/format";
import type { DrawResult } from "../types";

// The auditable result: the same component on the public page and the
// owner's page. paid lists the participating numbers (the paid ones), so
// visitors can recompute the hash and the winner in the browser.
const props = defineProps<{ draw: DrawResult; paid: number[] }>();

const { t, locale } = useI18n();
const list = computed(() => participantsList(props.paid));
const verdict = ref<"ok" | "fail" | null>(null);

async function verify(): Promise<void> {
	const [hash, winner] = await Promise.all([participantsHash(props.paid), pickWinner(props.draw.seed, props.paid)]);
	verdict.value = hash === props.draw.participants_sha256 && winner === props.draw.winning_number ? "ok" : "fail";
}

const row = "grid gap-1 sm:grid-cols-[14rem_1fr]";
const label = "text-sm text-zinc-500 dark:text-zinc-400";
</script>

<template>
	<section
		class="rounded-lg border border-zinc-200 p-4 dark:border-zinc-800"
		aria-labelledby="draw-title"
		data-testid="draw-result"
	>
		<h2 id="draw-title" class="mb-3 text-lg font-semibold">{{ t("draw.title") }}</h2>
		<p v-if="draw.winning_number !== null" class="mb-4">
			<span class="block text-sm text-zinc-500 dark:text-zinc-400">{{ t("draw.winning_number") }}</span>
			<span class="text-4xl font-bold tabular-nums text-brand-700 dark:text-brand-500" data-testid="draw-winner">
				{{ draw.winning_number }}
			</span>
		</p>
		<p v-else class="mb-4" data-testid="draw-no-winner">{{ t("draw.no_winner") }}</p>
		<slot />
		<dl class="space-y-2 text-sm">
			<div :class="row">
				<dt :class="label">{{ t("draw.drawn_at") }}</dt>
				<dd>{{ formatDateTime(draw.drawn_at, locale) }}</dd>
			</div>
			<div :class="row">
				<dt :class="label">{{ t("draw.trigger_label") }}</dt>
				<dd>{{ t(`draw.trigger.${draw.trigger}`) }}</dd>
			</div>
			<div :class="row">
				<dt :class="label">{{ t("draw.algorithm") }}</dt>
				<dd>{{ draw.algorithm }}</dd>
			</div>
			<div :class="row">
				<dt :class="label">{{ t("draw.seed") }}</dt>
				<dd class="font-mono break-all" data-testid="draw-seed">{{ draw.seed }}</dd>
			</div>
			<div :class="row">
				<dt :class="label">{{ t("draw.participants_count") }}</dt>
				<dd data-testid="draw-count">{{ draw.participants_count }}</dd>
			</div>
			<div v-if="list" :class="row">
				<dt :class="label">{{ t("draw.participants_list") }}</dt>
				<dd class="font-mono break-all" data-testid="draw-list">{{ list }}</dd>
			</div>
			<div :class="row">
				<dt :class="label">{{ t("draw.participants_sha256") }}</dt>
				<dd class="font-mono break-all" data-testid="draw-hash">{{ draw.participants_sha256 }}</dd>
			</div>
		</dl>
		<details class="mt-4 text-sm" data-testid="draw-verify">
			<summary class="cursor-pointer font-medium">{{ t("draw.verify.title") }}</summary>
			<div class="mt-2 space-y-2 text-zinc-700 dark:text-zinc-300">
				<p>{{ t("draw.verify.intro") }}</p>
				<ol class="list-decimal space-y-1 pl-5">
					<li>{{ t("draw.verify.step1") }}</li>
					<li>{{ t("draw.verify.step2") }}</li>
					<li>{{ t("draw.verify.step3") }}</li>
				</ol>
				<p class="text-zinc-500 dark:text-zinc-400">{{ t("draw.verify.docs") }}</p>
				<div v-if="draw.winning_number !== null" class="flex flex-wrap items-center gap-3">
					<button
						type="button"
						class="rounded-md border border-zinc-300 px-3 py-1 hover:bg-zinc-100 dark:border-zinc-700 dark:hover:bg-zinc-800"
						data-testid="draw-verify-run"
						@click="verify"
					>
						{{ t("draw.verify.run") }}
					</button>
					<span
						v-if="verdict"
						role="status"
						:class="verdict === 'ok' ? 'text-brand-700 dark:text-brand-500' : 'text-red-600 dark:text-red-400'"
						data-testid="draw-verify-result"
					>
						{{ verdict === "ok" ? t("draw.verify.ok") : t("draw.verify.fail") }}
					</span>
				</div>
			</div>
		</details>
	</section>
</template>
