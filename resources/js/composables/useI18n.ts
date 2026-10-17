import { router, usePage } from "@inertiajs/vue3";
import { computed } from "vue";

import en from "../../lang/en.json";
import ptBR from "../../lang/pt-BR.json";
import type { Locale, SharedProps } from "../types";

// The same JSON catalogs feed the Go i18n bundle (emails, validation).
const catalogs: Record<Locale, unknown> = { "pt-BR": ptBR, en };

function lookup(catalog: unknown, key: string): string | undefined {
	let node: unknown = catalog;
	for (const part of key.split(".")) {
		if (typeof node !== "object" || node === null) return undefined;
		node = (node as Record<string, unknown>)[part];
	}
	return typeof node === "string" ? node : undefined;
}

function interpolate(template: string, params: Record<string, string | number>): string {
	return template.replace(/\{(\w+)\}/g, (match, name: string) => (name in params ? String(params[name]) : match));
}

export function useI18n() {
	const page = usePage<SharedProps>();
	const locale = computed<Locale>(() => page.props.locale ?? "pt-BR");

	function t(key: string, params: Record<string, string | number> = {}): string {
		const template = lookup(catalogs[locale.value], key) ?? lookup(catalogs["pt-BR"], key) ?? key;
		return interpolate(template, params);
	}

	// The Go i18n middleware reads the "lang" cookie on every request.
	function setLocale(next: Locale): void {
		document.cookie = `lang=${encodeURIComponent(next)}; path=/; max-age=31536000; samesite=lax`;
		document.documentElement.lang = next;
		router.reload();
	}

	return { locale, t, setLocale };
}
