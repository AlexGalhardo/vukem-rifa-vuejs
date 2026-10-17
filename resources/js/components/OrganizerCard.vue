<script setup lang="ts">
import { computed, ref } from "vue";

import { useI18n } from "../composables/useI18n";
import { formatDateTime } from "../lib/format";
import type { Organizer } from "../types";

const props = defineProps<{ organizer: Organizer; slug: string; createdAt: string }>();

const { t, locale } = useI18n();
// The address is not in the page at all: a click asks the server for it
// (rate limited), so scrapers reading the HTML or props never get it.
const email = ref("");
const busy = ref(false);
const failed = ref(false);

async function reveal(): Promise<void> {
	busy.value = true;
	failed.value = false;
	try {
		const res = await fetch(`/r/${encodeURIComponent(props.slug)}/organizer-email`, {
			method: "POST",
			headers: { Accept: "application/json" },
		});
		const body: unknown = await res.json().catch(() => null);
		if (res.ok && typeof body === "object" && body !== null && "email" in body && typeof body.email === "string") {
			email.value = body.email;
			return;
		}
		failed.value = true;
	} catch {
		failed.value = true;
	} finally {
		busy.value = false;
	}
}
const since = computed(() =>
	new Intl.DateTimeFormat(locale.value, { month: "long", year: "numeric" }).format(
		new Date(props.organizer.member_since),
	),
);
const link = "text-brand-700 underline hover:no-underline dark:text-brand-500";
</script>

<template>
	<section class="rounded-lg border border-zinc-200 p-4 dark:border-zinc-800" data-testid="organizer-card">
		<h2 class="text-sm font-medium text-zinc-600 dark:text-zinc-400">{{ t("organizer.title") }}</h2>
		<p class="mt-1 text-lg font-semibold" data-testid="organizer-name">{{ organizer.name }}</p>
		<ul class="mt-2 space-y-1 text-sm text-zinc-700 dark:text-zinc-300">
			<li>{{ t("organizer.member_since", { date: since }) }}</li>
			<li data-testid="organizer-raffles">
				{{ t("organizer.raffles", { total: organizer.raffles_total, finished: organizer.raffles_finished }) }}
			</li>
			<li>{{ t("organizer.created_at", { date: formatDateTime(createdAt, locale) }) }}</li>
		</ul>
		<ul
			v-if="organizer.has_email || organizer.whatsapp || organizer.telegram"
			class="mt-3 flex flex-wrap gap-3 text-sm"
			data-testid="organizer-contact"
		>
			<li v-if="organizer.whatsapp">
				<a
					:href="organizer.whatsapp.url"
					target="_blank"
					rel="noopener noreferrer nofollow"
					:class="link"
					data-testid="organizer-whatsapp"
				>
					WhatsApp {{ organizer.whatsapp.display }}
				</a>
			</li>
			<li v-if="organizer.telegram">
				<a
					:href="organizer.telegram.url"
					target="_blank"
					rel="noopener noreferrer nofollow"
					:class="link"
					data-testid="organizer-telegram"
				>
					Telegram {{ organizer.telegram.display }}
				</a>
			</li>
			<li v-if="organizer.has_email">
				<a v-if="email" :href="`mailto:${email}`" rel="nofollow" :class="link" data-testid="organizer-email">{{
					email
				}}</a>
				<button
					v-else
					type="button"
					:class="link"
					:disabled="busy"
					data-testid="organizer-email-reveal"
					@click="reveal"
				>
					{{ t("organizer.show_email") }}
				</button>
				<span v-if="failed" role="alert" class="ml-2 text-red-600 dark:text-red-400">{{
					t("organizer.email_failed")
				}}</span>
			</li>
		</ul>
	</section>
</template>
