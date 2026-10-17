import AxeBuilder from "@axe-core/playwright";
import { expect, type Page } from "@playwright/test";

/** Fails on serious or critical axe violations of the page as it is now. */
export async function expectAccessible(page: Page, state: string): Promise<void> {
	// Wait for the theme class and Vue to settle so contrast is measured on the final colors.
	await page.waitForLoadState("networkidle");
	const results = await new AxeBuilder({ page }).analyze();
	const blocking = results.violations
		.filter((v) => v.impact === "serious" || v.impact === "critical")
		.map(
			(v) =>
				`${v.id} (${v.impact}): ${v.help} → ${v.nodes
					.slice(0, 5)
					.map((n) => `${n.target.join(" ")} ${n.html.slice(0, 120)}`)
					.join(" | ")}`,
		);
	// Soft: one run reports every failing state, not just the first.
	expect.soft(blocking, `axe violations on ${state}`).toEqual([]);
}
