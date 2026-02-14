#!/usr/bin/env node
/**
 * Sets the app name in Electron's Info.plist for macOS dev mode.
 * macOS shows the binary name ("Electron") in the dock when running from CLI.
 * This script patches Info.plist to show "Flowgentic" instead.
 */

const fs = require("fs");
const path = require("path");

if (process.platform !== "darwin") {
  process.exit(0);
}

const plistPath = path.join(
  __dirname,
  "..",
  "node_modules",
  "electron",
  "dist",
  "Electron.app",
  "Contents",
  "Info.plist"
);

if (!fs.existsSync(plistPath)) {
  console.warn("Electron Info.plist not found, skipping app name patch");
  process.exit(0);
}

try {
  let content = fs.readFileSync(plistPath, "utf8");
  
  // Replace CFBundleName
  content = content.replace(
    /<key>CFBundleName<\/key>\s*<string>[^<]*<\/string>/,
    "<key>CFBundleName</key>\n\t<string>Flowgentic</string>"
  );
  
  // Also replace CFBundleDisplayName if it exists
  content = content.replace(
    /<key>CFBundleDisplayName<\/key>\s*<string>[^<]*<\/string>/,
    "<key>CFBundleDisplayName</key>\n\t<string>Flowgentic</string>"
  );
  
  fs.writeFileSync(plistPath, content);
  console.log("✓ Patched Electron app name to 'Flowgentic'");
} catch (err) {
  console.warn("Failed to patch Electron app name:", err.message);
  process.exit(0);
}
