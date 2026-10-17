import type { Locale } from "../types";

/** Formats whole reais as BRL currency in the UI locale. */
export function formatReais(reais: number, locale: Locale): string {
	return new Intl.NumberFormat(locale, { style: "currency", currency: "BRL", maximumFractionDigits: 0 }).format(
		reais,
	);
}

/** Formats an RFC 3339 timestamp as a date and time in the UI locale (Brasília time). */
export function formatDateTime(iso: string, locale: Locale): string {
	return new Intl.DateTimeFormat(locale, {
		dateStyle: "medium",
		timeStyle: "short",
		timeZone: "America/Sao_Paulo",
	}).format(new Date(iso));
}
