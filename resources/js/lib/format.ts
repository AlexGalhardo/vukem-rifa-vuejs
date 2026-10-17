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

/** Formats integer centavos as BRL currency in the UI locale. */
export function formatCents(cents: number, locale: Locale): string {
	return new Intl.NumberFormat(locale, { style: "currency", currency: "BRL" }).format(cents / 100);
}

/** Formats seconds as m:ss. */
export function formatCountdown(seconds: number): string {
	const s = Math.max(0, Math.floor(seconds));
	return `${Math.floor(s / 60)}:${String(s % 60).padStart(2, "0")}`;
}
