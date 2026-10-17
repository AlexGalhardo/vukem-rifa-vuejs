import { devices, expect, type Page, test } from "@playwright/test";

import { expectAccessible } from "./helpers/a11y";
import { registerVerifiedOwner, uniqueEmail } from "./helpers/owner";

// Accessibility (axe) of every page and state, in light and dark, pt-BR and
// en, desktop and mobile. Needs the compose stack (ABACATEPAY_FAKE=true for
// the PIX modal) and `db:seed` for the drawn demo raffle.

type Variant = { theme: "light" | "dark"; locale: "pt-BR" | "en"; mobile: boolean };

const variants: Variant[] = [];
for (const theme of ["light", "dark"] as const) {
	for (const locale of ["pt-BR", "en"] as const) {
		for (const mobile of [false, true]) variants.push({ theme, locale, mobile });
	}
}

function lang(path: string, v: Variant): string {
	return `${path}${path.includes("?") ? "&" : "?"}lang=${v.locale}`;
}

async function createRaffle(page: Page): Promise<number> {
	await page.getByTestId("new-raffle").click();
	await page.getByTestId("raffle-title").fill("Rifa acessível");
	await page.getByTestId("raffle-description").fill("Prêmio: uma bicicleta.");
	await page.getByTestId("raffle-number-count").fill("12");
	await page.getByTestId("raffle-price").fill("5");
	await page.getByTestId("raffle-duration").fill("7");
	await page.getByTestId("raffle-submit").click();
	await expect(page).toHaveURL(/\/raffles\/\d+$/);
	return Number(new URL(page.url()).pathname.split("/").pop());
}

async function openLinkForm(page: Page, number: number): Promise<void> {
	await page.getByTestId(`number-${number}`).click();
	await expect(page.getByTestId("link-form")).toBeVisible();
}

