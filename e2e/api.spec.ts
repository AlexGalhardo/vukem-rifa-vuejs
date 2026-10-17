import { expect, test } from "@playwright/test";

import { PaymentLinkSchema, RafflePageSchema, RaffleSchema } from "../resources/js/generated/api";
import { registerVerifiedOwner, uniqueEmail } from "./helpers/owner";

test("owner creates an API key, uses the REST API and revokes the key", async ({ page, playwright, baseURL }) => {
	await registerVerifiedOwner(page, uniqueEmail("api"));

	await page.getByTestId("nav-api-keys").click();
	await expect(page).toHaveURL(/\/settings\/api-keys$/);
	await expect(page.getByTestId("api-key-empty")).toBeVisible();

	await page.getByTestId("api-key-name").fill("Integração e2e");
	await page.getByTestId("api-key-ability-raffles-write").check();
	await page.getByTestId("api-key-submit").click();

	const token = (await page.getByTestId("api-key-token").textContent())?.trim() ?? "";
	expect(token).toMatch(/^[A-Z0-9]+\.[A-Z0-9]+$/);
	await expect(page.getByTestId("api-key-row-name")).toHaveText("Integração e2e");

	// The token is shown once: a reload lists the key without it.
	await page.reload();
	await expect(page.getByTestId("api-key-token")).toHaveCount(0);
	expect(await page.content()).not.toContain(token);

	// A server-to-server client: no cookies, only the bearer token.
	const api = await playwright.request.newContext({
		baseURL,
		extraHTTPHeaders: { Authorization: `Bearer ${token}` },
	});

	const created = await api.post("/api/v1/raffles", {
		data: { title: "Rifa via API", description: "", number_count: 20, price_reais: 5, duration_days: 7 },
	});
	expect(created.status()).toBe(201);
	const raffle = RaffleSchema.parse(await created.json());

	const list = await api.get("/api/v1/raffles");
	expect(list.status()).toBe(200);
	const listed = RafflePageSchema.parse(await list.json());
	expect(listed.data?.map((r) => r.id)).toEqual([raffle.id]);

	const linkRequest = {
		headers: { "Idempotency-Key": `e2e-${raffle.id}` },
		data: { buyer_name: "Maria Silva", buyer_email: "maria@example.com", buyer_phone: "(11) 98765-4321" },
	};
	const first = await api.post(`/api/v1/raffles/${raffle.id}/numbers/3/payment-links`, linkRequest);
	expect(first.status()).toBe(201);
	const link = PaymentLinkSchema.parse(await first.json());
	expect(link.number).toBe(3);
	expect(link.url).toMatch(/\/pay\/[\w-]{43}$/);

	const replay = await api.post(`/api/v1/raffles/${raffle.id}/numbers/3/payment-links`, linkRequest);
	expect(replay.status()).toBe(201);
	expect(replay.headers()["idempotent-replayed"]).toBe("true");
	expect(PaymentLinkSchema.parse(await replay.json()).url).toBe(link.url);

	// The owner's browser session is not an API credential.
	const withCookies = await page.request.get("/api/v1/raffles");
	expect(withCookies.status()).toBe(401);
	expect(withCookies.headers()["content-type"]).toContain("application/problem+json");

	// The raffle page offers the audit export.
	await page.goto(`/raffles/${raffle.id}`);
	const csv = await page.request.get(`/raffles/${raffle.id}/export.csv`);
	expect(csv.status()).toBe(200);
	expect(csv.headers()["content-disposition"]).toContain("attachment");
	await expect(page.getByTestId("export-csv")).toHaveAttribute("href", `/raffles/${raffle.id}/export.csv`);

	await page.goto("/settings/api-keys");
	page.once("dialog", (d) => void d.accept());
	await page.getByTestId("api-key-revoke").click();
	await expect(page.getByTestId("api-key-empty")).toBeVisible();

	const revoked = await api.get("/api/v1/raffles");
	expect(revoked.status()).toBe(401);
	await api.dispose();
});

test("API docs load under the CSP", async ({ page }) => {
	const violations: string[] = [];
	page.on("console", (msg) => {
		if (msg.type() === "error" && /Content Security Policy/i.test(msg.text())) violations.push(msg.text());
	});

	const res = await page.goto("/api/docs");
	expect(res?.status()).toBe(200);
	expect(res?.headers()["content-security-policy"]).toMatch(/script-src 'self' 'nonce-/);
	await expect(page.getByRole("link", { name: /Your balance/ }).first()).toBeVisible();
	await expect(page.getByRole("link", { name: "raffles", exact: true })).toBeVisible();
	// workaround(vukem v0.25.0): the apidocs CSP shim misses one inline style Scalar applies;
	// it is blocked (no 'unsafe-inline') and the docs still render. Scripts must never be blocked.
	expect(violations.filter((v) => !/^Applying inline style violates/.test(v))).toEqual([]);
});
