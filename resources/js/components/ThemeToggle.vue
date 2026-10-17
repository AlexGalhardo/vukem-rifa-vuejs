<script setup lang="ts">
import { useI18n } from "../composables/useI18n";
import { type ThemeMode, useTheme } from "../composables/useTheme";

// Same control as the vukem docs site (components/theme-toggle.tsx): lucide
// Sun / Moon / Monitor icons, inlined to avoid an icon dependency.
const { mode, setMode } = useTheme();
const { t } = useI18n();

const options: { value: ThemeMode; label: string }[] = [
	{ value: "light", label: "theme.light" },
	{ value: "dark", label: "theme.dark" },
	{ value: "system", label: "theme.system" },
];
</script>

<template>
	<div
		role="radiogroup"
		:aria-label="t('theme.toggle')"
		data-testid="theme-toggle"
		class="flex items-center gap-0.5 rounded-full border border-zinc-300 p-0.5 dark:border-zinc-700"
	>
		<button
			v-for="o in options"
			:key="o.value"
			type="button"
			role="radio"
			:aria-checked="mode === o.value"
			:aria-label="t(o.label)"
			:title="t(o.label)"
			:class="[
				'inline-flex size-6 items-center justify-center rounded-full transition-colors',
				mode === o.value
					? 'bg-zinc-200 text-zinc-900 dark:bg-zinc-800 dark:text-zinc-100'
					: 'text-zinc-600 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-zinc-100',
			]"
			@click="setMode(o.value)"
		>
			<svg
				aria-hidden="true"
				class="size-3.5"
				viewBox="0 0 24 24"
				fill="none"
				stroke="currentColor"
				stroke-width="2"
				stroke-linecap="round"
				stroke-linejoin="round"
			>
				<template v-if="o.value === 'light'">
					<circle cx="12" cy="12" r="4" />
					<path d="M12 2v2" />
					<path d="M12 20v2" />
					<path d="m4.93 4.93 1.41 1.41" />
					<path d="m17.66 17.66 1.41 1.41" />
					<path d="M2 12h2" />
					<path d="M20 12h2" />
					<path d="m6.34 17.66-1.41 1.41" />
					<path d="m19.07 4.93-1.41 1.41" />
				</template>
				<path
					v-else-if="o.value === 'dark'"
					d="M20.985 12.486a9 9 0 1 1-9.473-9.472c.405-.022.617.46.402.803a6 6 0 0 0 8.268 8.268c.344-.215.825-.004.803.401"
				/>
				<template v-else>
					<rect width="20" height="14" x="2" y="3" rx="2" />
					<line x1="8" x2="16" y1="21" y2="21" />
					<line x1="12" x2="12" y1="17" y2="21" />
				</template>
			</svg>
		</button>
	</div>
</template>
