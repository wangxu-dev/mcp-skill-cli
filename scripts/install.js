"use strict";

const fs = require("fs");
const https = require("https");
const path = require("path");
const { platformName, archName } = require("./resolve");

const pkg = require("../package.json");

const repo = process.env.MCP_SKILL_RELEASE_REPO || "wangxu-dev/mcp-skill-cli";
const version = pkg.version;

const platform = platformName();
const arch = archName();
const ext = platform === "windows" ? ".exe" : "";
const binDir = path.join(__dirname, "..", "bin", "native");

const binaries = ["mcp", "skill"];

async function main() {
  if (process.env.MCP_SKIP_DOWNLOAD === "1") {
    return;
  }

  fs.mkdirSync(binDir, { recursive: true });
  for (const name of binaries) {
    const dest = path.join(binDir, name + ext);
    if (fs.existsSync(dest)) {
      continue;
    }
    await downloadBinary(name, dest);
    if (platform !== "windows") {
      fs.chmodSync(dest, 0o755);
    }
  }
}

async function downloadBinary(name, dest) {
  const base = `${name}_${version}_${platform}_${arch}`;
  const candidates = platform === "windows"
    ? [`${base}.exe`, `${base}.exe.exe`]
    : [base];

  let lastError = null;
  for (const assetName of candidates) {
    const url = `https://github.com/${repo}/releases/download/v${version}/${assetName}`;
    try {
      await download(url, dest);
      return;
    } catch (err) {
      lastError = err;
    }
  }
  throw lastError || new Error(`download failed for ${name}`);
}

function download(url, dest) {
  return new Promise((resolve, reject) => {
    const request = https.get(
      url,
      { headers: { "User-Agent": "mcp-skill-cli" } },
      (res) => {
        if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
          res.resume();
          download(res.headers.location, dest).then(resolve).catch(reject);
          return;
        }
        if (res.statusCode !== 200) {
          res.resume();
          reject(new Error(`download failed (${res.statusCode}): ${url}`));
          return;
        }
        const file = fs.createWriteStream(dest, { mode: 0o755 });
        res.pipe(file);
        file.on("finish", () => file.close(resolve));
        file.on("error", reject);
      }
    );
    request.on("error", reject);
  });
}

main().catch((err) => {
  console.error(err.message || String(err));
  process.exit(1);
});