async function generateLink(page: Page, number: number): Promise<string> {
	await openLinkForm(page, number);
	await page.getByTestId("buyer-name").fill("Maria Compradora");
	await page.getByTestId("buyer-email").fill(uniqueEmail("a11y-buyer"));
	await page.getByTestId("buyer-phone").fill("11987654321");
	await page.getByTestId("link-submit").click();
	const url = page.getByTestId("link-url");
	await expect(url).toHaveValue(/\/pay\//);
	return new URL(await url.inputValue()).pathname;
}

for (const v of variants) {
	const name = `${v.theme} · ${v.locale} · ${v.mobile ? "mobile" : "desktop"}`;

	test.describe(name, () => {
		test.use({
			locale: v.locale,
			colorScheme: v.theme,
			...(v.mobile ? { viewport: devices["Pixel 7"].viewport, isMobile: true, hasTouch: true } : {}),
		});

		test.beforeEach(async ({ page }) => {
			await page.addInitScript((theme) => {
				try {
					localStorage.setItem("theme", theme);
				} catch {
					// Storage blocked: the OS preference (colorScheme) still applies.
				}
			}, v.theme);
			page.on("dialog", (d) => void d.accept());
		});

		test("guest pages", async ({ page }) => {
			// /api/docs is the bundled third-party Scalar UI (vukem apidocs), not our markup: its axe
			// findings are reported upstream (framework-feedback) instead of failing this suite.
			for (const path of [
				"/",
				"/login",
				"/register",
				"/register/check-email",
				"/forgot-password",
				"/r/demodraw01",
			]) {
				await page.goto(lang(path, v));
				await expectAccessible(page, `${path} (${name})`);
			}

			const expired = await page.goto(lang("/pay/this-link-does-not-exist", v));
			expect(expired?.status()).toBe(404);
			await expectAccessible(page, `expired link (${name})`);
		});

		test("owner and buyer pages", async ({ page, browser }) => {
			test.setTimeout(180_000);

			// Registration labels are matched in pt-BR; switch to the variant locale after.
			await registerVerifiedOwner(page, uniqueEmail("a11y-owner"), "Dona da Rifa", "/register?lang=pt-BR");
			await page.goto(lang("/dashboard", v));
			await expectAccessible(page, `dashboard empty (${name})`);

			await page.goto("/raffles/create");
			await expectAccessible(page, `create raffle (${name})`);
			await page.goto("/dashboard");
			const id = await createRaffle(page);
			await expectAccessible(page, `raffle page (${name})`);

			await openLinkForm(page, 1);
			await expectAccessible(page, `link form (${name})`);
			await page.goto(`/raffles/${id}`);

			const payPath = await generateLink(page, 2);
			await expectAccessible(page, `link result (${name})`);

			const cancelPath = await generateLink(page, 3);
			await page.goto(`/raffles/${id}`);
			await page.getByTestId("number-3").click();
			await expectAccessible(page, `buyer card (${name})`);
			await page.getByTestId("link-cancel").click();
			await expect(page.getByTestId("number-grid").locator("li").nth(2)).toHaveAttribute(
				"data-status",
				"available",
			);

			await page.goto(`/raffles/${id}/edit`);
			await expectAccessible(page, `edit raffle (${name})`);
			await page.goto("/settings/api-keys");
			await expectAccessible(page, `api keys (${name})`);
			await page.goto("/dashboard");
			await expectAccessible(page, `dashboard with raffle (${name})`);

			// The buyer: no account, own context with the same theme/locale/viewport.
			const buyerCtx = await browser.newContext({
				locale: v.locale,
				colorScheme: v.theme,
				...(v.mobile ? { viewport: devices["Pixel 7"].viewport, isMobile: true, hasTouch: true } : {}),
			});
			const buyer = await buyerCtx.newPage();
			await buyer.goto(payPath);
			await expectAccessible(buyer, `pay page (${name})`);
			await buyer.getByTestId("pay-confirm").click();
			await expect(buyer.getByTestId("pix-inline")).toBeVisible();
			await expectAccessible(buyer, `PIX QR code (${name})`);
			await buyer.getByTestId("pix-copy").click();
			await expect(buyer.getByTestId("toast")).toBeVisible();
			await expectAccessible(buyer, `copy toast (${name})`);

			const paid = await buyer.request.post(`/dev${payPath}/simulate`);
			expect(paid.status()).toBe(200);
			await expect(buyer.getByTestId("pay-confirmed")).toBeVisible({ timeout: 15_000 });
			await expectAccessible(buyer, `payment confirmed (${name})`);

			const cancelled = await buyer.goto(cancelPath);
			expect(cancelled?.status()).toBe(410);
			await expectAccessible(buyer, `cancelled link (${name})`);
			await buyerCtx.close();

			await page.goto(`/raffles/${id}`);
			const publicHref = await page.getByTestId("public-link").getAttribute("href");
			expect(publicHref).toMatch(/^\/r\//);
			await page.goto(publicHref ?? "/");
			await expectAccessible(page, `public raffle (${name})`);
			await page.getByTestId("number-10").click();
			await expect(page.getByTestId("public-buy-form")).toBeVisible();
			await expectAccessible(page, `public buy form (${name})`);
			await page.goto(lang("/account/purchases", v));
			await expectAccessible(page, `my purchases (${name})`);
		});
	});
}

// One login for every variant: the login throttle (5/min per email and IP)
// would refuse eight parallel logins of the same admin.
test("admin payouts in every variant", async ({ browser }) => {
	const email = process.env.E2E_ADMIN_EMAIL;
	const password = process.env.E2E_ADMIN_PASSWORD;
	test.skip(!email || !password, "set E2E_ADMIN_EMAIL/E2E_ADMIN_PASSWORD and run db:seed (CI does)");
	test.setTimeout(180_000);

	const login = await browser.newContext({ locale: "pt-BR" });
	const page = await login.newPage();
	await page.goto("/login?lang=pt-BR");
	await page.getByLabel("E-mail").fill(email ?? "");
	await page.getByLabel("Senha", { exact: true }).fill(password ?? "");
	await page.getByRole("button", { name: "Entrar" }).click();
	await expect(page).toHaveURL(/\/dashboard$/);
	const storageState = await login.storageState();
	await login.close();

	for (const v of variants) {
		const name = `${v.theme} · ${v.locale} · ${v.mobile ? "mobile" : "desktop"}`;
		const ctx = await browser.newContext({
			storageState,
			locale: v.locale,
			colorScheme: v.theme,
			...(v.mobile ? { viewport: devices["Pixel 7"].viewport, isMobile: true, hasTouch: true } : {}),
		});
		await ctx.addInitScript((theme) => {
			try {
				localStorage.setItem("theme", theme);
			} catch {
				// Storage blocked: the OS preference (colorScheme) still applies.
			}
		}, v.theme);
		const admin = await ctx.newPage();
		await admin.goto(lang("/admin/payouts", v));
		await expect(admin.getByTestId("nav-payouts")).toBeAttached();
		await expectAccessible(admin, `admin payouts (${name})`);
		await ctx.close();
	}
});
