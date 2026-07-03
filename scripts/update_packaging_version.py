#!/usr/bin/env python3
"""Bump version + release-artifact SHA256s in the packaging manifests.

Run after release artifacts exist locally, e.g.:

    python3 scripts/update_packaging_version.py 0.1.1 out

`out/` must contain the six release archives named as the Release workflow
produces them (serv-<goos>-<goarch>.{tar.gz,zip}). Bumps Homebrew, Scoop,
winget, and the npm wrapper package.
"""

import hashlib
import json
import os
import re
import sys

ARTIFACTS = {
    "serv-darwin-arm64.tar.gz",
    "serv-darwin-amd64.tar.gz",
    "serv-linux-arm64.tar.gz",
    "serv-linux-amd64.tar.gz",
    "serv-windows-amd64.zip",
    "serv-windows-arm64.zip",
}

FILENAME_RE = re.compile(r"([\w.-]+\.(?:tar\.gz|zip))")

# Only bump the *package* version, never incidental semver-shaped strings
# like a winget "ManifestVersion: 1.6.0" or a schema URL's "...1.6.0...".
# Homebrew uses Ruby's `version "X.Y.Z"` (no colon); winget uses YAML's
# `PackageVersion: X.Y.Z`.
VERSION_FIELD_RE = re.compile(r'^(\s*(?:version\s+|PackageVersion:\s*)"?)\d+\.\d+\.\d+("?\s*)$')
URL_VERSION_RE = re.compile(r"(/v)\d+\.\d+\.\d+(/)")


def bump_version_in_line(line, version):
    line = VERSION_FIELD_RE.sub(rf"\g<1>{version}\g<2>", line)
    line = URL_VERSION_RE.sub(rf"\g<1>{version}\g<2>", line)
    return line


def sha256_of(path):
    h = hashlib.sha256()
    with open(path, "rb") as f:
        for chunk in iter(lambda: f.read(65536), b""):
            h.update(chunk)
    return h.hexdigest()


def load_hashes(artifact_dir):
    hashes = {}
    for fname in ARTIFACTS:
        path = os.path.join(artifact_dir, fname)
        if not os.path.isfile(path):
            raise SystemExit(f"missing release artifact: {path}")
        hashes[fname] = sha256_of(path)
    return hashes


def update_url_and_next_line_field(path, version, hashes, field_pattern):
    """Bump semver on every line, and on any line whose URL references a
    known artifact filename, rewrite the field on the following line
    (assumed to hold that artifact's checksum) with the freshly computed
    hash. Used for the Ruby/YAML manifests, which aren't machine-editable
    formats."""
    with open(path, encoding="utf-8") as f:
        lines = f.readlines()

    out = []
    i = 0
    while i < len(lines):
        line = bump_version_in_line(lines[i], version)
        out.append(line)
        m = FILENAME_RE.search(line)
        if m and m.group(1) in hashes and i + 1 < len(lines) and field_pattern.search(lines[i + 1]):
            out.append(field_pattern.sub(rf"\g<1>{hashes[m.group(1)]}\g<2>", lines[i + 1]))
            i += 1
        i += 1

    with open(path, "w", encoding="utf-8") as f:
        f.writelines(out)


def update_homebrew(version, hashes):
    update_url_and_next_line_field(
        "packaging/homebrew/serv.rb",
        version,
        hashes,
        re.compile(r'(sha256 ")[^"]*(")'),
    )


def update_winget(version, hashes):
    for path in [
        "packaging/winget/TillmanBuildsTech.Serv.installer.yaml",
        "packaging/winget/TillmanBuildsTech.Serv.locale.en-US.yaml",
        "packaging/winget/TillmanBuildsTech.Serv.yaml",
    ]:
        update_url_and_next_line_field(
            path,
            version,
            hashes,
            re.compile(r"(InstallerSha256:\s*)\S+()"),
        )


def update_scoop(version, hashes):
    path = "packaging/scoop/serv.json"
    with open(path, encoding="utf-8") as f:
        data = json.load(f)

    data["version"] = version
    arch_files = {"64bit": "serv-windows-amd64.zip", "arm64": "serv-windows-arm64.zip"}
    for arch, fname in arch_files.items():
        data["architecture"][arch]["url"] = (
            f"https://github.com/TillmanBuildsTech/serv/releases/download/v{version}/{fname}"
        )
        data["architecture"][arch]["hash"] = hashes[fname]

    with open(path, "w", encoding="utf-8") as f:
        json.dump(data, f, indent=4)
        f.write("\n")


def update_npm(version, hashes):
    pkg_path = "packaging/npm/serv/package.json"
    with open(pkg_path, encoding="utf-8") as f:
        pkg = json.load(f)
    pkg["version"] = version
    with open(pkg_path, "w", encoding="utf-8") as f:
        json.dump(pkg, f, indent=4)
        f.write("\n")

    checksums_path = "packaging/npm/serv/checksums.json"
    with open(checksums_path, encoding="utf-8") as f:
        checksums = json.load(f)
    checksums["version"] = version
    for fname in checksums:
        if fname in hashes:
            checksums[fname] = hashes[fname]
    with open(checksums_path, "w", encoding="utf-8") as f:
        json.dump(checksums, f, indent=4)
        f.write("\n")


def main():
    if len(sys.argv) != 3:
        raise SystemExit(f"usage: {sys.argv[0]} <version> <artifact-dir>")
    version, artifact_dir = sys.argv[1], sys.argv[2]

    hashes = load_hashes(artifact_dir)
    update_homebrew(version, hashes)
    update_scoop(version, hashes)
    update_winget(version, hashes)
    update_npm(version, hashes)


if __name__ == "__main__":
    main()
