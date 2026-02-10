#!/usr/bin/env node

const fs = require('node:fs');
const path = require('node:path');
const { spawn } = require('node:child_process');

const exeName = process.platform === 'win32' ? 'clarity.exe' : 'clarity';
const binaryPath = path.join(__dirname, exeName);

if (!fs.existsSync(binaryPath)) {
  console.error('Clarity binary is not installed. Reinstall the package and try again.');
  process.exit(1);
}

const child = spawn(binaryPath, process.argv.slice(2), {
  stdio: 'inherit'
});

child.on('error', (err) => {
  console.error(`Failed to start clarity binary: ${err.message}`);
  process.exit(1);
});

child.on('exit', (code, signal) => {
  if (signal) {
    process.kill(process.pid, signal);
    return;
  }

  process.exit(code ?? 1);
});
