# CheckScanProviders

Go library: match/resolve a receipt QR into eq receipt JSON. Embedded in CheckScan as a C-shared library.

Overview and `go test ./...`: [README.md](README.md).

## Map

| Path | What |
| --- | --- |
| [pkg/resolve](pkg/resolve/resolve.go) | Public API: `Match` / `Resolve`, default registry |
| [pkg/provider](pkg/provider/provider.go) | `Provider` interface |
| [pkg/eq](pkg/eq/receipt.go) | Receipt JSON model |
| [include/checkscan.h](include/checkscan.h) | C ABI (`checkscan_match` / `resolve` / `providers`) |
| [cmd/lib](cmd/lib/main.go) | c-shared exports |
| [internal/](internal/) | One package per provider |
| [testdata/](testdata/) | QR fixtures |

New formats live here, not in the Flutter app.

## Add a provider

1. New package under `internal/`, implement [provider.Provider](pkg/provider/provider.go).
2. Register it in [`DefaultRegistry`](pkg/resolve/resolve.go).
3. Cover match + parse with tests (use [testdata/](testdata/) when the QR is non-trivial).

Do not change the C ABI without updating CheckScan FFI (`packages/providers_native`).

## Conventions

- New code needs tests (`go test ./...`).
- Keep files small. Split growing files; split a package when it starts doing more than one job.
