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
	errors: Record<string, string>;
	// Present after vukem install:auth.
	auth?: { user: User | null };
	[key: string]: unknown;
}
