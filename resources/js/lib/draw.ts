// Browser re-implementation of the v1 draw (app/draws/algorithm.go,
// docs/draw-algorithm.md) so visitors can check a published result.

function hexToBytes(hex: string): Uint8Array<ArrayBuffer> {
	const out = new Uint8Array(hex.length / 2);
	for (let i = 0; i < out.length; i++) out[i] = Number.parseInt(hex.slice(i * 2, i * 2 + 2), 16);
	return out;
}

function bytesToHex(bytes: ArrayBuffer): string {
	return Array.from(new Uint8Array(bytes), (b) => b.toString(16).padStart(2, "0")).join("");
}

/** The canonical participants text: ascending numbers joined by "," ("3,7,12"). */
export function participantsList(numbers: number[]): string {
	return [...numbers].sort((a, b) => a - b).join(",");
}

export async function participantsHash(numbers: number[]): Promise<string> {
	const data = new TextEncoder().encode(participantsList(numbers));
	return bytesToHex(await crypto.subtle.digest("SHA-256", data));
}

/** Recomputes the winner from the seed (hex) and the paid numbers. */
export async function pickWinner(seedHex: string, numbers: number[]): Promise<number | null> {
	const sorted = [...numbers].sort((a, b) => a - b);
	if (sorted.length === 0 || seedHex.length !== 64) return null;

	const key = await crypto.subtle.importKey("raw", hexToBytes(seedHex), { name: "HMAC", hash: "SHA-256" }, false, [
		"sign",
	]);
	const n = BigInt(sorted.length);
	const limit = ((1n << 64n) / n) * n;

	for (let counter = 0n; ; counter++) {
		const msg = new DataView(new ArrayBuffer(8));
		msg.setBigUint64(0, counter);
		const mac = await crypto.subtle.sign("HMAC", key, msg.buffer);
		const v = new DataView(mac).getBigUint64(0);
		if (v < limit) return sorted[Number(v % n)] ?? null;
	}
}
