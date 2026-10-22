#!/usr/bin/env python3
"""Fetch a pinned, verified Ghostscript and stage MyGo platform resources.

Run on the target OS/architecture. Unix builds use the corresponding source
archive; Windows uses Artifex's x64 installer extracted without installation.
The exact unmodified source archive accompanies every distributable.
"""
from __future__ import annotations

import argparse
import hashlib
import os
from pathlib import Path
import platform
import shutil
import subprocess
import tarfile
import urllib.request

ROOT = Path(__file__).resolve().parent.parent
VERSION = "10.08.0"
BASE = "https://github.com/ArtifexSoftware/ghostpdl-downloads/releases/download/gs10080/"
ASSETS = {
    "ghostscript-10.08.0.tar.xz": "8006e2a32d03759a905b9548bdd83d4563173041006750e17e428b9eea24ad519aa4452ceecd7c073d3c420a68bf1b07ccc9e0533b1a05935f6b39ea8d9ce875",
    "gs10080w64.exe": "cb3ecc798508851ba28b05e3fb914ddefe78168190726376d8581546ce61d5b216ffa3359e6f829072786bc12350501c7b6f99110d578696b87213d157774269",
}


def run(*args: str, cwd: Path | None = None) -> None:
    print("+", *args, flush=True)
    subprocess.run(args, cwd=cwd, check=True)


def fetch(name: str) -> Path:
    cache = ROOT / ".cache" / "downloads"
    cache.mkdir(parents=True, exist_ok=True)
    dest = cache / name
    if not dest.exists():
        temporary = dest.with_suffix(dest.suffix + ".part")
        print("Downloading", name, flush=True)
        request = urllib.request.Request(BASE + name, headers={"User-Agent": "MyToolbox-build"})
        with urllib.request.urlopen(request, timeout=120) as source, temporary.open("wb") as out:
            shutil.copyfileobj(source, out)
        temporary.replace(dest)
    with dest.open("rb") as stream:
        digest = hashlib.file_digest(stream, "sha512").hexdigest()
    if digest != ASSETS[name]:
        raise RuntimeError(f"Checksum mismatch for {name}; remove {dest} and investigate")
    return dest


def host_target() -> str:
    system = {"Darwin": "darwin", "Linux": "linux", "Windows": "windows"}[platform.system()]
    arch = {"x86_64": "amd64", "AMD64": "amd64", "aarch64": "arm64", "arm64": "arm64"}[platform.machine()]
    return system + "-" + arch


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--target", default=host_target())
    args = parser.parse_args()
    if args.target != host_target():
        parser.error("Build the engine on a runner matching --target")
    if args.target == "windows-arm64":
        parser.error("This release supports Windows x64; no official ARM64 engine is available")

    archive = fetch("ghostscript-10.08.0.tar.xz")
    source_root = ROOT / ".cache" / ("ghostscript-source-" + args.target)
    source = source_root / ("ghostscript-" + VERSION)
    if not source.exists():
        source_root.mkdir(parents=True, exist_ok=True)
        with tarfile.open(archive) as tar:
            tar.extractall(source_root, filter="data")

    destination = ROOT / "resources" / args.target / "ghostscript"
    if destination.exists():
        shutil.rmtree(destination)
    destination.mkdir(parents=True)
    if args.target.startswith("windows-"):
        installer = fetch("gs10080w64.exe")
        extracted = ROOT / ".cache" / "ghostscript-windows"
        if extracted.exists():
            shutil.rmtree(extracted)
        run("7z", "x", "-y", "-o" + str(extracted), str(installer))
        candidates = list(extracted.rglob("gswin64c.exe"))
        if len(candidates) != 1:
            raise RuntimeError("Unexpected Ghostscript installer layout")
        runtime_root = candidates[0].parent.parent
        for name in ("bin", "lib", "Resource"):
            item = runtime_root / name
            if item.exists():
                shutil.copytree(item, destination / name)
        executable = destination / "bin" / "gswin64c.exe"
    else:
        if not (source / "Makefile").exists():
            run("./configure", "--without-x", "--disable-cups", "--without-tesseract",
                "--disable-dbus", "--disable-fontconfig", "--without-libidn",
                "--without-libpaper", cwd=source)
        run("make", "-j" + str(min(os.cpu_count() or 2, 4)), cwd=source)
        (destination / "bin").mkdir()
        executable = destination / "bin" / "gs"
        shutil.copy2(source / "bin" / "gs", executable)
        for name in ("Resource", "lib"):
            shutil.copytree(source / name, destination / name)
    shutil.copy2(source / "LICENSE", destination / "LICENSE")
    for directory in (ROOT / "resources", destination.parent, destination):
        directory.chmod(0o755)
    for item in destination.rglob("*"):
        item.chmod(0o755 if item.is_dir() else 0o644)
    executable.chmod(0o755)
    run(str(executable), "--version")

    # Keep the exact corresponding source beside the app in uploaded artifacts.
    sources = ROOT / "build" / "sources"
    sources.mkdir(parents=True, exist_ok=True)
    shutil.copy2(archive, sources / archive.name)
    print("Engine ready:", destination, flush=True)


if __name__ == "__main__":
    main()
