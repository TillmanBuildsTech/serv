#!/usr/bin/env python3
"""Bump the serv version + add a CHANGELOG entry.

Usage:
    python3 scripts/bump_version.py 0.2.0 [-m "summary of the release"]

What it does:
  - Writes the new version to internal/version/VERSION.
  - Inserts a new top entry in CHANGELOG.md under the "# Changelog" heading,
    with an "Unreleased" placeholder unless -m/--message is given.
  - Refuses to run if the new version already has a git tag (CI-equivalent
    local guard) or if it is <= the current VERSION.

The bump is a maintainer action taken at the dev -> release cut (see
docs/release-process.md). After this, commit the changes and merge to main;
the release workflow ships exactly this version.
"""

import argparse
import re
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
VERSION_FILE = ROOT / "internal" / "version" / "VERSION"
CHANGELOG_FILE = ROOT / "CHANGELOG.md"
SEMVER = re.compile(r"^\d+\.\d+\.\d+$")


def current_version() -> str:
    return VERSION_FILE.read_text().strip()


def tag_exists(version: str) -> bool:
    try:
        out = subprocess.run(
            ["git", "ls-remote", "--tags", "origin", f"v{version}"],
            capture_output=True,
            text=True,
            timeout=30,
        ).stdout
        return bool(out.strip())
    except Exception:
        # Can't reach origin (e.g. offline). Don't hard-fail on the local
        # check; the CI version-guard is authoritative.
        return False


def bump(new_version: str, message: str | None) -> None:
    cur = current_version()

    if not SEMVER.match(new_version):
        sys.exit(f"error: '{new_version}' is not a valid X.Y.Z version")

    # Compare numerically
    def key(v):
        return tuple(int(x) for x in v.split("."))

    if key(new_version) <= key(cur):
        sys.exit(f"error: {new_version} is not greater than current {cur}")

    if tag_exists(new_version):
        sys.exit(
            f"error: v{new_version} already has a tag on origin. Pick a new version."
        )

    # 1. VERSION file
    VERSION_FILE.write_text(new_version + "\n")
    print(f"VERSION -> {new_version}")

    # 2. CHANGELOG entry
    notes = message or "Unreleased."
    changelog = CHANGELOG_FILE.read_text()
    # Find the "# Changelog" heading and insert after it
    marker = "# Changelog\n"
    if marker not in changelog:
        sys.exit("error: could not find '# Changelog' heading in CHANGELOG.md")
    entry = (
        f"\n## [{new_version}]\n\n### Added\n\n- {notes}\n"
    )
    new_changelog = changelog.replace(marker, marker + entry, 1)
    CHANGELOG_FILE.write_text(new_changelog)
    print(f"CHANGELOG.md -> added [{new_version}]")


if __name__ == "__main__":
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("version", help="new X.Y.Z version, e.g. 0.2.0")
    ap.add_argument("-m", "--message", help="one-line summary for the CHANGELOG entry")
    args = ap.parse_args()
    bump(args.version, args.message)
