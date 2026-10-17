<script setup lang="ts">
import { computed, ref } from "vue";

import { useI18n } from "../composables/useI18n";
import { formatDateTime } from "../lib/format";
import type { Organizer } from "../types";

const props = defineProps<{ organizer: Organizer; createdAt: string }>();

const { t, locale } = useI18n();
// The email comes split in two and is joined only on a click, so scrapers
// reading the HTML never get the address.
const revealed = ref(false);
const email = computed(() =>
	props.organizer.email ? `${props.organizer.email.user}@${props.organizer.email.domain}` : "",
);
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
			v-if="organizer.email || organizer.whatsapp || organizer.telegram"
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
			<li v-if="organizer.email">
				<a v-if="revealed" :href="`mailto:${email}`" rel="nofollow" :class="link" data-testid="organizer-email">{{
					email
				}}</a>
				<button v-else type="button" :class="link" data-testid="organizer-email-reveal" @click="revealed = true">
					{{ t("organizer.show_email") }}
				</button>
			</li>
		</ul>
	</section>
</template>
