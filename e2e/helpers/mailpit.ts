import { z } from "zod";

// Mailpit catches local SMTP mail (docker compose service "mailpit").
const MAILPIT_URL = process.env.MAILPIT_URL ?? "http://localhost:8025";

const searchSchema = z.object({
	messages: z.array(z.object({ ID: z.string(), Subject: z.string() })),
});
const messageSchema = z.object({ Text: z.string(), HTML: z.string() });

async function getJSON(path: string): Promise<unknown> {
	const res = await fetch(`${MAILPIT_URL}${path}`);
	if (!res.ok) throw new Error(`mailpit ${path}: ${res.status}`);
	return res.json();
}

/** Waits for the latest mail to `to` and returns its first link. */
export async function latestLinkTo(to: string, timeoutMs = 10_000): Promise<string> {
	const deadline = Date.now() + timeoutMs;
	while (Date.now() < deadline) {
		const { messages } = searchSchema.parse(
			await getJSON(`/api/v1/search?query=${encodeURIComponent(`to:${to}`)}`),
		);
		const latest = messages[0];
		if (latest) {
			const { Text } = messageSchema.parse(await getJSON(`/api/v1/message/${latest.ID}`));
			const link = Text.match(/https?:\/\/[^\s)"]+/)?.[0];
			if (link) return link;
		}
		await new Promise((resolve) => setTimeout(resolve, 250));
	}
	throw new Error(`no mail with a link to ${to} within ${timeoutMs}ms`);
}
