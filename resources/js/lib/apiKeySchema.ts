import { z } from "zod";

import type { ApiAbility } from "../types";

export const API_ABILITIES = ["raffles:read", "raffles:write"] as const satisfies readonly ApiAbility[];

/** Mirrors requests.APIKeySchema (Go); the server stays the authority. */
export const apiKeySchema = z.object({
	name: z.string().trim().min(1).max(60),
	abilities: z
		.array(z.enum(API_ABILITIES))
		.min(1)
		.max(API_ABILITIES.length)
		.refine((list) => new Set(list).size === list.length),
});

export type ApiKeyForm = z.infer<typeof apiKeySchema>;
