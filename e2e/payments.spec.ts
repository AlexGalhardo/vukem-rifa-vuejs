import { expect, type Page, test } from "@playwright/test";
import { z } from "zod";

import { registerVerifiedOwner, uniqueEmail } from "./helpers/owner";

// Runs against an app started with ABACATEPAY_FAKE=true (compose.yaml): the
// fake gateway serves deterministic PIX charges and the /dev helpers below.

const EXPIRED = "Link expirado. Peça ao dono da rifa para gerar um novo link e te enviar novamente para pagar";
const MAILPIT_URL = process.env.MAILPIT_URL ?? "http://localhost:8025";

const linkSchema = z.object({ token: z.string().min(43), url: z.string().startsWith("/pay/") });
const outcomeSchema = z.object({ outcome: z.string() });
const mailSearchSchema = z.object({ messages: z.array(z.object({ ID: z.string(), Subject: z.string() })) });

async function createRaffle(page: Page): Promise<number> {
	await page.getByTestId("new-raffle").click();
	await page.getByTestId("raffle-title").fill("Rifa do PIX");
	await page.getByTestId("raffle-description").fill("Teste de pagamento");
	await page.getByTestId("raffle-number-count").fill("10");
	await page.getByTestId("raffle-price").fill("5");
	await page.getByTestId("raffle-duration").fill("7");
	await page.getByTestId("raffle-submit").click();
	await expect(page).toHaveURL(/\/raffles\/\d+$/);
	return Number(new URL(page.url()).pathname.split("/").pop());
}

// Stands in for the owner's "generate link" action (POST
// /raffles/{id}/numbers/{n}/link, raffles stream) through the fake-gateway
// helper, with the owner's session.
async function reserve(page: Page, raffleId: number, number: number, buyerEmail: string) {
	const res = await page.request.post(`/dev/raffles/${raffleId}/numbers/${number}/reserve`, {
		data: { buyer_name: "Maria Compradora", buyer_email: buyerEmail, buyer_phone: "(11) 98765-4321" },
	});
	expect(res.status()).toBe(201);
	return linkSchema.parse(await res.json());
}

async function subjectsTo(to: string): Promise<string[]> {
	const res = await fetch(`${MAILPIT_URL}/api/v1/search?query=${encodeURIComponent(`to:${to}`)}`);
	if (!res.ok) return [];
	return mailSearchSchema.parse(await res.json()).messages.map((m) => m.Subject);
}

test("buyer confirms a payment link, sees the PIX QR code and the confirmation", async ({ page, browser }) => {
	const owner = uniqueEmail("pay-owner");
	const buyerEmail = uniqueEmail("pay-buyer");
	await registerVerifiedOwner(page, owner);
	const raffleId = await createRaffle(page);
	const link = await reserve(page, raffleId, 7, buyerEmail);

	// The buyer has no account: a fresh browser context with only the link.
	const buyerCtx = await browser.newContext();
	const buyer = await buyerCtx.newPage();
	const opened = await buyer.goto(link.url);
	expect(opened?.status()).toBe(200);
	expect(opened?.headers()["cache-control"]).toBe("no-store");
	expect(opened?.headers()["referrer-policy"]).toBe("no-referrer");

	await expect(buyer.getByTestId("pay-raffle")).toHaveText("Rifa do PIX");
	await expect(buyer.getByTestId("pay-number")).toHaveText("Número 7");
	await expect(buyer.getByTestId("pay-amount")).toContainText("5,00");
	await expect(buyer.getByText("p•••@example.com")).toBeVisible();
	expect(await buyer.content()).not.toContain(buyerEmail);
	await expect(buyer.getByTestId("pay-countdown")).toContainText(/O link expira em \d+:\d\d/);

	await buyer.getByTestId("pay-confirm").click();
	const modal = buyer.getByTestId("pix-modal");
	await expect(modal).toBeVisible();
	await expect(buyer.getByTestId("pix-qr")).toHaveAttribute("src", /^data:image\/png;base64,/);
	await expect(buyer.getByTestId("pix-code")).toHaveValue(/^000201/);

	// Closing and reopening keeps the same charge.
	const code = await buyer.getByTestId("pix-code").inputValue();
	await buyer.getByTestId("pix-close").click();
	await expect(modal).toBeHidden();
	await buyer.getByTestId("pay-show-qr").click();
	await expect(buyer.getByTestId("pix-code")).toHaveValue(code);

	// AbacatePay dev mode "simulate payment" → transparent.completed.
	const paid = await buyer.request.post(`/dev${new URL(link.url, "http://x").pathname}/simulate`);
	expect(paid.status()).toBe(200);
	expect(outcomeSchema.parse(await paid.json()).outcome).toBe("paid");

	// The page polls the status every 3 s.
	await expect(buyer.getByTestId("pay-confirmed")).toContainText("Pagamento confirmado", { timeout: 15_000 });
	await expect(buyer.getByTestId("pix-modal")).toBeHidden();

	// The public grid shows the number as paid, without the buyer's data.
	await buyer.getByRole("link", { name: "Ver a rifa" }).click();
	await expect(buyer.getByTestId("number-grid").locator('li[data-status="paid"]')).toHaveCount(1);
	expect(await buyer.content()).not.toContain(buyerEmail);

	// The owner's balance is 90% of R$ 5,00.
	await page.goto("/dashboard");
	await expect(page.getByTestId("owner-balance")).toContainText("4,50");

	// Queued confirmation emails reach Mailpit.
	await expect.poll(() => subjectsTo(buyerEmail), { timeout: 15_000 }).toContain("Pagamento confirmado: número 7");
	await expect.poll(() => subjectsTo(owner), { timeout: 15_000 }).toContain("Número 7 vendido");

	await buyerCtx.close();
});

test("an unknown or expired payment link shows the expired message", async ({ page }) => {
	const res = await page.goto("/pay/this-link-does-not-exist");
	expect(res?.status()).toBe(404);
	await expect(page.getByTestId("pay-expired")).toHaveText(EXPIRED);
	await expect(page.getByTestId("pay-confirm")).toHaveCount(0);
});

test("the expired message is translated", async ({ browser }) => {
	const ctx = await browser.newContext({ locale: "en" });
	const page = await ctx.newPage();
	await page.goto("/pay/this-link-does-not-exist");
	await expect(page.getByTestId("pay-expired")).toHaveText(
		"Link expired. Ask the raffle owner to generate a new link and send it to you again to pay",
	);
	await ctx.close();
});
