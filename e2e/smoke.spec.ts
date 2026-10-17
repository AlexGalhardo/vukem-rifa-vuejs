import { expect, test } from "@playwright/test";

test("health endpoint answers", async ({ request }) => {
	const res = await request.get("/up");
	expect(res.ok()).toBeTruthy();
});

test("home renders in pt-BR by default", async ({ page }) => {
	await page.goto("/");
	await expect(page.getByTestId("home-title")).toHaveText("Rifas online com PIX");
	await expect(page.locator("html")).toHaveAttribute("lang", "pt-BR");
});

test("theme toggle switches and persists", async ({ page }) => {
	await page.emulateMedia({ colorScheme: "light" });
	await page.goto("/");
	const html = page.locator("html");
	await expect(html).not.toHaveClass(/dark/);

	await page.getByTestId("theme-toggle").click();
	await expect(html).toHaveClass(/dark/);

	await page.reload();
	await expect(html).toHaveClass(/dark/);
});

test("locale switcher changes language", async ({ page }) => {
	await page.goto("/");
	await page.getByTestId("locale-switcher").selectOption("en");
	await expect(page.getByTestId("home-title")).toHaveText("Online raffles with PIX");

	await page.reload();
	await expect(page.getByTestId("home-title")).toHaveText("Online raffles with PIX");
});

test("no console errors on home (CSP included)", async ({ page }) => {
	const errors: string[] = [];
	page.on("console", (msg) => {
		if (msg.type() === "error") errors.push(msg.text());
	});
	await page.goto("/");
	await page.getByTestId("theme-toggle").click();
	expect(errors).toEqual([]);
});

test.describe("english browser", () => {
	test.use({ locale: "en-US" });

	test("Accept-Language picks the en catalog", async ({ page }) => {
		await page.goto("/");
		await expect(page.getByTestId("home-title")).toHaveText("Online raffles with PIX");
	});
});
