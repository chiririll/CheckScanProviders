#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
JNI_ROOT="$ROOT/adapters/flutter/android/src/main/jniLibs"
if [[ $# -eq 0 ]]; then
  ABIS=(arm64-v8a x86_64)
else
  ABIS=("$@")
fi

read_prop() {
  local file="$1" key="$2"
  [[ -f "$file" ]] || return 0
  while IFS= read -r line || [[ -n "$line" ]]; do
    if [[ "$line" =~ ^[[:space:]]*${key}=(.*)$ ]]; then
      echo "${BASH_REMATCH[1]}" | tr -d '\r'
      return 0
    fi
  done < "$file"
}

find_ndk() {
  if [[ -n "${ANDROID_NDK_HOME:-}" && -d "$ANDROID_NDK_HOME" ]]; then
    echo "$ANDROID_NDK_HOME"
    return
  fi
  if [[ -n "${ANDROID_NDK_ROOT:-}" && -d "$ANDROID_NDK_ROOT" ]]; then
    echo "$ANDROID_NDK_ROOT"
    return
  fi
  local props=("$ROOT/local.properties" "$ROOT/../../android/local.properties")
  local sdk_candidates=()
  local prop
  for prop in "${props[@]}"; do
    local ndk
    ndk="$(read_prop "$prop" "ndk.dir")"
    if [[ -n "$ndk" && -d "$ndk" ]]; then
      echo "$ndk"
      return
    fi
    local sdk
    sdk="$(read_prop "$prop" "sdk.dir")"
    if [[ -n "$sdk" ]]; then
      sdk_candidates+=("$sdk")
    fi
  done
  if [[ -n "${ANDROID_HOME:-}" ]]; then
    sdk_candidates+=("$ANDROID_HOME")
  fi
  local sdk
  for sdk in "${sdk_candidates[@]}"; do
    if [[ -d "$sdk/ndk" ]]; then
      local latest
      latest="$(ls -1 "$sdk/ndk" | sort -V | tail -n 1)"
      if [[ -n "$latest" && -d "$sdk/ndk/$latest" ]]; then
        echo "$sdk/ndk/$latest"
        return
      fi
    fi
    if [[ -d "$sdk/ndk-bundle" ]]; then
      echo "$sdk/ndk-bundle"
      return
    fi
  done
  echo "NDK not found. Set ANDROID_NDK_HOME or ndk.dir in local.properties." >&2
  exit 1
}

find_clang() {
  local ndk="$1" triple="$2"
  local prebuilt="$ndk/toolchains/llvm/prebuilt"
  local host clang
  for host in linux-x86_64 darwin-arm64 darwin-x86_64 windows-x86_64; do
    clang="$prebuilt/$host/bin/${triple}-clang"
    if [[ -x "$clang" ]]; then
      echo "$clang"
      return
    fi
  done
  local first
  first="$(ls -1 "$prebuilt" 2>/dev/null | head -n 1)"
  clang="$prebuilt/$first/bin/${triple}-clang"
  if [[ -x "$clang" ]]; then
    echo "$clang"
    return
  fi
  echo "clang not found for $triple under $prebuilt" >&2
  exit 1
}

go_arch_for() {
  case "$1" in
    arm64-v8a) echo arm64 ;;
    x86_64) echo amd64 ;;
    *) echo "Unsupported ABI $1" >&2; exit 1 ;;
  esac
}

triple_for() {
  case "$1" in
    arm64-v8a) echo aarch64-linux-android24 ;;
    x86_64) echo x86_64-linux-android24 ;;
    *) echo "Unsupported ABI $1" >&2; exit 1 ;;
  esac
}

NDK="$(find_ndk)"
echo "NDK: $NDK"

for abi in "${ABIS[@]}"; do
  out_dir="$JNI_ROOT/$abi"
  mkdir -p "$out_dir"
  out_so="$out_dir/libcheckscan.so"
  clang="$(find_clang "$NDK" "$(triple_for "$abi")")"
  echo "Building $abi with $clang"
  (
    cd "$ROOT"
    export GOOS=android
    export GOARCH="$(go_arch_for "$abi")"
    export CGO_ENABLED=1
    export CC="$clang"
    export CGO_LDFLAGS="-Wl,-soname,libcheckscan.so -llog"
    go build -buildmode=c-shared -o "$out_so" ./cmd/lib
  )
  rm -f "${out_so%.so}.h"
  echo "Wrote $out_so"
done

rm -f "$JNI_ROOT/.native_version"
echo "Done."
