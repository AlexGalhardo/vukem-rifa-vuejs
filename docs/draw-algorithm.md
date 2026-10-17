# Draw algorithm (v1)

Every raffle result published on `/r/{slug}` can be recomputed by anyone. This page specifies algorithm `v1` byte by byte; the implementation is `app/draws/algorithm.go` and its test vectors are in `app/draws/algorithm_test.go`.

## Published data

| Field | Meaning |
| --- | --- |
| `seed` | 32 random bytes from `crypto/rand`, generated at draw time, lowercase hex (64 characters) |
| `participants_count` | how many numbers were paid (`n`) |
| `participants_sha256` | lowercase hex SHA-256 of the participants list (below) |
| `winning_number` | the winner, or empty when no number was paid |
| `algorithm` | `v1` |
| `trigger` | `deadline` (the end date passed) or `sold_out` (every number was paid) |
| `drawn_at` | when the draw ran |

The public number grid shows which numbers are paid; after the draw no number can change, so those are the participants.

## Participants list

The paid numbers in ascending order, written in decimal, joined by `,` with no spaces and no trailing separator. For numbers 3, 7 and 12 the list is the ASCII text `3,7,12`; its SHA-256 is `participants_sha256`. With no paid number the list is empty (SHA-256 `e3b0c442…b855`) and the raffle is closed without a winner.

## Picking the winner

Let `numbers` be the sorted participants and `n = len(numbers)`.

1. For `counter = 0, 1, 2, …`:
   1. `block = HMAC-SHA256(key = seed bytes, message = counter as 8 bytes, big-endian)`.
   2. `v` = the first 8 bytes of `block` read as a big-endian unsigned 64-bit integer.
   3. If `v < floor(2^64 / n) * n`, stop; otherwise try the next counter (rejection sampling).
2. The winner is `numbers[v mod n]` (0-based index).

Rejecting the top `2^64 mod n` values removes modulo bias: every participant has exactly the same chance. For realistic `n` (at most 1000) a rejection happens with probability below 2^-54, so the first counter almost always decides.

## Reference code

Python 3, standard library only:

```python
import hashlib, hmac, struct

def participants_hash(numbers):
    text = ",".join(str(n) for n in sorted(numbers))
    return hashlib.sha256(text.encode()).hexdigest()

def pick(seed_hex, numbers):
    seed, nums = bytes.fromhex(seed_hex), sorted(numbers)
    n = len(nums)
    limit = (2**64 // n) * n
    counter = 0
    while True:
        block = hmac.new(seed, struct.pack(">Q", counter), hashlib.sha256).digest()
        v = int.from_bytes(block[:8], "big")
        if v < limit:
            return nums[v % n]
        counter += 1
```

Go (same as `draws.Pick`):

```go
func pick(seed []byte, numbers []int) int {
	sorted := slices.Sorted(slices.Values(numbers))
	n := uint64(len(sorted))
	rem := (math.MaxUint64%n + 1) % n // 2^64 mod n
	var msg [8]byte
	for counter := uint64(0); ; counter++ {
		binary.BigEndian.PutUint64(msg[:], counter)
		mac := hmac.New(sha256.New, seed)
		mac.Write(msg[:])
		v := binary.BigEndian.Uint64(mac.Sum(nil)[:8])
		if rem == 0 || v <= math.MaxUint64-rem {
			return sorted[v%n]
		}
	}
}
```

The page also has a "verify in the browser" button that runs the same steps with Web Crypto (`resources/js/lib/draw.ts`).

## Test vectors

| Seed (hex) | Participants | Winner | Participants SHA-256 |
| --- | --- | --- | --- |
| `00` × 32 | `1` | 1 | `6b86b273…875b4b` |
| `00` × 32 | `1,2,…,10` | 5 | `d10434d8…79c425` |
| `ff` × 32 | `3,7,12` | 7 | `6e1edfa5…e656df` |
| `000102…1e1f` | `1,2,…,1000` | 706 | `72827ef0…576db6` |

## When the draw runs

- **Deadline:** creating a raffle queues a `RunDraw` job delayed until `ends_at` (re-queued if the duration changes; an early job is a no-op), and a scheduler task (`draws:due`, every minute) draws any active raffle past its deadline whose job was lost.
- **Sold out:** after the last number is paid the payment flow queues `RunDraw` with trigger `sold_out`; the draw re-checks that every number is paid and otherwise does nothing. The every-minute task also draws sold-out raffles.
- **Exactly once:** the draw locks the raffle row (`SELECT … FOR UPDATE` on Postgres) and a guarded `UPDATE raffles SET status = … WHERE id = ? AND status = 'active'` decides concurrent runs; `draws.raffle_id` is unique as a backstop. Retries and duplicate triggers are no-ops.
- **End of sales:** in the same transaction, live payment links are expired and their numbers released, so nothing can be paid after the draw; a PIX paid later is recorded as a refund.
- **Zero paid numbers:** the raffle becomes `closed` and the draw row has no winner.
- **Archived raffles** are never drawn, even past their deadline: archiving is the owner cancelling the raffle. The public page says so; refunds of numbers already paid are handled by an admin outside the app.
- The seed is generated when the draw runs, not published beforehand. Publishing `sha256(seed)` at raffle creation (commit-reveal) is a possible later enhancement (PLAN.md D12).

## Testing

The algorithm, the exactly-once guarantee (20 concurrent runs on Postgres), triggers and pages are tested in Go (`app/draws`, `tests/draws_test.go`, `tests/draws_integration_test.go`). End-to-end tests cannot wait for a real deadline and the app has no test-only shortcut to force one: `e2e/draw.spec.ts` reads a raffle created by the `DrawDemo` seeder (`go run ./cmd/app db:seed`, refused in production without `--force`), drawn by the real `draws.Service`.
