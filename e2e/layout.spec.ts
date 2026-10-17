import { expect, test } from "@playwright/test";

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
