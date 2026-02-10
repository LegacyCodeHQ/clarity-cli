#!/usr/bin/env node

const fs = require('node:fs');
const path = require('node:path');
const os = require('node:os');
const crypto = require('node:crypto');
const { spawnSync } = require('node:child_process');
const https = require('node:https');
const { pipeline } = require('node:stream/promises');

const OWNER = 'LegacyCodeHQ';
const REPO = 'clarity';
const packageJson = require('../package.json');

function toTarget(platform, arch) {
  if (platform === 'darwin' && arch === 'x64') {
    return { os: 'darwin', arch: 'amd64', archiveExt: 'tar.gz', exeName: 'clarity' };
  }

  if (platform === 'darwin' && arch === 'arm64') {
    return { os: 'darwin', arch: 'arm64', archiveExt: 'tar.gz', exeName: 'clarity' };
  }

  if (platform === 'linux' && arch === 'x64') {
    return { os: 'linux', arch: 'amd64', archiveExt: 'tar.gz', exeName: 'clarity' };
  }

  if (platform === 'linux' && arch === 'arm64') {
    return { os: 'linux', arch: 'arm64', archiveExt: 'tar.gz', exeName: 'clarity' };
  }

  if (platform === 'win32' && arch === 'x64') {
    return { os: 'windows', arch: 'amd64', archiveExt: 'zip', exeName: 'clarity.exe' };
  }

  return null;
}

function download(url, destination, redirects = 0) {
  return new Promise((resolve, reject) => {
    const req = https.get(
      url,
      {
        headers: {
          'User-Agent': '@legacycodehq/clarity postinstall'
        }
      },
      async (res) => {
        if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
          res.resume();
          if (redirects > 5) {
            reject(new Error(`Too many redirects while downloading ${url}`));
            return;
          }

          try {
            await download(res.headers.location, destination, redirects + 1);
            resolve();
          } catch (err) {
            reject(err);
          }
          return;
        }

        if (res.statusCode !== 200) {
          const chunks = [];
          res.on('data', (chunk) => chunks.push(chunk));
          res.on('end', () => {
            reject(
              new Error(
                `Failed to download ${url}. HTTP ${res.statusCode}: ${Buffer.concat(chunks)
                  .toString('utf8')
                  .slice(0, 300)}`
              )
            );
          });
          return;
        }

        const out = fs.createWriteStream(destination);
        try {
          await pipeline(res, out);
          resolve();
        } catch (err) {
          reject(err);
        }
      }
    );

    req.on('error', reject);
  });
}

function sha256(filePath) {
  const hash = crypto.createHash('sha256');
  const data = fs.readFileSync(filePath);
  hash.update(data);
  return hash.digest('hex');
}

function parseChecksums(checksumPath) {
  const table = new Map();
  const content = fs.readFileSync(checksumPath, 'utf8');
  for (const line of content.split(/\r?\n/)) {
    if (!line.trim()) {
      continue;
    }

    const match = line.match(/^([a-f0-9]{64})\s+\*?(.+)$/i);
    if (!match) {
      continue;
    }

    table.set(match[2].trim(), match[1].toLowerCase());
  }

  return table;
}

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function extractArchive(archivePath, destination) {
  fs.mkdirSync(destination, { recursive: true });

  if (archivePath.endsWith('.tar.gz')) {
    const result = spawnSync('tar', ['-xzf', archivePath, '-C', destination], { stdio: 'inherit' });
    if (result.status !== 0) {
      throw new Error('Failed to extract tar.gz archive using tar');
    }
    return;
  }

  if (archivePath.endsWith('.zip')) {
    if (process.platform === 'win32') {
      throw new Error('Use extractArchiveWithRetry() for zip archives on Windows');
    }

    const result = spawnSync('unzip', ['-o', archivePath, '-d', destination], { stdio: 'inherit' });
    if (result.status !== 0) {
      throw new Error('Failed to extract zip archive using unzip');
    }
    return;
  }

  throw new Error(`Unsupported archive format: ${archivePath}`);
}

