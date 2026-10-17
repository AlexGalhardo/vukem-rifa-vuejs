import { ref } from "vue";

/** The preference the user picks; "system" follows the OS (prefers-color-scheme). */
export type ThemeMode = "light" | "dark" | "system";

function stored(): ThemeMode {
	try {
		const v = localStorage.getItem("theme");
		return v === "light" || v === "dark" ? v : "system";
	} catch {
		return "system";
	}
}

const isBrowser = typeof document !== "undefined";
const mode = ref<ThemeMode>(isBrowser ? stored() : "system");
const media = isBrowser ? matchMedia("(prefers-color-scheme: dark)") : null;

function paint(): void {
	const dark = mode.value === "dark" || (mode.value === "system" && (media?.matches ?? false));
	document.documentElement.classList.toggle("dark", dark);
}

// The pre-paint script in app.html applies the first theme; this keeps
// "system" in sync when the OS switches while the page is open.
media?.addEventListener("change", () => {
	if (mode.value === "system") paint();
});

export function useTheme(): { mode: typeof mode; setMode: (next: ThemeMode) => void } {
	function setMode(next: ThemeMode): void {
		mode.value = next;
		try {
			if (next === "system") localStorage.removeItem("theme");
			else localStorage.setItem("theme", next);
		} catch {
			// Storage can be blocked (private mode); the choice still applies to this page view.
		}
		paint();
	}

	return { mode, setMode };
}
