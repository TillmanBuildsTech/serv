#!/usr/bin/env node

// Thin shim: execs the platform binary that install.js downloaded into
// this same bin/ directory, passing through argv/stdio/exit code.

const path = require('path');
const os = require('os');
const { spawnSync } = require('child_process');

const binName = os.platform() === 'win32' ? 'serv.exe' : 'serv';
const binPath = path.join(__dirname, binName);

const result = spawnSync(binPath, process.argv.slice(2), { stdio: 'inherit' });

if (result.error) {
  if (result.error.code === 'ENOENT') {
    console.error(
      `serv: binary not found at ${binPath} — try reinstalling this package (npm install)`
    );
  } else {
    console.error(`serv: failed to run: ${result.error.message}`);
  }
  process.exit(1);
}

process.exit(result.status === null ? 1 : result.status);
