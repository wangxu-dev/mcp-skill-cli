"use strict";

const fs = require("fs");
const path = require("path");

const versionArg = process.argv[2];
if (!versionArg) {
  console.error("usage: node scripts/release-notes.js <version>");
  process.exit(1);
}

const version = versionArg.replace(/^v/, "");
const changelogPath = path.join(__dirname, "..", "CHANGELOG.md");
const content = fs.readFileSync(changelogPath, "utf8");
const lines = content.split(/\r?\n/);

let inSection = false;
const output = [];
for (const line of lines) {
  if (line.startsWith("## ")) {
    if (inSection) {
      break;
    }
    inSection = line.includes(`## ${version} `) || line.trim() === `## ${version}`;
    if (inSection) {
      output.push(line);
    }
    continue;
  }
  if (inSection) {
    output.push(line);
  }
}

if (output.length === 0) {
  console.error(`version ${version} not found in CHANGELOG.md`);
  process.exit(1);
}

process.stdout.write(output.join("\n").trim() + "\n");
