#!/usr/bin/env node

const path = require("path");
const { execFileSync } = require("child_process");
const os = require("os");

const binName = os.platform() === "win32" ? "neohtop-cli.exe" : "neohtop-cli";
const binPath = path.join(__dirname, binName);

try {
  execFileSync(binPath, process.argv.slice(2), { stdio: "inherit" });
} catch (err) {
  if (err.status !== null) {
    process.exit(err.status);
  }
  console.error(`neohtop-cli: failed to run binary at ${binPath}`);
  console.error("Run 'npm install -g neohtop-cli' to reinstall.");
  process.exit(1);
}
