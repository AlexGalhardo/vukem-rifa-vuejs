import { ref } from "vue";

export type Theme = "light" | "dark";

const theme = ref<Theme>(
	typeof document !== "undefined" && document.documentElement.classList.contains("dark") ? "dark" : "light",
);

export function useTheme() {
	function apply(next: Theme): void {
		theme.value = next;
		document.documentElement.classList.toggle("dark", next === "dark");
		try {
			localStorage.setItem("theme", next);
		} catch {
			// Storage can be blocked (private mode); the toggle still works for this page view.
		}
	}

	function toggle(): void {
		apply(theme.value === "dark" ? "light" : "dark");
	}

	return { theme, toggle };
}
