// postinstall: fetch the prebuilt Nodule binary for this platform from the
// matching GitHub release and place it in vendor/.
//
// Mirrors the continuum-mcp install pattern: vendor directory next to the
// launcher, with the binary named after the platform triple.

"use strict";

const fs = require("fs");
const path = require("path");
const pkg = require("../package.json");

// Node platform/arch -> Go GOOS/GOARCH used in the release asset names.
const TARGETS = {
  "linux-x64":   { goos: "linux",   goarch: "amd64" },
  "linux-arm64": { goos: "linux",   goarch: "arm64" },
  "darwin-x64":  { goos: "darwin",  goarch: "amd64" },
  "darwin-arm64":{ goos: "darwin",  goarch: "arm64" },
  "win32-x64":   { goos: "windows", goarch: "amd64" },
  "win32-arm64": { goos: "windows", goarch: "arm64" },
};

const REPO = "redstone-md/nodule";

async function main() {
  const key = process.platform + "-" + process.arch;
  const target = TARGETS[key];
  if (!target) {
    process.stderr.write(
      "nodule: no prebuilt binary for " + key + "; build from source (go install github.com/redstone-md/nodule/cmd/nodule@latest)\n"
    );
    return;
  }

  const ext = process.platform === "win32" ? ".exe" : "";
  const base = "https://github.com/" + REPO + "/releases/download/v" + pkg.version;
  const vendor = path.join(__dirname, "..", "vendor");
  fs.mkdirSync(vendor, { recursive: true });

  const name = "nodule-" + target.goos + "-" + target.goarch + ext;
  const dest = path.join(vendor, name);
  const url = base + "/" + name;

  process.stderr.write("nodule: downloading " + name + "\n");

  const res = await fetch(url, { redirect: "follow" });
  if (!res.ok) {
    throw new Error("download failed (" + res.status + ") for " + name);
  }
  fs.writeFileSync(dest, Buffer.from(await res.arrayBuffer()));
  if (process.platform !== "win32") {
    fs.chmodSync(dest, 0o755);
  }
  process.stderr.write("nodule: ready\n");
}

main().catch((err) => {
  process.stderr.write("nodule: install failed: " + err.message + "\n");
  process.exit(1);
});
