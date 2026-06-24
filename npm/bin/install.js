#!/usr/bin/env node
/**
 * Nodule — postinstall / on-demand binary downloader.
 *
 * Downloads the platform-appropriate Nodule binary from the latest GitHub
 * Release. Supports both async (postinstall) and sync (first-run from nodule.js)
 * modes. Falls back to `go install` if download fails.
 */

const https = require('https');
const fs = require('fs');
const path = require('path');
const os = require('os');
const { spawnSync } = require('child_process');

const GITHUB_REPO = 'redstone-md/nodule';
const BIN_DIR = __dirname;

function getTarget() {
  var platform = os.platform();
  var arch = os.arch();

  var goos, goarch, ext = '';

  switch (platform) {
    case 'linux':   goos = 'linux';   break;
    case 'darwin':  goos = 'darwin';  break;
    case 'win32':   goos = 'windows'; ext = '.exe'; break;
    default: return null;
  }

  switch (arch) {
    case 'x64':     goarch = 'amd64'; break;
    case 'arm64':   goarch = 'arm64'; break;
    case 'ia32':    goarch = '386';   break;
    default: return null;
  }

  return {
    goos: goos,
    goarch: goarch,
    ext: ext,
    assetName: 'nodule-' + goos + '-' + goarch + ext
  };
}

function getDestPath(target) {
  return path.join(BIN_DIR, 'nodule-' + target.goos + '-' + target.goarch + target.ext);
}

/**
 * Synchronous download — used by nodule.js at first run.
 * Returns true if binary was installed successfully.
 */
function installSync() {
  var target = getTarget();
  if (!target) return false;

  var destPath = getDestPath(target);
  if (fs.existsSync(destPath)) return true;

  process.stderr.write('nodule: downloading binary for ' + target.goos + '/' + target.goarch + '...\n');

  // Fetch latest release URL synchronously via curl or Invoke-WebRequest
  var curlResult = spawnSync('curl', [
    '-s', '-L', '-H', 'User-Agent: nodule-installer',
    'https://api.github.com/repos/' + GITHUB_REPO + '/releases/latest'
  ], { encoding: 'utf8', timeout: 30000 });

  if (curlResult.status !== 0 || !curlResult.stdout) {
    tryGoInstall();
    return fs.existsSync(destPath);
  }

  var release;
  try { release = JSON.parse(curlResult.stdout); }
  catch (e) {
    tryGoInstall();
    return fs.existsSync(destPath);
  }

  var asset = (release.assets || []).find(function(a) { return a.name === target.assetName; });
  if (!asset) {
    process.stderr.write('nodule: asset ' + target.assetName + ' not found in ' + release.tag_name + '\n');
    tryGoInstall();
    return fs.existsSync(destPath);
  }

  // Download binary via curl
  var dlResult = spawnSync('curl', [
    '-s', '-L', '-o', destPath,
    '-H', 'User-Agent: nodule-installer',
    '-H', 'Accept: application/octet-stream',
    asset.browser_download_url
  ], { timeout: 60000 });

  if (dlResult.status !== 0) {
    process.stderr.write('nodule: download failed\n');
    try { fs.unlinkSync(destPath); } catch (e) {}
    tryGoInstall();
    return fs.existsSync(destPath);
  }

  if (target.goos !== 'windows') {
    try { fs.chmodSync(destPath, 0o755); } catch (e) {}
  }

  process.stderr.write('nodule: installed ' + release.tag_name + '\n');
  return true;
}

function tryGoInstall() {
  process.stderr.write('nodule: falling back to go install...\n');
  var result = spawnSync('go', [
    'install', 'github.com/' + GITHUB_REPO + '/cmd/nodule@latest'
  ], { stdio: 'inherit', timeout: 120000 });

  if (result.status !== 0) {
    process.stderr.write('nodule: go install failed. Install manually:\n');
    process.stderr.write('  go install github.com/redstone-md/nodule/cmd/nodule@latest\n');
  } else {
    process.stderr.write('nodule: installed via go install\n');
  }
}

// --- Async mode (postinstall) ---

function fetchLatestReleaseTag() {
  return new Promise(function(resolve, reject) {
    var url = 'https://api.github.com/repos/' + GITHUB_REPO + '/releases/latest';
    https.get(url, { headers: { 'User-Agent': 'nodule-installer' } }, function(res) {
      if (res.statusCode !== 200) {
        return reject(new Error('GitHub API returned ' + res.statusCode));
      }
      var body = '';
      res.on('data', function(chunk) { body += chunk; });
      res.on('end', function() {
        try { resolve(JSON.parse(body)); }
        catch (e) { reject(e); }
      });
    }).on('error', reject);
  });
}

function downloadAsset(url, destPath) {
  return new Promise(function(resolve, reject) {
    var file = fs.createWriteStream(destPath);
    https.get(url, { headers: { 'User-Agent': 'nodule-installer' } }, function(res) {
      if (res.statusCode === 301 || res.statusCode === 302) {
        file.close();
        fs.unlinkSync(destPath);
        return downloadAsset(res.headers.location, destPath).then(resolve, reject);
      }
      if (res.statusCode !== 200) {
        file.close();
        fs.unlinkSync(destPath);
        return reject(new Error('Download failed: HTTP ' + res.statusCode));
      }
      res.pipe(file);
      file.on('finish', function() { file.close(resolve); });
    }).on('error', function(err) {
      file.close();
      try { fs.unlinkSync(destPath); } catch (e) {}
      reject(err);
    });
  });
}

async function main() {
  var target = getTarget();
  if (!target) {
    process.stderr.write('nodule: unsupported platform, skipping download\n');
    return;
  }

  var destPath = getDestPath(target);
  if (fs.existsSync(destPath)) {
    console.log('nodule: binary already present, skipping download');
    return;
  }

  console.log('nodule: downloading binary for ' + target.goos + '/' + target.goarch + '...');

  try {
    var release = await fetchLatestReleaseTag();
    var asset = (release.assets || []).find(function(a) { return a.name === target.assetName; });
    if (!asset) {
      throw new Error('asset ' + target.assetName + ' not found in release ' + release.tag_name);
    }

    await downloadAsset(asset.browser_download_url, destPath);

    if (target.goos !== 'windows') {
      fs.chmodSync(destPath, 0o755);
    }

    console.log('nodule: installed ' + release.tag_name);
  } catch (err) {
    console.warn('nodule: could not download binary (' + err.message + ')');
    tryGoInstall();
  }
}

// Export sync installer for nodule.js, run async on postinstall
module.exports = { installSync: installSync };

// When run directly (postinstall), execute async download
if (require.main === module) {
  main();
}
