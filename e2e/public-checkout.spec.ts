import { expect, type Page, test } from "@playwright/test";

import { registerVerifiedOwner, uniqueEmail } from "./helpers/owner";

// Owner request: anyone with the public raffle link buys a free number
// without an account; the purchase shows up in the buyer's history when
// they have one.

async function createRaffle(page: Page): Promise<string> {
	await page.getByTestId("new-raffle").click();
	await page.getByTestId("raffle-title").fill("Rifa pública");
	await page.getByTestId("raffle-number-count").fill("12");
	await page.getByTestId("raffle-price").fill("7");
	await page.getByTestId("raffle-duration").fill("7");
	await page.getByTestId("raffle-submit").click();
	await expect(page).toHaveURL(/\/raffles\/\d+$/);
	return (await page.getByTestId("public-link").getAttribute("href")) ?? "/";
}

test("a visitor buys a free number from the public page and pays by PIX", async ({ page, browser }) => {
	await registerVerifiedOwner(page, uniqueEmail("pub-owner"));
	const publicPath = await createRaffle(page);

	// The owner reserves number 2: visitors cannot pick it.
	await page.getByTestId("number-2").click();
	await page.getByTestId("buyer-name").fill("Reservado");
	await page.getByTestId("buyer-email").fill(uniqueEmail("res"));
	await page.getByTestId("buyer-phone").fill("11987654321");
	await page.getByTestId("link-submit").click();
	await expect(page.getByTestId("link-url")).toBeVisible();

	const ctx = await browser.newContext();
	const visitor = await ctx.newPage();
	await visitor.goto(publicPath);
	await expect(visitor.getByTestId("number-2")).toHaveCount(0);
	await expect(visitor.locator('li[data-number="2"]')).toHaveAttribute("data-status", "reserved");

	// Consent is required before anything is reserved.
	await visitor.getByTestId("number-5").click();
	await visitor.getByTestId("public-buyer-name").fill("Paula Pública");
	await visitor.getByTestId("public-buyer-email").fill(uniqueEmail("visitor"));
	await visitor.getByTestId("public-buyer-phone").fill("31998765432");
	await visitor.getByTestId("public-buy-submit").click();
	await expect(visitor.getByText("Aceite o aviso de privacidade para continuar.")).toBeVisible();

	await visitor.getByTestId("public-buyer-consent").check();
	await visitor.getByTestId("public-buy-submit").click();
	await expect(visitor).toHaveURL(/\/pay\/[A-Za-z0-9_-]{43}$/);
	const payPath = new URL(visitor.url()).pathname;
	await expect(visitor.getByTestId("pay-number")).toHaveText("Número 5");

	await visitor.getByTestId("pay-confirm").click();
	await expect(visitor.getByTestId("pix-inline")).toBeVisible();
	expect((await visitor.request.post(`/dev${payPath}/simulate`)).status()).toBe(200);
	await expect(visitor.getByTestId("pay-confirmed")).toBeVisible({ timeout: 15_000 });

	// The public page shows the number as paid and no buyer data.
	await visitor.goto(publicPath);
	await expect(visitor.locator('li[data-number="5"]')).toHaveAttribute("data-status", "paid");
	expect(await visitor.content()).not.toContain("Paula Pública");

	// The owner sees where the purchase came from.
	await page.reload();
	await page.getByTestId("number-5").click();
	await expect(page.getByTestId("buyer-origin")).toHaveText("Comprado pela página pública");
	await ctx.close();
});

test("a signed-in buyer finds the purchase in Minhas compras", async ({ page, browser }) => {
	const ownerCtx = await browser.newContext();
	const owner = await ownerCtx.newPage();
	await registerVerifiedOwner(owner, uniqueEmail("pub-owner2"));
	const publicPath = await createRaffle(owner);
	await ownerCtx.close();

	const email = uniqueEmail("buyer-acct");
	await registerVerifiedOwner(page, email, "Bia Compradora");
	await page.goto(publicPath);
	await page.getByTestId("number-7").click();
	await expect(page.getByTestId("public-buyer-email")).toHaveValue(email);
	await page.getByTestId("public-buyer-phone").fill("31998765432");
	await page.getByTestId("public-buyer-consent").check();
	await page.getByTestId("public-buy-submit").click();
	await expect(page).toHaveURL(/\/pay\//);

	await page.getByTestId("nav-purchases").click();
	await expect(page).toHaveURL(/\/account\/purchases$/);
	await expect(page.getByTestId("purchases-unpaid")).toContainText("Rifa pública");
	await expect(page.getByTestId("purchases-unpaid")).toContainText("Aguardando pagamento");
});
