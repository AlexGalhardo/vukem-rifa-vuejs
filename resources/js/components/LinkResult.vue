<script setup lang="ts">
import { computed, onBeforeUnmount, ref } from "vue";

import { useI18n } from "../composables/useI18n";
import type { PaymentLink } from "../types";

const props = defineProps<{ link: PaymentLink; raffleTitle: string }>();
const emit = defineEmits<{ close: [] }>();

const { t } = useI18n();
const now = ref(Date.now());
const copied = ref(false);
const timer = window.setInterval(() => {
	now.value = Date.now();
}, 1000);
onBeforeUnmount(() => window.clearInterval(timer));

const remaining = computed(() => Math.max(0, Math.floor((Date.parse(props.link.expires_at) - now.value) / 1000)));
const countdown = computed(() => {
	const s = remaining.value;
	return `${Math.floor(s / 60)}:${String(s % 60).padStart(2, "0")}`;
});
const whatsapp = computed(() => {
	const text = t("link.result.whatsapp_text", {
		name: props.link.buyer_name,
		number: props.link.number,
		title: props.raffleTitle,
		url: props.link.url,
	});
	return `https://wa.me/${props.link.buyer_phone.replace(/\D/g, "")}?text=${encodeURIComponent(text)}`;
});

async function copy(): Promise<void> {
	try {
		await navigator.clipboard.writeText(props.link.url);
		copied.value = true;
		window.setTimeout(() => {
			copied.value = false;
		}, 2000);
	} catch {
		// Clipboard blocked: the field stays selectable for a manual copy.
	}
}

function selectAll(e: Event): void {
	if (e.target instanceof HTMLInputElement) e.target.select();
}
</script>

<template>
	<section class="space-y-3" data-testid="link-result">
		<div class="flex items-start justify-between gap-4">
			<h3 class="text-lg font-semibold">{{ t("link.result.title", { number: link.number }) }}</h3>
			<button
				type="button"
				class="rounded-md px-2 text-zinc-600 dark:text-zinc-400 hover:bg-zinc-100 dark:hover:bg-zinc-800"
				:aria-label="t('link.form.close')"
				@click="emit('close')"
			>
				✕
			</button>
		</div>
		<p class="rounded-md bg-amber-50 px-3 py-2 text-sm text-amber-900 dark:bg-amber-950 dark:text-amber-100">
			{{ t("link.result.once") }}
		</p>
		<input
			:value="link.url"
			readonly
			:aria-label="t('link.result.url')"
			class="block w-full rounded-md border border-zinc-300 bg-zinc-50 px-3 py-2 font-mono text-sm dark:border-zinc-700 dark:bg-zinc-900"
			data-testid="link-url"
			@focus="selectAll"
		/>
		<div class="flex flex-wrap items-center gap-2">
			<button
				type="button"
				class="rounded-md bg-brand-600 px-4 py-2 font-medium text-white hover:bg-brand-700"
				data-testid="link-copy"
				@click="copy"
			>
				{{ copied ? t("link.result.copied") : t("link.result.copy") }}
			</button>
			<a
				:href="whatsapp"
				target="_blank"
				rel="noopener noreferrer"
				class="rounded-md border border-zinc-300 px-4 py-2 font-medium hover:bg-zinc-100 dark:border-zinc-700 dark:hover:bg-zinc-800"
				data-testid="link-whatsapp"
			>
				{{ t("link.result.whatsapp") }}
			</a>
			<span
				class="ml-auto text-sm tabular-nums"
				:class="remaining > 0 ? 'text-zinc-600 dark:text-zinc-400' : 'text-red-600 dark:text-red-400'"
				data-testid="link-countdown"
				aria-live="off"
			>
				{{ remaining > 0 ? t("link.result.expires_in", { time: countdown }) : t("link.result.expired") }}
			</span>
		</div>
	</section>
</template>
