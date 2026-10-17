import { createHash } from "node:crypto";

import { expect, test } from "@playwright/test";

// The draw itself is tested in Go (algorithm vectors, concurrency, triggers).
// Here the public result of the raffle created by the DrawDemo seeder
// (go run ./cmd/app db:seed, see docs/draw-algorithm.md) is rendered and
// verified in the browser. There is no test-only way to force a deadline.
const slug = "demodraw01";
const paid = [2, 5, 9];

test("public draw result is auditable and has no personal data", async ({ page }) => {
	const res = await page.goto(`/r/${slug}`);
	expect(res?.status(), "run `db:seed` against the e2e database first").toBe(200);

	const result = page.getByTestId("draw-result");
	await expect(result).toBeVisible();
	await expect(page.getByTestId("draw-count")).toHaveText("3");
	await expect(page.getByTestId("draw-list")).toHaveText(paid.join(","));
	await expect(page.getByTestId("draw-hash")).toHaveText(createHash("sha256").update(paid.join(",")).digest("hex"));
	await expect(page.getByTestId("draw-seed")).toHaveText(/^[0-9a-f]{64}$/);
	expect(paid).toContain(Number(await page.getByTestId("draw-winner").innerText()));
	await expect(result).toContainText("Fim do prazo");

	await page.getByTestId("draw-verify").locator("summary").click();
	await expect(page.getByText("Como conferir")).toBeVisible();
	await page.getByTestId("draw-verify-run").click();
	await expect(page.getByTestId("draw-verify-result")).toHaveText(
		"Conferido: o hash da lista e o número sorteado batem.",
	);

	const html = await page.content();
	expect(html).not.toContain("Pessoa Demo");
	expect(html).not.toContain("example.invalid");
	expect(html).not.toContain("5511987654321");
});
