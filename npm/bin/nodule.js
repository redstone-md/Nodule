#!/usr/bin/env node
/**
 * Nodule — binary launcher.
 *
 * Mirrors the continuum-mcp pattern: exec the prebuilt binary with stdio
 * inherit, blocking until the child exits. All env vars (BYOM/BYOK) are
 * inherited from the parent process.
 *
 * If the platform binary is not present (e.g. no postinstall), downloads
 * it synchronously via curl before launching.
 */

const { spawnSync, spawn } = require('child_process');
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

function getLocalBinary() {
  const target = getTarget();
  if (!target) return null;
  const name = 'nodule-' + target.goos + '-' + target.goarch + target.ext;
  return path.join(__dirname, name);
}

function getVendorBinary() {
  // Mirror continuum's vendor layout: ../vendor/nodule[-windows-amd64].exe
  const target = getTarget();
  if (!target) return null;
  const name = 'nodule-' + target.goos + '-' + target.goarch + target.ext;
  return path.join(__dirname, '..', 'vendor', name);
}

function ensureBinary() {
  // 1. Local bin/<binary>
  let p = getLocalBinary();
  if (p && fs.existsSync(p)) return p;

  // 2. vendor/<binary> (continuum-style)
  p = getVendorBinary();
  if (p && fs.existsSync(p)) return p;

  // 3. PATH lookup (go install case)
  const isWindows = os.platform() === 'win32';
  return isWindows ? 'nodule.exe' : 'nodule';
}

function downloadSync() {
  const target = getTarget();
  if (!target) return false;

  // Use vendor directory (same convention as continuum-mcp)
  const vendorDir = path.join(__dirname, '..', 'vendor');
  try { fs.mkdirSync(vendorDir, { recursive: true }); } catch (e) {}
  const dest = path.join(vendorDir, 'nodule-' + target.goos + '-' + target.goarch + target.ext);

  if (fs.existsSync(dest)) return true;

  process.stderr.write('nodule: downloading binary for ' + target.goos + '/' + target.goarch + '...\n');

  const curlResult = spawnSync('curl', [
    '-s', '-L', '-H', 'User-Agent: nodule-installer',
    'https://api.github.com/repos/redstone-md/nodule/releases/latest'
  ], { encoding: 'utf8', timeout: 30000 });

  if (curlResult.status !== 0 || !curlResult.stdout) {
    process.stderr.write('nodule: failed to fetch release info\n');
    return false;
  }

  let release;
  try { release = JSON.parse(curlResult.stdout); }
  catch (e) {
    process.stderr.write('nodule: invalid release JSON\n');
    return false;
  }

  const asset = (release.assets || []).find(function(a) {
    return a.name === target.goos + '/' + target.goarch && false; // disabled, real match below
  });

  // Build asset name same as CI: nodule-<os>-<arch>[.exe]
  const assetName = 'nodule-' + target.goos + '-' + target.goarch + target.ext;
  const rightAsset = (release.assets || []).find(function(a) { return a.name === assetName; });
  if (!rightAsset) {
    process.stderr.write('nodule: asset ' + assetName + ' not found in ' + release.tag_name + '\n');
    return false;
  }

  const dlResult = spawnSync('curl', [
    '-s', '-L', '-o', dest,
    '-H', 'User-Agent: nodule-installer',
    '-H', 'Accept: application/octet-stream',
    rightAsset.browser_download_url
  ], { timeout: 60000 });

  if (dlResult.status !== 0) {
    process.stderr.write('nodule: download failed\n');
    try { fs.unlinkSync(dest); } catch (e) {}
    return false;
  }

  if (target.goos !== 'windows') {
    try { fs.chmodSync(dest, 0o755); } catch (e) {}
  }
  process.stderr.write('nodule: installed ' + release.tag_name + '\n');
  return true;
}

function main() {
  let binaryPath = ensureBinary();

  // If still not found, try downloading synchronously
  if (!fs.existsSync(binaryPath) && path.isAbsolute(binaryPath)) {
    if (downloadSync()) {
      binaryPath = ensureBinary();
    }
  }

  if (!fs.existsSync(binaryPath) && !path.isAbsolute(binaryPath)) {
    // PATH lookup — try go install
    const goResult = spawnSync('go', [
      'install', 'github.com/redstone-md/nodule/cmd/nodule@latest'
    ], { stdio: 'inherit', timeout: 120000 });
    if (goResult.status === 0) {
      binaryPath = os.platform() === 'win32' ? 'nodule.exe' : 'nodule';
    }
  }

  // Run the binary with stdio inherit (block until child exits)
  // Use spawnSync so we properly pipe stdio through (this is the
  // continuum-mcp pattern, which works on Windows + opencode).
  const result = spawnSync(binaryPath, process.argv.slice(2), { stdio: 'inherit' });
  if (result.error) {
    process.stderr.write('nodule: ' + result.error.message + '\n');
    process.exit(1);
  }
  if (result.signal) {
    process.exit(1);
  }
  process.exit(result.status === null ? 1 : result.status);
}

main();
