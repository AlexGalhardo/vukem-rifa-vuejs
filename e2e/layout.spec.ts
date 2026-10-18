import { expect, test } from "@playwright/test";

import { expectAccessible } from "./helpers/a11y";
import { registerVerifiedOwner, uniqueEmail } from "./helpers/owner";

test("navbar uses the docs site container (90rem, 1rem/1.5rem gutters, sticky)", async ({ page }) => {
	await page.setViewportSize({ width: 1600, height: 900 });
	await page.goto("/");
	const box = page.getByTestId("navbar-container");
	const style = await box.evaluate((el) => {
		const s = getComputedStyle(el);
		return { maxWidth: s.maxWidth, paddingLeft: s.paddingLeft };
	});
	expect(style).toEqual({ maxWidth: "1440px", paddingLeft: "24px" });
	expect((await box.boundingBox())?.width).toBe(1440);
	await expect(page.getByTestId("navbar")).toHaveCSS("position", "sticky");

	await page.setViewportSize({ width: 390, height: 800 });
	await expect(box).toHaveCSS("padding-left", "16px");
});

test("on a phone the navbar never overflows and the links live in a menu", async ({ page }) => {
	await page.setViewportSize({ width: 360, height: 780 });
	await page.goto("/?lang=en");
	expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
	await expect(page.getByTestId("mobile-menu")).toHaveCount(0);
	await page.getByTestId("nav-menu").click();
	await expect(page.getByTestId("nav-menu")).toHaveAttribute("aria-expanded", "true");
	await expect(page.getByTestId("mobile-menu").getByRole("link", { name: "Log in" })).toBeVisible();
});

test("an Inertia visit that hits a 404 shows the Error page inside the app layout", async ({ page }) => {
	await registerVerifiedOwner(page, uniqueEmail("error-page"));
	await page.getByTestId("new-raffle").click();
	await page.getByTestId("raffle-title").fill("Rifa que some");
	await page.getByTestId("raffle-description").fill("Prêmio: nada.");
	await page.getByTestId("raffle-number-count").fill("10");
	await page.getByTestId("raffle-price").fill("5");
	await page.getByTestId("raffle-duration").fill("7");
	await page.getByTestId("raffle-submit").click();
	await expect(page).toHaveURL(/\/raffles\/\d+$/);
	const path = new URL(page.url()).pathname;

	await page.goto("/dashboard");
	// Deleted behind the dashboard's back: its link now points at a 404.
	const status = await page.evaluate(
		(p) =>
			fetch(p, { method: "DELETE", headers: { Accept: "application/json" }, redirect: "manual" }).then(
				(r) => r.status,
			),
		path,
	);
	expect(status).toBeLessThan(400);

	await page.getByRole("link", { name: "Rifa que some" }).click();
	await expect(page.getByTestId("error-page")).toBeVisible();
	await expect(page.getByTestId("error-status")).toHaveText("404");
	await expect(page.getByRole("heading", { level: 1 })).toHaveText("Rifa não encontrada.");
	// Inside the layout, not in Inertia's raw-HTML modal.
	await expect(page.getByTestId("navbar")).toBeVisible();
	await expect(page.locator("dialog, #inertia-error-modal, iframe")).toHaveCount(0);
	await expectAccessible(page, "error page 404");
});
