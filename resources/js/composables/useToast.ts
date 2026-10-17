import { ref } from "vue";

export type ToastKind = "success" | "error";

export interface Toast {
	id: number;
	kind: ToastKind;
	message: string;
}

const toasts = ref<Toast[]>([]);
let next = 1;

/** App-wide toasts (rendered once by AppLayout), dismissed after a few seconds. */
export function useToast(): {
	toasts: typeof toasts;
	show: (message: string, kind?: ToastKind, ms?: number) => void;
	dismiss: (id: number) => void;
} {
	function dismiss(id: number): void {
		toasts.value = toasts.value.filter((t) => t.id !== id);
	}

	function show(message: string, kind: ToastKind = "success", ms = 3000): void {
		const id = next++;
		toasts.value = [...toasts.value, { id, kind, message }];
		window.setTimeout(() => dismiss(id), ms);
	}

	return { toasts, show, dismiss };
}
