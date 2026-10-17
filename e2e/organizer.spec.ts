import { expect, test } from "@playwright/test";

import { registerVerifiedOwner, uniqueEmail } from "./helpers/owner";

// Owner request: the public page shows who organizes the raffle; contacts
// appear only when the organizer opts in.

test("organizer card with opt-in WhatsApp, Telegram and click-to-reveal email", async ({ page, browser }) => {
	const email = uniqueEmail("organizer");
	await registerVerifiedOwner(page, email, "Ana Organizadora");
	await page.getByTestId("new-raffle").click();
	await page.getByTestId("raffle-title").fill("Rifa da Ana");
	await page.getByTestId("raffle-number-count").fill("10");
	await page.getByTestId("raffle-price").fill("5");
	await page.getByTestId("raffle-duration").fill("7");
	await page.getByTestId("raffle-submit").click();
	await expect(page).toHaveURL(/\/raffles\/\d+$/);
	const publicPath = (await page.getByTestId("public-link").getAttribute("href")) ?? "/";

	const visitorCtx = await browser.newContext();
	const visitor = await visitorCtx.newPage();
	await visitor.goto(publicPath);
	await expect(visitor.getByTestId("organizer-name")).toHaveText("Ana Organizadora");
	await expect(visitor.getByTestId("organizer-raffles")).toContainText("1 rifa(s) criada(s)");
	await expect(visitor.getByTestId("organizer-contact")).toHaveCount(0);
	expect(await visitor.content()).not.toContain(email);

	// Invalid input is refused before saving.
	await page.goto("/settings/contact");
	await expect(page.getByTestId("contact-privacy")).toBeVisible();
	await page.getByTestId("contact-whatsapp").fill("123");
	await page.getByTestId("contact-show-whatsapp").check();
	await page.getByTestId("contact-submit").click();
	await expect(page.getByText("Informe um número brasileiro com DDD.")).toBeVisible();

	await page.getByTestId("contact-whatsapp").fill("11987654321");
	await page.getByTestId("contact-telegram").fill("@Ana_Rifas");
	await page.getByTestId("contact-show-telegram").check();
	await page.getByTestId("contact-show-email").check();
	await page.getByTestId("contact-submit").click();
	await expect(page.getByText("Contato público salvo.")).toBeVisible();

	await visitor.reload();
	const wa = visitor.getByTestId("organizer-whatsapp");
	await expect(wa).toHaveAttribute("href", "https://wa.me/5511987654321");
	await expect(wa).toHaveAttribute("rel", "noopener noreferrer nofollow");
	await expect(visitor.getByTestId("organizer-telegram")).toHaveAttribute("href", "https://t.me/ana_rifas");
	expect(await visitor.content()).not.toContain(email);
	await visitor.getByTestId("organizer-email-reveal").click();
	await expect(visitor.getByTestId("organizer-email")).toHaveAttribute("href", `mailto:${email}`);

	// Turning a contact off removes it right away.
	await page.getByTestId("contact-show-whatsapp").uncheck();
	await page.getByTestId("contact-submit").click();
	await expect(page.getByText("Contato público salvo.")).toBeVisible();
	await visitor.reload();
	await expect(visitor.getByTestId("organizer-whatsapp")).toHaveCount(0);
	await visitorCtx.close();
});
