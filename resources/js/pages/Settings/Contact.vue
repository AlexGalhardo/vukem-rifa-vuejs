<script setup lang="ts">
import { Head, useForm } from "@inertiajs/vue3";

import PrimaryButton from "../../components/PrimaryButton.vue";
import { useI18n } from "../../composables/useI18n";
import { maskPhone } from "../../lib/buyerSchema";
import { type ContactForm, contactSchema } from "../../lib/contactSchema";

const props = defineProps<{ contact: ContactForm & { email: string } }>();

const { t } = useI18n();
const form = useForm<ContactForm>({
	show_email: props.contact.show_email,
	whatsapp: props.contact.whatsapp ? maskPhone(props.contact.whatsapp) : "",
	show_whatsapp: props.contact.show_whatsapp,
	telegram: props.contact.telegram ? `@${props.contact.telegram}` : "",
	show_telegram: props.contact.show_telegram,
});
const input =
	"mt-1 block w-full rounded-md border border-zinc-300 bg-transparent px-3 py-2 focus:border-brand-500 focus:outline-none dark:border-zinc-700";

function onWhatsApp(e: Event): void {
	if (e.target instanceof HTMLInputElement) form.whatsapp = maskPhone(e.target.value);
}

function submit(): void {
	form.clearErrors();
	const parsed = contactSchema.safeParse(form.data());
	if (!parsed.success) {
		for (const issue of parsed.error.issues) {
			const key = issue.path[0];
			if (key === "whatsapp" || key === "telegram") form.setError(key, t(`contact.invalid_${key}`));
		}
		return;
	}
	form.put("/settings/contact", { preserveScroll: true });
}
</script>

<template>
	<Head :title="t('contact.title')" />
	<h1 class="mb-2 text-2xl font-semibold">{{ t("contact.title") }}</h1>
	<p class="mb-4 text-zinc-600 dark:text-zinc-400">{{ t("contact.lead") }}</p>
	<p
		class="mb-6 rounded-md bg-amber-50 px-3 py-2 text-sm text-amber-900 dark:bg-amber-950 dark:text-amber-100"
		data-testid="contact-privacy"
	>
		{{ t("contact.privacy") }}
	</p>

	<form class="max-w-lg space-y-6" novalidate data-testid="contact-form" @submit.prevent="submit">
		<fieldset class="space-y-2">
			<label class="flex items-center gap-2">
				<input v-model="form.show_email" type="checkbox" data-testid="contact-show-email" />
				<span>{{ t("contact.show_email", { email: contact.email }) }}</span>
			</label>
		</fieldset>

		<fieldset class="space-y-2">
			<label class="block">
				<span class="font-medium">{{ t("contact.whatsapp") }}</span>
				<input
					:value="form.whatsapp"
					type="tel"
					inputmode="tel"
					autocomplete="tel"
					placeholder="(11) 98765-4321"
					:class="input"
					data-testid="contact-whatsapp"
					@input="onWhatsApp"
				/>
			</label>
			<p v-if="form.errors.whatsapp" class="text-sm text-red-600 dark:text-red-400">{{ form.errors.whatsapp }}</p>
			<label class="flex items-center gap-2">
				<input v-model="form.show_whatsapp" type="checkbox" data-testid="contact-show-whatsapp" />
				<span>{{ t("contact.show_whatsapp") }}</span>
			</label>
		</fieldset>

		<fieldset class="space-y-2">
			<label class="block">
				<span class="font-medium">{{ t("contact.telegram") }}</span>
				<input
					v-model="form.telegram"
					type="text"
					autocomplete="off"
					placeholder="@usuario"
					:class="input"
					data-testid="contact-telegram"
				/>
			</label>
			<p v-if="form.errors.telegram" class="text-sm text-red-600 dark:text-red-400">{{ form.errors.telegram }}</p>
			<label class="flex items-center gap-2">
				<input v-model="form.show_telegram" type="checkbox" data-testid="contact-show-telegram" />
				<span>{{ t("contact.show_telegram") }}</span>
			</label>
		</fieldset>

		<PrimaryButton :disabled="form.processing" data-testid="contact-submit">{{ t("contact.save") }}</PrimaryButton>
	</form>
</template>
