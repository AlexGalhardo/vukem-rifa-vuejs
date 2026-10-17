import { defineConfig, devices } from "@playwright/test";

// The app must already be running (docker compose up, or the CI job) at BASE_URL.
export default defineConfig({
	testDir: "./e2e",
	fullyParallel: true,
	forbidOnly: !!process.env.CI,
	retries: process.env.CI ? 1 : 0,
	reporter: process.env.CI ? [["github"], ["html", { open: "never" }]] : "list",
	use: {
		baseURL: process.env.BASE_URL ?? "http://localhost:3000",
		trace: "retain-on-failure",
		// Brazilian audience: browsers send pt-BR; the en catalog is covered by its own test.
		locale: "pt-BR",
	},
	projects: [{ name: "chromium", use: { ...devices["Desktop Chrome"] } }],
});
