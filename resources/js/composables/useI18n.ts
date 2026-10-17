import { router, usePage } from "@inertiajs/vue3";
import { computed } from "vue";

import type { Locale, SharedProps } from "../types";

function interpolate(template: string, params: Record<string, string | number>): string {
	return template.replace(/\{(\w+)\}/g, (match, name: string) => (name in params ? String(params[name]) : match));
}

/**
 * Translates with the catalog the server shares (vukem inertia.I18n: flat
 * dotted keys from resources/lang, resolved for the request locale).
 */
export function useI18n() {
	const page = usePage<SharedProps>();
	const locale = computed<Locale>(() => page.props.locale ?? "pt-BR");

	function t(key: string, params: Record<string, string | number> = {}): string {
		const template = page.props.messages?.[key];
		return typeof template === "string" ? interpolate(template, params) : key;
	}

	// The Go i18n middleware reads the "lang" cookie on every request.
	function setLocale(next: Locale): void {
		document.cookie = `lang=${encodeURIComponent(next)}; path=/; max-age=31536000; samesite=lax`;
		document.documentElement.lang = next;
		router.reload();
	}

	return { locale, t, setLocale };
}
