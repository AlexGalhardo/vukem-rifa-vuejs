import { expect, type Page, test } from "@playwright/test";

import { registerVerifiedOwner, uniqueEmail } from "./helpers/owner";

async function createRaffle(page: Page, title: string): Promise<void> {
	await page.getByTestId("new-raffle").click();
	await page.getByTestId("raffle-title").fill(title);
	await page.getByTestId("raffle-number-count").fill("20");
	await page.getByTestId("raffle-price").fill("5");
	await page.getByTestId("raffle-duration").fill("7");
	await page.getByTestId("raffle-submit").click();
	await expect(page).toHaveURL(/\/raffles\/\d+$/);
}

test("owner generates a payment link once, sees the buyer and cancels it", async ({ page, browser }) => {
	await registerVerifiedOwner(page, uniqueEmail("links"));
	await createRaffle(page, "Rifa do videogame");

	const xss = `<img src=x onerror="alert(1)">Ana`;
	const dialogs: string[] = [];
	page.on("dialog", (d) => {
		if (d.type() === "alert") dialogs.push(d.message());
	});

	await page.getByTestId("number-7").click();
	// Client validation mirrors the server: no request for an invalid phone.
	await page.getByTestId("buyer-name").fill(xss);
	await page.getByTestId("buyer-email").fill("Ana@Example.com");
	await page.getByTestId("buyer-phone").fill("123");
	await page.getByTestId("link-submit").click();
	await expect(page.getByText("Informe um telefone brasileiro com DDD.")).toBeVisible();

	await page.getByTestId("buyer-phone").fill("11987654321");
	await expect(page.getByTestId("buyer-phone")).toHaveValue("(11) 98765-4321");
	await page.getByTestId("link-submit").click();

	const url = page.getByTestId("link-url");
	await expect(url).toHaveValue(/\/pay\/[A-Za-z0-9_-]{43}$/);
	const link = await url.inputValue();
	await expect(page.getByTestId("link-countdown")).toContainText(/Expira em (10:00|9:\d\d)/);
	await expect(page.getByTestId("link-whatsapp")).toHaveAttribute("href", /^https:\/\/wa\.me\/5511987654321\?text=/);
	await expect(page.locator('[data-testid="number-7"][aria-pressed="true"]')).toBeVisible();
	await expect(page.locator('li[data-number="7"]')).toHaveAttribute("data-status", "reserved");

	// The link is shown once: after a reload it is gone for good.
	await page.reload();
	expect(await page.content()).not.toContain(link.split("/pay/")[1]);

	// The owner sees the buyer, rendered as text (no HTML injection).
	await page.getByTestId("number-7").click();
	await expect(page.getByTestId("buyer-card-name")).toHaveText(xss);
	await expect(page.getByTestId("buyer-status")).toHaveText("Link enviado, aguardando o comprador");
	await expect(page.getByTestId("buyer-card")).toContainText("ana@example.com");
	await expect(page.getByTestId("buyer-card")).toContainText("+5511987654321");
	expect(dialogs).toEqual([]);

	// The public page shows the number as reserved, never who reserved it.
	const href = (await page.getByTestId("public-link").getAttribute("href")) ?? "/";
	const visitor = await browser.newPage();
	await visitor.goto(href);
	await expect(visitor.locator('li[data-number="7"]')).toHaveAttribute("data-status", "reserved");
	const html = await visitor.content();
	expect(html).not.toContain("ana@example.com");
	expect(html).not.toContain("onerror");
	await expect(visitor.getByTestId("draw-pending")).toBeVisible();
	await visitor.close();

	page.once("dialog", (d) => void d.accept());
	await page.getByTestId("link-cancel").click();
	await expect(page.getByText("Link cancelado. O número voltou a ficar disponível.")).toBeVisible();
	await expect(page.locator('li[data-number="7"]')).toHaveAttribute("data-status", "available");
});

test("a reserved number cannot get a second link", async ({ page }) => {
	await registerVerifiedOwner(page, uniqueEmail("taken"));
	await createRaffle(page, "Rifa da cesta");

	await page.getByTestId("number-3").click();
	await page.getByTestId("buyer-name").fill("Bruno Souza");
	await page.getByTestId("buyer-email").fill("bruno@example.com");
	await page.getByTestId("buyer-phone").fill("(21) 3333-4444");
	await page.getByTestId("link-submit").click();
	await expect(page.getByTestId("link-result")).toBeVisible();

	// Same request again (e.g. a second tab): the server refuses with 409.
	const status = await page.evaluate(async () => {
		const res = await fetch(`${location.pathname}/numbers/3/link`, {
			method: "POST",
			headers: { "Content-Type": "application/json", Accept: "application/json" },
			body: JSON.stringify({ buyer_name: "Outro", buyer_email: "o@example.com", buyer_phone: "11987654321" }),
		});
		return res.status;
	});
	expect(status).toBe(409);
});
