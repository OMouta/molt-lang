import fs from "node:fs";
import path from "node:path";

const root = path.resolve(import.meta.dirname, "..");
const manifest = JSON.parse(fs.readFileSync(path.join(root, "package.json"), "utf8"));

const requiredFiles = [
  "README.md",
  "CHANGELOG.md",
  "LICENSE",
  "language-configuration.json",
  "syntaxes/molt.tmLanguage.json",
  "icons/molt.png"
];

for (const relative of requiredFiles) {
  if (!fs.existsSync(path.join(root, relative))) {
    throw new Error(`Missing required extension file: ${relative}`);
  }
}

if (!manifest.engines || typeof manifest.engines.vscode !== "string") {
  throw new Error("package.json must declare engines.vscode");
}

if (!manifest.contributes?.languages?.length) {
  throw new Error("package.json must contribute at least one language");
}

if (!manifest.contributes?.grammars?.length) {
  throw new Error("package.json must contribute at least one grammar");
}

console.log("manifest ok");
