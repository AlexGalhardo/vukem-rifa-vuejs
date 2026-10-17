import { expect, test } from "@playwright/test";

import { PASSWORD, registerVerifiedOwner, uniqueEmail } from "./helpers/owner";

test("register, verify email through Mailpit, reach the dashboard, log out and back in", async ({ page }) => {
	const email = uniqueEmail("owner");

	await registerVerifiedOwner(page, email);
	await expect(page.getByTestId("dashboard-hello")).toHaveText("Olá, Dona da Rifa");

	await page.getByTestId("logout").click();
	await expect(page).toHaveURL(/\/$/);

	await page.goto("/login");
	await page.getByLabel("E-mail").fill(email);
	await page.getByLabel("Senha", { exact: true }).fill(PASSWORD);
	await page.getByRole("button", { name: "Entrar", exact: true }).click();
	await expect(page).toHaveURL(/\/dashboard$/);
});
