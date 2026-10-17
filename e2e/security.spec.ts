import { expect, test } from "@playwright/test";

import { registerVerifiedOwner, uniqueEmail } from "./helpers/owner";

// Browser-side OWASP checks (docs/owasp-matrix.md); the Go suite covers the rest.

test("pages send the security headers and raise no CSP violations", async ({ page }) => {
	const violations: string[] = [];
	page.on("console", (m) => {
		if (m.type() === "error" && /Content Security Policy/i.test(m.text())) violations.push(m.text());
	});

	for (const path of ["/", "/login", "/register", "/r/demodraw01", "/pay/unknown-token"]) {
		const res = await page.goto(path);
		const h = res?.headers() ?? {};
		expect(h["x-frame-options"], path).toBe("DENY");
		expect(h["x-content-type-options"], path).toBe("nosniff");
		expect(h["content-security-policy"], path).toContain("frame-ancestors 'none'");
		expect(h["content-security-policy"], path).not.toContain("unsafe-inline");
	}

	expect(violations).toEqual([]);
});

test("a cancelled payment link cannot be reused", async ({ page, browser }) => {
	await registerVerifiedOwner(page, uniqueEmail("sec-owner"));
	await page.getByTestId("new-raffle").click();
	await page.getByTestId("raffle-title").fill("Rifa segura");
	await page.getByTestId("raffle-number-count").fill("10");
	await page.getByTestId("raffle-price").fill("5");
	await page.getByTestId("raffle-duration").fill("7");
	await page.getByTestId("raffle-submit").click();
	await expect(page).toHaveURL(/\/raffles\/\d+$/);

	await page.getByTestId("number-4").click();
	await page.getByTestId("buyer-name").fill("Comprador");
	await page.getByTestId("buyer-email").fill(uniqueEmail("sec-buyer"));
	await page.getByTestId("buyer-phone").fill("11987654321");
	await page.getByTestId("link-submit").click();
	const linkPath = new URL(await page.getByTestId("link-url").inputValue()).pathname;

	page.on("dialog", (d) => void d.accept());
	await page.reload();
	await page.getByTestId("number-4").click();
	await page.getByTestId("link-cancel").click();
	await expect(page.getByTestId("number-grid").locator("li").nth(3)).toHaveAttribute("data-status", "available");

	const buyerCtx = await browser.newContext();
	const buyer = await buyerCtx.newPage();
	const res = await buyer.goto(linkPath);
	expect(res?.status()).toBe(410);
	await expect(buyer.getByTestId("pay-expired")).toBeVisible();
	await expect(buyer.getByTestId("pay-confirm")).toHaveCount(0);

	// Confirming the dead link directly is refused too.
	const confirm = await buyer.request.post(`${linkPath}/confirm`, {
		headers: { "X-Requested-With": "XMLHttpRequest" },
	});
	expect(confirm.status()).not.toBe(200);
	await buyerCtx.close();
});
