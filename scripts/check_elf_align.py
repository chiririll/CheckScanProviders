#!/usr/bin/env python3
"""Fail if any libcheckscan.so under ROOT has PT_LOAD align < 16 KiB."""

from __future__ import annotations

import pathlib
import struct
import sys

MIN_ALIGN = 16384


def load_aligns(data: bytes) -> list[int]:
    if data[:4] != b"\x7fELF":
        raise ValueError("not ELF")
    if data[4] != 2 or data[5] != 1:
        raise ValueError("expected ELF64 little-endian")
    e_phoff = struct.unpack_from("<Q", data, 32)[0]
    e_phentsize = struct.unpack_from("<H", data, 54)[0]
    e_phnum = struct.unpack_from("<H", data, 56)[0]
    aligns: list[int] = []
    for i in range(e_phnum):
        off = e_phoff + i * e_phentsize
        p_type = struct.unpack_from("<I", data, off)[0]
        if p_type != 1:
            continue
        aligns.append(struct.unpack_from("<Q", data, off + 48)[0])
    return aligns


def main() -> int:
    if len(sys.argv) != 2:
        print("usage: check_elf_align.py ROOT", file=sys.stderr)
        return 2
    root = pathlib.Path(sys.argv[1])
    files = sorted(root.glob("*/libcheckscan.so"))
    if not files:
        print(f"no libcheckscan.so under {root}", file=sys.stderr)
        return 1
    bad = False
    for path in files:
        aligns = load_aligns(path.read_bytes())
        ok = aligns and min(aligns) >= MIN_ALIGN
        print(f"{path}: {aligns} {'ok' if ok else 'UNALIGNED'}")
        bad = bad or not ok
    return 1 if bad else 0


if __name__ == "__main__":
    raise SystemExit(main())
