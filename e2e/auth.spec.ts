import { expect, test } from "@playwright/test";

import { latestLinkTo } from "./helpers/mailpit";

test("register, verify email through Mailpit, reach the dashboard, log out and back in", async ({ page }) => {
	const email = `owner-${Date.now()}-${Math.floor(Math.random() * 1e6)}@example.com`;
	const password = "correct horse battery staple";

	await page.goto("/register");
	await page.getByLabel("Nome").fill("Dona da Rifa");
	await page.getByLabel("E-mail").fill(email);
	await page.getByLabel("Senha (mínimo de 12 caracteres)").fill(password);
	await page.getByLabel("Confirme a senha").fill(password);
	await page.getByRole("button", { name: "Criar conta" }).click();

	await expect(page).toHaveURL(/\/email\/verify$/);
	await expect(page.getByRole("heading", { name: "Confirme seu e-mail" })).toBeVisible();

	// The app builds links from APP_URL; keep only the path for the test origin.
	const link = new URL(await latestLinkTo(email));
	await page.goto(link.pathname + link.search);

	await expect(page).toHaveURL(/\/dashboard$/);
	await expect(page.getByTestId("dashboard-hello")).toHaveText("Olá, Dona da Rifa");

	await page.getByTestId("logout").click();
	await expect(page).toHaveURL(/\/$/);

	await page.goto("/login");
	await page.getByLabel("E-mail").fill(email);
	await page.getByLabel("Senha", { exact: true }).fill(password);
	await page.getByRole("button", { name: "Entrar", exact: true }).click();
	await expect(page).toHaveURL(/\/dashboard$/);
});
