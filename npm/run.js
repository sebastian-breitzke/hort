#!/usr/bin/env node
"use strict";

const { execFileSync } = require("child_process");
const path = require("path");

const binaryName = process.platform === "win32" ? "hort.exe" : "hort";
const binaryPath = path.join(__dirname, "bin", binaryName);

try {
  execFileSync(binaryPath, process.argv.slice(2), { stdio: "inherit" });
} catch (err) {
  process.exit(err.status || 1);
}
