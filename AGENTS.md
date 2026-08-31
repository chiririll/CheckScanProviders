# CheckScanProviders

Go library: match/resolve a receipt QR into eq receipt JSON. Embedded in CheckScan as a C-shared library. Flutter FFI lives in this repo.

Overview and `go test ./...`: [README.md](README.md).

## Map

| Path | What |
| --- | --- |
| [pkg/resolve](pkg/resolve/resolve.go) | Public API: `Match` / `Resolve`, default registry |
| [pkg/provider](pkg/provider/provider.go) | `Provider` interface |
| [pkg/eq](pkg/eq/receipt.go) | Receipt JSON model |
| [adapters/c/include/checkscan.h](adapters/c/include/checkscan.h) | C adapter: ABI (`checkscan_match` / `resolve` / `settings` / `set_config` / `set_log`) |
| [cmd/lib](cmd/lib/main.go) | c-shared implementation of the C adapter |
| [internal/nativelog](internal/nativelog/nativelog.go) | Library logs: stderr, or host `checkscan_set_log` |
| [internal/](internal/) | One package per provider |
| [testdata/](testdata/) | QR fixtures |
| [adapters/flutter/](adapters/flutter/) | Flutter adapter: FFI plugin (`providers_native`) |
| [scripts/build_android.sh](scripts/build_android.sh) | Local Android `.so` build (also [`.ps1`](scripts/build_android.ps1)) |
| [.github/workflows/release.yml](.github/workflows/release.yml) | Tag `v*` → GitHub Release with `android-jniLibs.zip` |

`Parse` is local unless `provider.Remote(ctx)` is true (scan and refresh pass `remote`). `internal/httplimit` is a per-host request budget (sliding windows + 403/429/503 cooldown). It is not receipt storage: CheckScan already dedupes by QR hash in SQLite before `resolve`. Gate timestamps may be written under `TMPDIR` so a new isolate still sees the budget. A successful receipt with no line items sets `checkscan.items_unavailable`; an empty list without that flag is still incomplete. `checkscan_resolve` may receive the stored receipt JSON; resolve keeps it unless the new parse is richer (more items, merchant, tax id, or a non-zero total).

New formats live here, not in the Flutter app.

## Add a provider

1. New package under `internal/`, implement [provider.Provider](pkg/provider/provider.go).
2. Register it in [`DefaultRegistry`](pkg/resolve/resolve.go).
3. Cover match + parse with tests (use [testdata/](testdata/) when the QR is non-trivial).
4. If the provider needs a host secret, implement `provider.HasSecrets` and read it via `nativecfg.Get(id + "." + secretID)`.

Do not change the C ABI in [adapters/c/](adapters/c/) without updating the Flutter FFI in [adapters/flutter/](adapters/flutter/).

## Native release

1. Bump `version` in [adapters/flutter/pubspec.yaml](adapters/flutter/pubspec.yaml).
2. Tag `vX.Y.Z` and push. CI uploads `android-jniLibs.zip`.
3. CheckScan's plugin `preBuild` fetches that asset when local `.so` files are missing.

Android `.so` files must be 16 KB ELF-aligned (`-Wl,-z,max-page-size=16384`). CI checks this with [scripts/check_elf_align.py](scripts/check_elf_align.py).

JSON replies use `{status, message, data}` with HTTP-like status classes (`pkg/status`: 200/206/4xx/5xx). `checkscan_settings` lists `{fields:[{key,type,label}]}`. The host stores values and calls `checkscan_set_config`. Do not log secret values.

## Conventions

- This repo is the library plus Flutter FFI bindings. No client surface: UI, copy, localization, images, or other presentation. That lives in CheckScan.
- New code needs tests (`go test ./...`).
- Native logs: stderr by default. Host may set `checkscan_set_log`. Do not log tokens or full HTTP bodies.
- Keep files small. Split growing files; split a package when it starts doing more than one job.
