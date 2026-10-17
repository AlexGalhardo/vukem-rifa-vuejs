import { expect, type Page } from "@playwright/test";

import { latestLinkTo } from "./mailpit";

export const PASSWORD = "correct horse battery staple";

export function uniqueEmail(prefix: string): string {
	return `${prefix}-${Date.now()}-${Math.floor(Math.random() * 1e6)}@example.com`;
}

/** Registers an owner, verifies the email through Mailpit, logs in and lands on the dashboard. */
/** The labels are matched in pt-BR: pass "/register?lang=pt-BR" from an English context. */
export async function registerVerifiedOwner(
	page: Page,
	email: string,
	name = "Dona da Rifa",
	registerPath = "/register",
): Promise<void> {
	await page.goto(registerPath);
	await page.getByLabel("Nome").fill(name);
	await page.getByLabel("E-mail").fill(email);
	await page.getByLabel("Senha (mínimo de 12 caracteres)").fill(PASSWORD);
	await page.getByLabel("Confirme a senha").fill(PASSWORD);
	await page.getByRole("button", { name: "Criar conta" }).click();
	// Sign-up never logs in (same answer for new and existing emails).
	await expect(page).toHaveURL(/\/register\/check-email$/);

	// The app builds links from APP_URL; keep only the path for the test origin.
	const link = new URL(await latestLinkTo(email));
	await page.goto(link.pathname + link.search);
	await expect(page).toHaveURL(/\/login$/);
	// Keep the labels in pt-BR when the caller forced it (?lang is per request).
	const lang = new URL(registerPath, "http://localhost").searchParams.get("lang");
	if (lang) await page.goto(`/login?lang=${lang}`);

	await page.getByLabel("E-mail").fill(email);
	await page.getByLabel("Senha", { exact: true }).fill(PASSWORD);
	await page.getByRole("button", { name: "Entrar", exact: true }).click();
	await expect(page).toHaveURL(/\/dashboard$/);
}
