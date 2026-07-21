import { existsSync, readFileSync, readdirSync } from "node:fs";
import { dirname, relative, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const scriptDir = dirname(fileURLToPath(import.meta.url));
const frontendRoot = resolve(scriptDir, "..");
const repoRoot = resolve(frontendRoot, "..");
const sourceRoot = resolve(frontendRoot, "src");

const errors = [];

const removedDirectories = [
  "react",
  "features",
  "views",
  "layouts",
  "plugins",
  "connect",
  "store",
];
for (const directory of removedDirectories) {
  if (existsSync(resolve(sourceRoot, directory))) {
    errors.push(`src/${directory}/ must not be recreated`);
  }
}

const requiredDirectories = [
  "app",
  "app/router",
  "api",
  "components/ui",
  "modules",
  "routes/auth",
  "routes/project",
  "routes/workspace",
  "stores/app",
];
for (const directory of requiredDirectories) {
  if (!existsSync(resolve(sourceRoot, directory))) {
    errors.push(`required owner directory is missing: src/${directory}`);
  }
}

const listSourceFiles = (directory) =>
  readdirSync(directory, { withFileTypes: true }).flatMap((entry) => {
    const path = resolve(directory, entry.name);
    if (entry.isDirectory()) return listSourceFiles(path);
    return /\.(?:ts|tsx)$/.test(entry.name) ? [path] : [];
  });

const retiredAliases = [
  ["@", "react"].join("/"),
  ["@", "views"].join("/"),
  ["@", "layouts"].join("/"),
  ["@", "connect"].join("/"),
  ["@", "plugins"].join("/"),
];

const sourceFiles = listSourceFiles(sourceRoot);
for (const file of sourceFiles) {
  const source = readFileSync(file, "utf8");
  const path = relative(frontendRoot, file);
  for (const alias of retiredAliases) {
    if (source.includes(alias)) errors.push(`${path} uses retired alias ${alias}`);
  }
  if (/from\s+["']@\/store(?:[\/"'])/.test(source)) {
    errors.push(`${path} uses retired alias @/store`);
  }
}

const sharedRoots = ["api", "components", "hooks", "lib", "modules", "stores"];
for (const root of sharedRoots) {
  for (const file of listSourceFiles(resolve(sourceRoot, root))) {
    const source = readFileSync(file, "utf8");
    if (source.includes("@/routes/")) {
      errors.push(`${relative(frontendRoot, file)} imports route-owned code`);
    }
  }
}

const agentFiles = [
  resolve(repoRoot, "AGENTS.md"),
  resolve(repoRoot, "CLAUDE.md"),
  resolve(repoRoot, "docs/superpowers/AGENTS.md"),
  resolve(frontendRoot, "AGENTS.md"),
  resolve(frontendRoot, "CLAUDE.md"),
  resolve(frontendRoot, "components.json"),
  resolve(frontendRoot, "package.json"),
  resolve(frontendRoot, "vite.config.ts"),
  resolve(frontendRoot, "explain-visualizer.html"),
  resolve(frontendRoot, ".claude/BUTTON_SPACING_STANDARDIZATION.md"),
  resolve(frontendRoot, "tests/e2e/AGENTS.md"),
  resolve(frontendRoot, "src/modules/agent/AGENTS.md"),
  resolve(frontendRoot, "src/modules/sql-editor/AGENTS.md"),
  resolve(repoRoot, ".sonarcloud.properties"),
];
const retiredDocumentationTokens = [
  ["src", "react"].join("/"),
  ["src", "features"].join("/"),
  ["src", "views"].join("/"),
  ["src", "layouts"].join("/"),
  ["src", "plugins"].join("/"),
  ["src", "connect"].join("/"),
  ["src", "store", ""].join("/"),
  ["@", "react"].join("/"),
  "VueShellBridgeEvent",
  "bb.vue-notification",
];
for (const file of agentFiles) {
  if (!existsSync(file)) {
    errors.push(`agent-facing file is missing: ${relative(repoRoot, file)}`);
    continue;
  }
  const source = readFileSync(file, "utf8");
  for (const token of retiredDocumentationTokens) {
    if (source.includes(token)) {
      errors.push(`${relative(repoRoot, file)} contains retired reference ${token}`);
    }
  }
}

for (const file of [resolve(repoRoot, "CLAUDE.md"), resolve(frontendRoot, "CLAUDE.md")]) {
  if (readFileSync(file, "utf8").trim() !== "@AGENTS.md") {
    errors.push(`${relative(repoRoot, file)} must delegate only to AGENTS.md`);
  }
}

if (errors.length > 0) {
  console.error("Frontend structure check failed:\n");
  for (const error of errors) console.error(`- ${error}`);
  process.exit(1);
}

console.log("Frontend structure check passed");
