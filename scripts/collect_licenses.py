#!/usr/bin/env python3
"""Copy licenses from the actual Go module graph into shipped resources."""
import json
from pathlib import Path
import subprocess

root = Path(__file__).resolve().parent.parent
result = subprocess.run(["go", "list", "-m", "-json", "all"], cwd=root, check=True, capture_output=True, text=True)
decoder = json.JSONDecoder()
remaining = result.stdout
out = root / "resources" / "licenses"
out.mkdir(parents=True, exist_ok=True)
out.chmod(0o755)
while remaining.strip():
    module, end = decoder.raw_decode(remaining.lstrip())
    remaining = remaining.lstrip()[end:]
    if module.get("Main") or "Dir" not in module:
        continue
    name = module["Path"].replace("/", "_")
    for pattern in ("LICENSE*", "COPYING*", "NOTICE*"):
        for source in Path(module["Dir"]).glob(pattern):
            if source.is_file():
                (out / (name + "_" + source.name)).write_bytes(source.read_bytes())
(out / "MyToolbox_LICENSE").write_bytes((root / "LICENSE").read_bytes())
(out / "THIRD_PARTY_NOTICES.md").write_bytes((root / "THIRD_PARTY_NOTICES.md").read_bytes())
for item in out.iterdir():
    if item.is_file(): item.chmod(0o644)
