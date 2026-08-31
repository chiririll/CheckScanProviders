# CheckScanProviders

Go library for receipt QR providers. Used as a C-shared native library inside CheckScan.

Public API: `pkg/resolve` (`Match` / `Resolve`). Host bindings live under `adapters/`: C ABI in `adapters/c/include/checkscan.h`, Flutter FFI in `adapters/flutter/` (`providers_native`).

```
go test ./...
```

## Android library

Tag `vX.Y.Z` (same as `adapters/flutter/pubspec.yaml` `version`) and push. [Release](.github/workflows/release.yml) builds `libcheckscan.so` and uploads `android-jniLibs.zip`.

The Flutter plugin downloads that zip on `preBuild` when local `.so` files are missing.

Local rebuild (Go + Android NDK):

```
./scripts/build_android.sh
./scripts/build_android.ps1
```

Needs Go 1.22+ and an NDK (`ANDROID_NDK_HOME` or `ndk.dir` in `local.properties`). Native JSON is `{status, message, data}` (`status` is HTTP-like). Settings schema: `checkscan_settings`. Secrets stay out of the `.so`; the host calls `checkscan_set_config`.
