# validation.md — server_auth

## How to Run Tests

```bash
# Unit tests with race detector and coverage:
go test -race -count=1 -cover ./internal/auth/...

# Detailed per-function coverage:
go test -count=1 -coverprofile=coverage.out ./internal/auth/...
go tool cover -func=coverage.out

# Full suite (must not regress):
make test
```

**"Passing" definition:** all tests exit `PASS`, no data races, statement coverage
≥ 90% for `ekvs/internal/auth`, and `make test` remains fully green.

---

## Manual Checklist

- [ ] `internal/auth/errors.go` exports exactly `ErrKeyNotFound`, `ErrInvalidSignature`, `ErrReplayDetected`.
- [ ] `go build ./internal/auth/...` produces no errors.
- [ ] `go vet ./internal/auth/...` produces no diagnostics.
- [ ] `go mod tidy` leaves `go.mod` / `go.sum` unchanged.
- [ ] The server process does **not** write to the `.keys/` directory.
- [ ] A `.pub` file with a free-form name (e.g. `alice_laptop.pub`) placed in `.keys/` is picked up by `Lookup` and `List` without server restart (reads happen on each call).
- [ ] A non-`.pub` file in `.keys/` is silently ignored by both `Lookup` and `List`.
- [ ] An unparseable `.pub` file is silently skipped; other valid keys are still found.
- [ ] `List()` returns fingerprints in alphabetical order.
- [ ] `AuthMiddleware`: a request signed with a registered key reaches the next handler.
- [ ] `AuthMiddleware`: a request signed with an unregistered key is rejected with 401.
- [ ] `AuthMiddleware`: a request with a timestamp older than **30 seconds** is rejected with 401 (consistent with `ssh_auth_primitives.CheckTimestamp`).
- [ ] `errors.Is(err, auth.ErrInvalidSignature)` returns true for signature errors propagated from middleware internals.
- [ ] `errors.Is(err, auth.ErrReplayDetected)` returns true for replay errors propagated from middleware internals.
- [ ] Running the concurrency test with `-race` reports **no data races**.

---

## Test-Case Matrix

### `NewKeyStore`

| # | Input | Expected |
|---|-------|----------|
| NK-1 | Accessible existing directory | No error |
| NK-2 | Non-existent or inaccessible path | Error returned |

### `Lookup`

| # | Input | Expected |
|---|-------|----------|
| L-1 | Fingerprint of a key present in `.keys/` | Returns matching `gossh.PublicKey`, no error |
| L-2 | Unknown fingerprint | `ErrKeyNotFound` |
| L-3 | `.keys/` contains an unparseable file alongside a valid one | Valid key returned; bad file silently skipped |
| L-4 | Non-`.pub` files present in `.keys/` | Ignored; `ErrKeyNotFound` if no valid `.pub` matches |

### `List`

| # | Input | Expected |
|---|-------|----------|
| LS-1 | Empty `.keys/` directory | Empty slice (non-nil), no error |
| LS-2 | One valid `.pub` file | Slice with one fingerprint |
| LS-3 | Multiple valid `.pub` files | Fingerprints sorted alphabetically |
| LS-4 | Mix of valid, unparseable, and non-`.pub` files | Only valid fingerprints returned, sorted |

### `AuthMiddleware`

| # | Scenario | Expected |
|---|----------|----------|
| AM-1 | Valid headers + key present in `.keys/` + fresh timestamp | Next handler called; `UserIDFromContext` returns fingerprint |
| AM-2 | Missing `X-Timestamp` header | 401, JSON error body |
| AM-3 | Missing `X-Fingerprint` header | 401, JSON error body |
| AM-4 | Missing `X-Signature` header | 401, JSON error body |
| AM-5 | `X-Timestamp` is not a valid integer | 401, JSON error body |
| AM-6 | `X-Fingerprint` not present in `.keys/` | 401, JSON error body |
| AM-7 | Signature does not match message | 401, JSON error body |
| AM-8 | Timestamp older than **30 seconds** | 401, JSON error body |

### `UserIDFromContext`

| # | Scenario | Expected |
|---|----------|----------|
| UC-1 | Context populated by `AuthMiddleware` | Returns fingerprint string and `true` |
| UC-2 | Plain context without middleware | Returns `""` and `false` |

### Concurrency

| # | Scenario | Expected |
|---|----------|----------|
| C-1 | N goroutines calling `Lookup` concurrently on same key | All return same result; `-race` reports no races |
| C-2 | N goroutines calling `List` concurrently | All return consistent results; no races |
| C-3 | Mix of `Lookup` and `List` calls concurrently | No races |


