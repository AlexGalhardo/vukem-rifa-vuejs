import { z } from "zod";

// Mirrors requests.ContactSchema (Go): the server stays the authority and
// stores WhatsApp in E.164 and Telegram without "@".
const brPhone = /^(\+?55[\s.-]?)?\(?[1-9][1-9]\)?[\s.-]?(9\d{4}|[2-8]\d{3})[\s.-]?\d{4}$/;
const telegram = /^@?[A-Za-z][A-Za-z0-9_]{4,31}$/;

export const contactSchema = z
	.object({
		show_email: z.boolean(),
		whatsapp: z
			.string()
			.trim()
			.refine((v) => v === "" || brPhone.test(v)),
		show_whatsapp: z.boolean(),
		telegram: z
			.string()
			.trim()
			.refine((v) => v === "" || telegram.test(v)),
		show_telegram: z.boolean(),
	})
	.refine((c) => !c.show_whatsapp || c.whatsapp !== "", { path: ["whatsapp"] })
	.refine((c) => !c.show_telegram || c.telegram !== "", { path: ["telegram"] });

export type ContactForm = z.input<typeof contactSchema>;
