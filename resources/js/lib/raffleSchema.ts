import { z } from "zod";

import type { RaffleLimits } from "../types";

/** Mirrors requests.RaffleSchema (Go); the server stays the authority. */
export function raffleSchema(l: RaffleLimits) {
	return z.object({
		title: z.string().trim().min(3).max(120),
		description: z.string().trim().max(5000),
		number_count: z.number().int().min(l.min_numbers).max(l.max_numbers),
		price_reais: z.number().int().min(l.min_price).max(l.max_price),
		duration_days: z.number().int().min(l.min_duration).max(l.max_duration),
	});
}

export type RaffleForm = z.infer<ReturnType<typeof raffleSchema>>;
