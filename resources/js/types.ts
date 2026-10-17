// Shapes of the props the Go controllers send.

export type Locale = "pt-BR" | "en";

export interface User {
	id: number;
	name: string;
	email: string;
	email_verified_at: string | null;
}

export interface SharedProps {
	appName: string;
	locale: Locale;
	locales: Locale[];
	// Shared by vukem inertia.I18n (flat dotted keys; plural entries are objects).
	messages?: Record<string, string | Record<string, string>>;
	errors: Record<string, string>;
	// Present after vukem install:auth.
	auth?: { user: User | null };
	[key: string]: unknown;
}

export type RaffleStatus = "active" | "drawn" | "closed" | "archived";
export type NumberStatus = "available" | "reserved" | "paid";

/** Fields anyone may see (public page). */
export interface PublicRaffle {
	slug: string;
	title: string;
	description: string;
	number_count: number;
	price_reais: number;
	starts_at: string;
	ends_at: string;
	status: RaffleStatus;
}

/** The owner's view of a raffle. */
export interface OwnedRaffle extends PublicRaffle {
	id: number;
	duration_days: number;
	locked: boolean;
	created_at: string;
}

export interface RaffleSummary extends PublicRaffle {
	id: number;
	duration_days: number;
	locked_at: string | null;
	available: number;
	reserved: number;
	paid: number;
}

export interface RaffleNumber {
	id: number;
	number: number;
	status: NumberStatus;
	paid_at: string | null;
}

export interface RaffleLimits {
	min_numbers: number;
	max_numbers: number;
	min_price: number;
	max_price: number;
	min_duration: number;
	max_duration: number;
}
