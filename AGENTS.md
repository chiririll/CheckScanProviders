# CheckScanProviders

Go library: match/resolve a receipt QR into eq receipt JSON. Embedded in CheckScan as a C-shared library.

Overview and `go test ./...`: [README.md](README.md).

## Map

| Path | What |
| --- | --- |
| [pkg/resolve](pkg/resolve/resolve.go) | Public API: `Match` / `Resolve`, default registry |
| [pkg/provider](pkg/provider/provider.go) | `Provider` interface |
| [pkg/eq](pkg/eq/receipt.go) | Receipt JSON model |
| [include/checkscan.h](include/checkscan.h) | C ABI (`checkscan_match` / `resolve` / `providers` / `set_log`) |
| [cmd/lib](cmd/lib/main.go) | c-shared exports |
| [internal/nativelog](internal/nativelog/nativelog.go) | Library logs: stderr, or host `checkscan_set_log` |
| [internal/](internal/) | One package per provider |
| [testdata/](testdata/) | QR fixtures |

`Parse` is local unless `provider.Remote(ctx)` is true (scan and refresh pass `remote`). `internal/httplimit` is a per-host request budget (sliding windows + 403/429/503 cooldown). It is not receipt storage: CheckScan already dedupes by QR hash in SQLite before `resolve`. Gate timestamps may be written under `TMPDIR` so a new isolate still sees the budget. A successful receipt with no line items sets `checkscan.items_unavailable`; an empty list without that flag is still incomplete.

New formats live here, not in the Flutter app.

## Add a provider

1. New package under `internal/`, implement [provider.Provider](pkg/provider/provider.go).
2. Register it in [`DefaultRegistry`](pkg/resolve/resolve.go).
3. Cover match + parse with tests (use [testdata/](testdata/) when the QR is non-trivial).

Do not change the C ABI without updating CheckScan FFI (`packages/providers_native`).

## Conventions

- This repo is the library only. No client surface: UI, copy, localization, images, or other presentation. That lives in CheckScan.
- New code needs tests (`go test ./...`).
- Native logs: stderr by default. Host may set `checkscan_set_log`. Do not log tokens or full HTTP bodies.
- Keep files small. Split growing files; split a package when it starts doing more than one job.