function findBinary(rootDir, exeName) {
  const stack = [rootDir];

  while (stack.length > 0) {
    const current = stack.pop();
    const entries = fs.readdirSync(current, { withFileTypes: true });

    for (const entry of entries) {
      const fullPath = path.join(current, entry.name);
      if (entry.isDirectory()) {
        stack.push(fullPath);
        continue;
      }

      if (entry.isFile() && entry.name === exeName) {
        return fullPath;
      }
    }
  }

  return null;
}

async function extractArchiveWithRetry(archivePath, destination) {
  const maxRetries = 5;
  const baseDelayMs = 500;

  for (let attempt = 1; attempt <= maxRetries; attempt++) {
    try {
      const result = spawnSync(
        'powershell.exe',
        ['-NoProfile', '-Command', `Expand-Archive -Path '${archivePath}' -DestinationPath '${destination}' -Force`],
        { stdio: 'inherit' }
      );

      if (result.status === 0) {
        return;
      }

      const stderr = (result.stderr || '').toString();
      const stdout = (result.stdout || '').toString();
      const output = `${stdout}\n${stderr}`;
      const isFileLockError =
        output.includes('being used by another process') ||
        output.includes('Access is denied') ||
        output.includes('cannot access the file');

      if (isFileLockError && attempt < maxRetries) {
        await sleep(baseDelayMs * Math.pow(2, attempt - 1));
        continue;
      }

      throw new Error('Failed to extract zip archive using PowerShell Expand-Archive');
    } catch (err) {
      if (attempt === maxRetries) {
        throw err;
      }
      await sleep(baseDelayMs * Math.pow(2, attempt - 1));
    }
  }
}

async function main() {
  if (process.env.CI) {
    console.log('Skipping binary download in CI environment');
    return;
  }

  const target = toTarget(process.platform, process.arch);
  if (!target) {
    throw new Error(`Unsupported platform/architecture: ${process.platform}/${process.arch}`);
  }

  const version = packageJson.version;
  if (!/^\d+\.\d+\.\d+([-.+].+)?$/.test(version)) {
    throw new Error(`Invalid package version '${version}'. Expected a release version like 0.1.0`);
  }

  const archiveName = `clarity_${version}_${target.os}_${target.arch}.${target.archiveExt}`;
  const checksumName = `clarity_${version}_checksums.txt`;
  const releaseBase = `https://github.com/${OWNER}/${REPO}/releases/download/v${version}`;

  const tmpRoot = fs.mkdtempSync(path.join(os.tmpdir(), 'clarity-npm-'));
  const archivePath = path.join(tmpRoot, archiveName);
  const checksumPath = path.join(tmpRoot, checksumName);
  const extractDir = path.join(tmpRoot, 'extract');

  console.log(`Installing clarity ${version} for ${process.platform}/${process.arch}...`);

  await download(`${releaseBase}/${checksumName}`, checksumPath);
  await download(`${releaseBase}/${archiveName}`, archivePath);

  const checksums = parseChecksums(checksumPath);
  const expected = checksums.get(archiveName);
  if (!expected) {
    throw new Error(`Missing checksum entry for ${archiveName}`);
  }

  const actual = sha256(archivePath);
  if (actual !== expected) {
    throw new Error(`Checksum mismatch for ${archiveName}: expected ${expected}, got ${actual}`);
  }

  if (target.archiveExt === 'zip' && process.platform === 'win32') {
    await extractArchiveWithRetry(archivePath, extractDir);
  } else {
    extractArchive(archivePath, extractDir);
  }

  const extractedBinary = findBinary(extractDir, target.exeName);
  if (!extractedBinary) {
    throw new Error(`Could not find extracted binary '${target.exeName}' in archive ${archiveName}`);
  }

  const outDir = path.resolve(__dirname, '..', 'bin');
  const outPath = path.join(outDir, target.exeName);
  fs.mkdirSync(outDir, { recursive: true });
  fs.copyFileSync(extractedBinary, outPath);

  if (process.platform !== 'win32') {
    fs.chmodSync(outPath, 0o755);
  }

  console.log(`Installed binary to ${outPath}`);
}

main().catch((err) => {
  console.error(`Failed to install clarity binary: ${err.message}`);
  process.exit(1);
});
