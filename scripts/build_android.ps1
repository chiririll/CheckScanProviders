param(
    [string[]]$Abis = @("arm64-v8a", "x86_64")
)

$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
$jniRoot = Join-Path $root "flutter\android\src\main\jniLibs"

function Read-Prop([string]$Path, [string]$Key) {
    if (-not (Test-Path $Path)) { return $null }
    foreach ($line in Get-Content $Path) {
        if ($line -match "^\s*$([regex]::Escape($Key))=(.+)$") {
            return $Matches[1].Trim()
        }
    }
    return $null
}

function Get-NdkRoot {
    if ($env:ANDROID_NDK_HOME -and (Test-Path $env:ANDROID_NDK_HOME)) {
        return $env:ANDROID_NDK_HOME
    }
    if ($env:ANDROID_NDK_ROOT -and (Test-Path $env:ANDROID_NDK_ROOT)) {
        return $env:ANDROID_NDK_ROOT
    }
    $sdkCandidates = @()
    $propFiles = @(
        (Join-Path $root "local.properties"),
        (Join-Path $root "..\..\android\local.properties")
    )
    foreach ($localProps in $propFiles) {
        $ndk = Read-Prop $localProps "ndk.dir"
        if ($ndk) {
            $ndk = $ndk.Replace("\\", "\")
            if (Test-Path $ndk) { return $ndk }
        }
        $sdk = Read-Prop $localProps "sdk.dir"
        if ($sdk) { $sdkCandidates += $sdk.Replace("\\", "\") }
    }
    if ($env:ANDROID_HOME) { $sdkCandidates += $env:ANDROID_HOME }
    foreach ($sdk in $sdkCandidates) {
        $ndkDir = Join-Path $sdk "ndk"
        if (Test-Path $ndkDir) {
            $latest = Get-ChildItem $ndkDir -Directory | Sort-Object Name -Descending | Select-Object -First 1
            if ($latest) { return $latest.FullName }
        }
        $side = Join-Path $sdk "ndk-bundle"
        if (Test-Path $side) { return $side }
    }
    throw "NDK not found. Set ANDROID_NDK_HOME or ndk.dir in local.properties."
}

function Get-Clang([string]$Ndk, [string]$Triple) {
    $prebuilt = Join-Path $Ndk "toolchains\llvm\prebuilt"
    $hostDir = Get-ChildItem $prebuilt -Directory | Where-Object { $_.Name -match "windows" } | Select-Object -First 1
    if (-not $hostDir) {
        $hostDir = Get-ChildItem $prebuilt -Directory | Select-Object -First 1
    }
    if (-not $hostDir) { throw "NDK prebuilt toolchain not found in $prebuilt" }
    $clang = Join-Path $hostDir.FullName "bin\$Triple-clang.cmd"
    if (-not (Test-Path $clang)) {
        $clang = Join-Path $hostDir.FullName "bin\$Triple-clang"
    }
    if (-not (Test-Path $clang)) { throw "clang not found: $clang" }
    return $clang
}

$abiMap = @{
    "arm64-v8a" = @{ GoArch = "arm64"; Triple = "aarch64-linux-android24" }
    "x86_64"    = @{ GoArch = "amd64"; Triple = "x86_64-linux-android24" }
}

$ndk = Get-NdkRoot
Write-Host "NDK: $ndk"

foreach ($abi in $Abis) {
    $meta = $abiMap[$abi]
    if (-not $meta) { throw "Unsupported ABI $abi" }
    $outDir = Join-Path $jniRoot $abi
    New-Item -ItemType Directory -Force -Path $outDir | Out-Null
    $outSo = Join-Path $outDir "libcheckscan.so"
    $clang = Get-Clang $ndk $meta.Triple
    Write-Host "Building $abi with $clang"
    $env:GOOS = "android"
    $env:GOARCH = $meta.GoArch
    $env:CGO_ENABLED = "1"
    $env:CC = $clang
    $env:CGO_LDFLAGS = "-Wl,-soname,libcheckscan.so -llog"
    Push-Location $root
    try {
        go build -buildmode=c-shared -o $outSo ./cmd/lib
        if ($LASTEXITCODE -ne 0) { throw "go build failed for $abi" }
    } finally {
        Pop-Location
        Remove-Item Env:GOOS -ErrorAction SilentlyContinue
        Remove-Item Env:GOARCH -ErrorAction SilentlyContinue
        Remove-Item Env:CGO_ENABLED -ErrorAction SilentlyContinue
        Remove-Item Env:CC -ErrorAction SilentlyContinue
        Remove-Item Env:CGO_LDFLAGS -ErrorAction SilentlyContinue
    }
    $generatedH = [System.IO.Path]::ChangeExtension($outSo, ".h")
    if (Test-Path $generatedH) { Remove-Item $generatedH }
    Write-Host "Wrote $outSo"
}

$stamp = Join-Path $jniRoot ".native_version"
if (Test-Path $stamp) { Remove-Item $stamp }
Write-Host "Done."
