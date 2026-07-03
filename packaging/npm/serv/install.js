#!/usr/bin/env node

// Downloads the serv release archive matching this package's version and
// the current platform, verifies it against the pinned checksums.json, and
// extracts the binary into bin/. Runs as a postinstall hook; bin/serv.js
// execs the resulting binary at invocation time.

const fs = require('fs');
const path = require('path');
const os = require('os');
const https = require('https');
const crypto = require('crypto');
const { execFileSync } = require('child_process');

const REPO = 'TillmanBuildsTech/serv';
const PACKAGE_DIR = __dirname;
const BIN_DIR = path.join(PACKAGE_DIR, 'bin');

function archiveFilename(version) {
  const platform = os.platform() === 'win32' ? 'windows' : os.platform();
  const archMap = { x64: 'amd64', arm64: 'arm64' };
  const arch = archMap[os.arch()];
  if (!['windows', 'darwin', 'linux'].includes(platform) || !arch) {
    throw new Error(`Unsupported platform: ${os.platform()}/${os.arch()}`);
  }
  const ext = platform === 'windows' ? 'zip' : 'tar.gz';
  return { platform, filename: `serv-${platform}-${arch}.${ext}` };
}

function download(url) {
  return new Promise((resolve, reject) => {
    https
      .get(url, { headers: { 'User-Agent': 'serv-npm-installer' } }, (res) => {
        if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
          download(res.headers.location).then(resolve, reject);
          return;
        }
        if (res.statusCode !== 200) {
          reject(new Error(`GET ${url} failed: HTTP ${res.statusCode}`));
          return;
        }
        const chunks = [];
        res.on('data', (chunk) => chunks.push(chunk));
        res.on('end', () => resolve(Buffer.concat(chunks)));
      })
      .on('error', reject);
  });
}

async function main() {
  const { version } = require('./package.json');
  const checksums = require('./checksums.json');
  const { platform, filename } = archiveFilename(version);

  const expected = checksums[filename];
  if (!expected || expected.startsWith('REPLACE_WITH_')) {
    throw new Error(`No pinned checksum for ${filename} in checksums.json`);
  }

  const url = `https://github.com/${REPO}/releases/download/v${version}/${filename}`;
  console.error(`serv: downloading ${filename} for v${version}...`);
  const archive = await download(url);

  const actual = crypto.createHash('sha256').update(archive).digest('hex');
  if (actual.toLowerCase() !== expected.toLowerCase()) {
    throw new Error(
      `Checksum mismatch for ${filename}: expected ${expected}, got ${actual}`
    );
  }

  fs.mkdirSync(BIN_DIR, { recursive: true });
  const archivePath = path.join(os.tmpdir(), `serv-install-${Date.now()}-${filename}`);
  fs.writeFileSync(archivePath, archive);

  try {
    // Windows 10 1803+ ships tar.exe (bsdtar), which extracts both zip and
    // tar.gz; GNU tar on Linux and bsdtar on macOS both auto-detect gzip.
    execFileSync('tar', ['-xf', archivePath, '-C', BIN_DIR], { stdio: 'inherit' });
  } finally {
    fs.rmSync(archivePath, { force: true });
  }

  const binName = platform === 'windows' ? 'serv.exe' : 'serv';
  const binPath = path.join(BIN_DIR, binName);
  if (!fs.existsSync(binPath)) {
    throw new Error(`Extraction did not produce ${binPath}`);
  }
  if (platform !== 'windows') {
    fs.chmodSync(binPath, 0o755);
  }

  console.error('serv: installed successfully');
}

main().catch((err) => {
  console.error(`serv: install failed: ${err.message}`);
  process.exit(1);
});
