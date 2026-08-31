# CheckScanProviders

Go library for receipt QR providers. Used as a C-shared native library inside CheckScan.

Public API: `pkg/resolve` (`Match` / `Resolve`). C ABI: `include/checkscan.h`.
Flutter FFI plugin: `flutter/` (`providers_native`).

```
go test ./...
```

## Android library

Tag `vX.Y.Z` (same as `flutter/pubspec.yaml` `version`) and push. [Release](.github/workflows/release.yml) builds `libcheckscan.so` and uploads `android-jniLibs.zip`.

The Flutter plugin downloads that zip on `preBuild` when local `.so` files are missing.

Local rebuild (Go + Android NDK):

```
./scripts/build_android.sh
./scripts/build_android.ps1
```

Needs Go 1.22+ and an NDK (`ANDROID_NDK_HOME` or `ndk.dir` in `local.properties`). Provider secrets (tokens) are not baked into the `.so`: the host calls `checkscan_set_config` with values from `checkscan_providers` `secrets`.
