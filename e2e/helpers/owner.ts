import { expect, type Page } from "@playwright/test";

import { latestLinkTo } from "./mailpit";

export const PASSWORD = "correct horse battery staple";

export function uniqueEmail(prefix: string): string {
	return `${prefix}-${Date.now()}-${Math.floor(Math.random() * 1e6)}@example.com`;
}

/** Registers an owner, verifies the email through Mailpit and lands on the dashboard. */
export async function registerVerifiedOwner(page: Page, email: string, name = "Dona da Rifa"): Promise<void> {
	await page.goto("/register");
	await page.getByLabel("Nome").fill(name);
	await page.getByLabel("E-mail").fill(email);
	await page.getByLabel("Senha (mínimo de 12 caracteres)").fill(PASSWORD);
	await page.getByLabel("Confirme a senha").fill(PASSWORD);
	await page.getByRole("button", { name: "Criar conta" }).click();
	await expect(page).toHaveURL(/\/email\/verify$/);

	// The app builds links from APP_URL; keep only the path for the test origin.
	const link = new URL(await latestLinkTo(email));
	await page.goto(link.pathname + link.search);
	await expect(page).toHaveURL(/\/dashboard$/);
}
