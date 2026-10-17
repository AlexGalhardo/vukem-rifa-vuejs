import { z } from "zod";

// Same pattern as brPhone in app/requests/buyer.go.
const brPhone = /^(\+?55[\s.-]?)?\(?[1-9][1-9]\)?[\s.-]?(9\d{4}|[2-8]\d{3})[\s.-]?\d{4}$/;

/** Mirrors requests.BuyerSchema (Go); the server stays the authority and normalizes the phone to E.164. */
export const buyerSchema = z.object({
	buyer_name: z.string().trim().min(2).max(100),
	buyer_email: z.string().trim().toLowerCase().max(255).pipe(z.email()),
	buyer_phone: z.string().trim().regex(brPhone),
});

export type BuyerForm = z.input<typeof buyerSchema>;

/** Formats typed digits as a Brazilian phone, "(11) 98765-4321" or "(21) 3333-4444". */
export function maskPhone(input: string): string {
	let d = input.replace(/\D/g, "");
	if (d.length > 11 && d.startsWith("55")) d = d.slice(2);
	d = d.slice(0, 11);

	if (d.length <= 2) return d.length ? `(${d}` : "";
	const area = `(${d.slice(0, 2)}) `;
	const rest = d.slice(2);
	const split = rest.length === 9 ? 5 : 4;
	if (rest.length <= split) return area + rest;
	return `${area}${rest.slice(0, split)}-${rest.slice(split)}`;
}
