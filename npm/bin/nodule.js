#!/usr/bin/env node
/**
 * Nodule — binary launcher.
 *
 * Runs the platform-specific Nodule MCP server binary with stdio passthrough.
 * This is a thin wrapper: the Go binary handles all MCP protocol logic.
 *
 * All environment variables (NODULE_LLM_PROVIDER, NODULE_API_KEY, etc.)
 * are inherited from the parent process — Nodule is fully BYOM/BYOK.
 */

const { spawn } = require('child_process');
const path = require('path');
const os = require('os');
const fs = require('fs');

function getBinaryPath() {
  const platform = os.platform();
  const arch = os.arch();

  let goos, ext = '';
  switch (platform) {
    case 'linux':   goos = 'linux'; break;
    case 'darwin':  goos = 'darwin'; break;
    case 'win32':   goos = 'windows'; ext = '.exe'; break;
    default: throw new Error(`unsupported platform: ${platform}`);
  }

  let goarch;
  switch (arch) {
    case 'x64':   goarch = 'amd64'; break;
    case 'arm64': goarch = 'arm64'; break;
    case 'ia32':  goarch = '386'; break;
    default: throw new Error(`unsupported arch: ${arch}`);
  }

  const binaryName = `nodule-${goos}-${goarch}${ext}`;
  return path.join(__dirname, binaryName);
}

function main() {
  let binaryPath = getBinaryPath();

  // If the platform binary doesn't exist, try PATH lookup (go install case)
  if (!fs.existsSync(binaryPath)) {
    const isWindows = os.platform() === 'win32';
    binaryPath = isWindows ? 'nodule.exe' : 'nodule';
  }

  const child = spawn(binaryPath, process.argv.slice(2), {
    stdio: 'inherit',
    env: process.env,
  });

  child.on('error', (err) => {
    if (err.code === 'ENOENT') {
      process.stderr.write(
        'nodule: binary not found. Reinstall with `npm install nodule` or `go install github.com/redstone-md/nodule/cmd/nodule@latest`\n'
      );
    } else {
      process.stderr.write(`nodule: ${err.message}\n`);
    }
    process.exit(1);
  });

  child.on('exit', (code, signal) => {
    if (signal) {
      process.kill(process.pid, signal);
    } else {
      process.exit(code ?? 1);
    }
  });
}

main();
