#!/usr/bin/env node
/**
 * Nodule — binary launcher.
 *
 * Runs the platform-specific Nodule MCP server binary with stdio passthrough.
 * If the binary is not present (first run, no postinstall), downloads it
 * synchronously before launching. All environment variables are inherited.
 */

const { spawn, spawnSync } = require('child_process');
const path = require('path');
const os = require('os');
const fs = require('fs');

function getTarget() {
  const platform = os.platform();
  const arch = os.arch();

  let goos, ext = '';
  switch (platform) {
    case 'linux':   goos = 'linux'; break;
    case 'darwin':  goos = 'darwin'; break;
    case 'win32':   goos = 'windows'; ext = '.exe'; break;
    default: return null;
  }

  let goarch;
  switch (arch) {
    case 'x64':   goarch = 'amd64'; break;
    case 'arm64': goarch = 'arm64'; break;
    case 'ia32':  goarch = '386'; break;
    default: return null;
  }

  return { goos: goos, goarch: goarch, ext: ext };
}

function getBinaryPath() {
  const target = getTarget();
  if (!target) return null;
  const binaryName = 'nodule-' + target.goos + '-' + target.goarch + target.ext;
  return path.join(__dirname, binaryName);
}

function ensureBinary() {
  let binaryPath = getBinaryPath();
  if (!binaryPath) {
    // Unsupported platform — try PATH lookup
    const isWindows = os.platform() === 'win32';
    return isWindows ? 'nodule.exe' : 'nodule';
  }

  // Binary already present
  if (fs.existsSync(binaryPath)) {
    return binaryPath;
  }

  // Binary missing — download synchronously before launching
  process.stderr.write('nodule: first run, downloading binary...\n');
  try {
    var installer = require('./install.js');
    installer.installSync();
  } catch (e) {
    // ignore
  }

  // Re-check after install attempt
  if (fs.existsSync(binaryPath)) {
    return binaryPath;
  }

  // Final fallback: PATH lookup (go install case)
  const isWindows = os.platform() === 'win32';
  return isWindows ? 'nodule.exe' : 'nodule';
}

function main() {
  const binaryPath = ensureBinary();

  const child = spawn(binaryPath, process.argv.slice(2), {
    stdio: 'inherit',
    env: process.env,
  });

  child.on('error', (err) => {
    if (err.code === 'ENOENT') {
      process.stderr.write(
        'nodule: binary not found. Install with:\n' +
        '  npm install @redstone-md/nodule\n' +
        'or:\n' +
        '  go install github.com/redstone-md/nodule/cmd/nodule@latest\n'
      );
    } else {
      process.stderr.write('nodule: ' + err.message + '\n');
    }
    process.exit(1);
  });

  child.on('exit', (code, signal) => {
    if (signal) {
      process.kill(process.pid, signal);
    } else {
      process.exit(code == null ? 1 : code);
    }
  });
}

main();
