import type { Plugin } from "vite";

export interface VukemPluginOptions {
	input: string[];
	publicDir?: string;
	buildDir?: string;
}

export default function vukem(options: VukemPluginOptions): Plugin;
