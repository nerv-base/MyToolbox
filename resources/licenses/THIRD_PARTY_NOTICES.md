# Third-party components

MyToolbox!!!!! is licensed under **AGPL-3.0-only**.

- **MyGo** — MIT. https://github.com/egoist/mygo
  Pinned to the version in `go.mod`; upstream license:
  https://github.com/egoist/mygo/blob/main/LICENSE
- **Ghostscript 10.08.0** — AGPL-3.0-only (Artifex Software).
  https://ghostscript.com/licensing/
  The complete license is included in each bundled `ghostscript/LICENSE`.
  Exact corresponding source:
  https://github.com/ArtifexSoftware/ghostpdl-downloads/releases/download/gs10080/ghostscript-10.08.0.tar.xz
  `scripts/prepare_engine.py` verifies the source and Windows installer using
  pinned SHA-512 checksums published with this release. Build artifacts include
  the unmodified source archive and the MyToolbox source and build scripts.

Go transitive dependency licenses are collected at build time in
`resources/licenses/` by `scripts/collect_licenses.py` and ship inside the app.
Consult each component's license for its terms. No commercial Ghostscript
license is included or required for distribution compliant with the AGPL.
