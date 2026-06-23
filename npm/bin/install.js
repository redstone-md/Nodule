#!/usr/bin/env node
/**
 * Nodule — postinstall script.
 *
 * Downloads the platform-appropriate Nodule binary from the latest GitHub
 * Release. Falls back to `go install` if download fails.
 */

const https = require('https');
const fs = require('fs');
const path = require('path');
const os = require('os');
const { spawnSync } = require('child_process');

const GITHUB_REPO = 'redstone-md/nodule';
const BIN_DIR = __dirname;

function getTarget() {
  const platform = os.platform();
  const arch = os.arch();

  let goos, goarch, ext = '';

  switch (platform) {
    case 'linux':   goos = 'linux';   break;
    case 'darwin':  goos = 'darwin';  break;
    case 'win32':   goos = 'windows'; ext = '.exe'; break;
    default:
      console.warn('nodule: unsupported platform ' + platform + ', skipping download');
      console.warn('nodule: install manually: go install github.com/redstone-md/nodule/cmd/nodule@latest');
      return null;
  }

  switch (arch) {
    case 'x64':     goarch = 'amd64'; break;
    case 'arm64':   goarch = 'arm64'; break;
    case 'ia32':    goarch = '386';   break;
    default:
      console.warn('nodule: unsupported arch ' + arch + ', skipping download');
      return null;
  }

  const assetName = 'nodule-' + goos + '-' + goarch + ext;
  return { goos: goos, goarch: goarch, ext: ext, assetName: assetName };
}

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
  if (!target) return;

  var destPath = path.join(BIN_DIR, 'nodule-' + target.goos + '-' + target.goarch + target.ext);

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
    console.warn('nodule: falling back to go install...');

    var result = spawnSync('go', [
      'install', 'github.com/' + GITHUB_REPO + '/cmd/nodule@latest'
    ], { stdio: 'inherit' });

    if (result.status !== 0) {
      console.warn('nodule: go install failed. Install manually:');
      console.warn('         go install github.com/redstone-md/nodule/cmd/nodule@latest');
    } else {
      console.log('nodule: installed via go install (ensure $GOPATH/bin is in PATH)');
    }
  }
}

main();
