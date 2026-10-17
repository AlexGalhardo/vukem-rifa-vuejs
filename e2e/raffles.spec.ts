import { expect, test } from "@playwright/test";

import { registerVerifiedOwner, uniqueEmail } from "./helpers/owner";

test("owner creates a raffle and shares its public page", async ({ page, browser }) => {
	const email = uniqueEmail("raffle");
	await registerVerifiedOwner(page, email);

	await page.getByTestId("new-raffle").click();
	await page.getByTestId("raffle-title").fill("Rifa da bicicleta");
	await page.getByTestId("raffle-description").fill("Bicicleta aro 29 <script>alert(1)</script>");
	await page.getByTestId("raffle-number-count").fill("25");
	await page.getByTestId("raffle-price").fill("5");
	await page.getByTestId("raffle-duration").fill("7");
	await page.getByTestId("raffle-submit").click();

	await expect(page).toHaveURL(/\/raffles\/\d+$/);
	await expect(page.getByTestId("raffle-heading")).toHaveText("Rifa da bicicleta");
	await expect(page.getByTestId("number-grid").locator("li")).toHaveCount(25);

	const publicHref = await page.getByTestId("public-link").getAttribute("href");
	expect(publicHref).toMatch(/^\/r\/[a-z0-9]{10}$/);

	// A visitor without a session sees the raffle, never the owner's data.
	const visitor = await browser.newPage();
	const dialogs: string[] = [];
	visitor.on("dialog", (d) => {
		dialogs.push(d.message());
		void d.dismiss();
	});
	await visitor.goto(publicHref ?? "/");
	await expect(visitor.getByTestId("public-title")).toHaveText("Rifa da bicicleta");
	await expect(visitor.getByTestId("number-grid").locator('li[data-status="available"]')).toHaveCount(25);
	await expect(visitor.getByText("<script>alert(1)</script>", { exact: false })).toBeVisible();
	expect(await visitor.content()).not.toContain(email);
	expect(dialogs).toEqual([]);
	await visitor.close();
});

test("client validation blocks out-of-range values", async ({ page }) => {
	await registerVerifiedOwner(page, uniqueEmail("limits"));
	await page.goto("/raffles/create");
	await page.getByTestId("raffle-title").fill("Rifa inválida");
	await page.getByTestId("raffle-number-count").fill("5");
	await page.getByTestId("raffle-submit").click();

	await expect(page).toHaveURL(/\/raffles\/create$/);
	await expect(page.getByText("Valor inválido para este campo.")).toBeVisible();
});
