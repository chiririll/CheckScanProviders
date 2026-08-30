# CheckScanProviders

Go library for receipt QR providers. Used as a C-shared native library inside CheckScan.

Public API: `pkg/resolve` (`Match` / `Resolve`). C ABI: `include/checkscan.h`.

```
go test ./...
```

Android `.so` is built from the CheckScan app: `scripts/build_android_native.ps1`.
