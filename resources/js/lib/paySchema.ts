import { z } from "zod";

/** Mirrors GET /pay/{token}/status (Go PayController.Status). */
export const payStatusSchema = z.object({
	status: z.enum(["pending", "awaiting_payment", "paid", "expired"]),
});

export type PayStatus = z.infer<typeof payStatusSchema>["status"];

/** Mirrors requests.PayoutSchema (Go); the balance check stays on the server too. */
export function payoutSchema(balanceCents: number, maxCents: number) {
	return z.object({
		user_id: z.number().int().positive(),
		amount_cents: z.number().int().positive().max(Math.min(balanceCents, maxCents)),
		reference: z.string().trim().min(1).max(255),
	});
}
