"use strict";

const fs = require("fs");
const path = require("path");

function resolveBinary(name) {
  const platform = platformName();
  const arch = archName();
  const ext = platform === "windows" ? ".exe" : "";
  const binPath = path.join(__dirname, "..", "bin", "native", name + ext);

  if (!fs.existsSync(binPath)) {
    const hint = `binary not found (${platform}/${arch}); re-run npm install`;
    throw new Error(hint);
  }
  return binPath;
}

function platformName() {
  switch (process.platform) {
    case "win32":
      return "windows";
    case "darwin":
      return "darwin";
    case "linux":
      return "linux";
    default:
      throw new Error(`unsupported platform: ${process.platform}`);
  }
}

function archName() {
  switch (process.arch) {
    case "x64":
      return "amd64";
    case "arm64":
      return "arm64";
    default:
      throw new Error(`unsupported architecture: ${process.arch}`);
  }
}

module.exports = {
  resolveBinary,
  platformName,
  archName
};
