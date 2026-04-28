// frontend/scripts/find-dead-vue.mjs
//
// Detects unreferenced Vue single-file components under src/components/ as part
// of the Vue → React migration sweep. A component counts as live if it is
// reachable from a fixed root set via:
//   - explicit `from "..."` / `import("...")` / `import.meta.glob("...")` paths,
//     including bare relative imports that Vite resolves to .vue;
//   - PascalCase or kebab-case tag references in any live .vue template, mapped
//     through the unplugin-vue-components registry at frontend/components.d.ts;
//   - Vue Router lazy imports under src/router/.
//
// The script is intentionally a static scanner. It does not evaluate dynamic
// `:is="ComponentName"` strings, computed component names, or imports built
// from variables. Passing this scan does not prove a component is unused — the
// real gate is `pnpm --dir frontend build`, which fails on unresolved
// auto-imports. Use this scan to find candidates, then verify with build.
//
// Usage:
//   node scripts/find-dead-vue.mjs               # report only
//   node scripts/find-dead-vue.mjs --delete      # delete dead .vue files
//   node scripts/find-dead-vue.mjs --react-only  # list .vue whose importers
//                                                # are all under src/react/

import { existsSync, readFileSync, readdirSync, statSync, unlinkSync } from "node:fs";
import { dirname, relative, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = dirname(fileURLToPath(import.meta.url));
const FRONTEND_ROOT = resolve(__dirname, "..");
const SRC = resolve(FRONTEND_ROOT, "src");
const COMPONENTS_DIR = resolve(SRC, "components");
const REGISTRY_FILE = resolve(FRONTEND_ROOT, "components.d.ts");

const args = new Set(process.argv.slice(2));
const MODE_DELETE = args.has("--delete");
const MODE_REACT_ONLY = args.has("--react-only");

// Files that are always live entry points (auto-roots).
const ROOT_FILES = [
  "src/main.ts",
  "src/explain-visualizer-main.ts",
  "src/init.ts",
  "src/App.vue",
  "src/AuthContext.vue",
  "src/NotificationContext.vue",
  "src/ExplainVisualizerApp.vue",
  "src/customAppProfile.ts",
  "src/DummyRootView.ts",
  "src/shell-bridge.test.ts",
];
// Whole subtrees that are always live: nothing inside is a candidate, and
// anything they import is reachable.
const ROOT_DIRS = [
  "src/layouts",
  "src/views",
  "src/router",
  "src/store",
  "src/plugins",
  "src/composables",
  "src/types",
  "src/utils",
  "src/connect",
  "src/locales",
  "src/bbkit",
  "src/react",
];

// File extensions to scan for imports.
const SOURCE_EXT = /\.(?:vue|ts|tsx|mts|cts|mjs|cjs|js)$/;

const listFiles = (dir, out = []) => {
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const full = resolve(dir, entry.name);
    if (entry.isDirectory()) {
      // Skip generated noise.
      if (entry.name === "node_modules" || entry.name === "dist") continue;
      listFiles(full, out);
    } else if (entry.isFile() && SOURCE_EXT.test(entry.name)) {
      out.push(full);
    }
  }
  return out;
};

const toRel = (abs) => relative(FRONTEND_ROOT, abs).split("\\").join("/");

const isUnderAny = (abs, dirs) => {
  for (const d of dirs) {
    const full = resolve(FRONTEND_ROOT, d);
    if (abs === full || abs.startsWith(full + "/")) return true;
  }
  return false;
};

// Resolve a Vite-style import path (with @ alias, relative path, optional
// extension) to an absolute file path on disk, if it points to something we
// recognize. Returns null if it does not resolve to a real file.
const resolveImport = (specifier, fromFile) => {
  if (!specifier) return null;
  // Skip vendor packages and virtual modules.
  if (specifier.startsWith("~") || specifier.includes("?")) return null;
  if (
    !specifier.startsWith(".") &&
    !specifier.startsWith("@/") &&
    !specifier.startsWith("/")
  ) {
    return null;
  }

  let basePath;
  if (specifier.startsWith("@/")) {
    basePath = resolve(SRC, specifier.slice(2));
  } else if (specifier.startsWith("/")) {
    basePath = resolve(FRONTEND_ROOT, specifier.slice(1));
  } else {
    basePath = resolve(dirname(fromFile), specifier);
  }

  // Direct hit (with extension).
  if (existsSync(basePath) && statSync(basePath).isFile()) {
    return basePath;
  }
  // Try common extensions.
  for (const ext of [".vue", ".ts", ".tsx", ".js", ".mjs"]) {
    if (existsSync(basePath + ext)) {
      return basePath + ext;
    }
  }
  // Try directory index.
  for (const ext of [".vue", ".ts", ".tsx", ".js", ".mjs"]) {
    const indexPath = basePath + "/index" + ext;
    if (existsSync(indexPath)) {
      return indexPath;
    }
  }
  return null;
};

// Regex for `from "path"`, `import("path")`, and `import.meta.glob` literals.
const STATIC_FROM = /\bfrom\s+["']([^"']+)["']/g;
const DYNAMIC_IMPORT = /\bimport\s*\(\s*["']([^"']+)["']\s*\)/g;
const GLOB_LITERAL = /import\.meta\.glob\s*\(\s*(\[[^\]]*\]|["'][^"']+["'])/g;
// re-export: `export * from "x"`, `export { ... } from "x"`.
const EXPORT_FROM = /\bexport\s+(?:\*|\{[^}]*\})\s+from\s+["']([^"']+)["']/g;

// Pull all string literal imports out of a file and resolve to abs paths.
const collectImportsFromCode = (source, fromFile) => {
  const refs = new Set();
  const push = (specifier) => {
    const abs = resolveImport(specifier, fromFile);
    if (abs) refs.add(abs);
  };

  for (const m of source.matchAll(STATIC_FROM)) push(m[1]);
  for (const m of source.matchAll(DYNAMIC_IMPORT)) push(m[1]);
  for (const m of source.matchAll(EXPORT_FROM)) push(m[1]);
  for (const m of source.matchAll(GLOB_LITERAL)) {
    const inside = m[1];
    // Pull every quoted literal out of either a single string or an array.
    for (const lit of inside.matchAll(/["']([^"']+)["']/g)) {
      // import.meta.glob accepts a glob — strip the wildcard suffix and
      // walk the directory if it points at .vue files.
      const pat = lit[1];
      if (!pat.includes("*")) {
        push(pat);
        continue;
      }
      // Globs that include .vue: enumerate matching files under the base dir.
      if (!pat.endsWith(".vue") && !pat.includes("*.vue")) continue;
      const baseSpec = pat.replace(/\/\*\*\/.*$/, "").replace(/\/\*[^/]*$/, "");
      let baseAbs;
      if (baseSpec.startsWith("@/")) baseAbs = resolve(SRC, baseSpec.slice(2));
      else if (baseSpec.startsWith(".")) baseAbs = resolve(dirname(fromFile), baseSpec);
      else continue;
      if (!existsSync(baseAbs) || !statSync(baseAbs).isDirectory()) continue;
      for (const f of listFiles(baseAbs)) {
        if (f.endsWith(".vue")) refs.add(f);
      }
    }
  }
  return refs;
};

// Parse PascalCase / kebab-case tag references in a .vue (or .tsx) source.
// Returns a Set of normalized PascalCase names.
const TAG_PASCAL = /<([A-Z][A-Za-z0-9]*)\b/g;
const TAG_KEBAB = /<([a-z][a-z0-9]*(?:-[a-z0-9]+)+)\b/g;
const kebabToPascal = (s) =>
  s.replace(/(^|-)([a-z0-9])/g, (_, _dash, ch) => ch.toUpperCase());
const collectTagNames = (source) => {
  const names = new Set();
  for (const m of source.matchAll(TAG_PASCAL)) names.add(m[1]);
  for (const m of source.matchAll(TAG_KEBAB)) names.add(kebabToPascal(m[1]));
  return names;
};

// Parse components.d.ts into a Map<PascalName, absPath>.
const loadAutoImportRegistry = () => {
  if (!existsSync(REGISTRY_FILE)) {
    console.error(
      `error: ${toRel(REGISTRY_FILE)} not found. Run 'pnpm --dir frontend dev' or ` +
        "'pnpm --dir frontend build' once to let unplugin-vue-components generate it."
    );
    process.exit(2);
  }
  const source = readFileSync(REGISTRY_FILE, "utf-8");
  const map = new Map();
  // Match e.g.  AccessGrantView: typeof import('./src/components/...vue')
  const re = /^\s*['"]?([A-Za-z][A-Za-z0-9:]*)['"]?:\s*typeof import\(['"]([^'"]+\.vue)['"]\)/gm;
  for (const m of source.matchAll(re)) {
    const [, name, relPath] = m;
    if (name.includes(":")) continue; // Icon resolver entries.
    const abs = resolve(FRONTEND_ROOT, relPath.replace(/^\.\//, ""));
    if (!map.has(name)) map.set(name, abs);
  }
  return map;
};

const loadFile = (abs) => {
  try {
    return readFileSync(abs, "utf-8");
  } catch {
    return "";
  }
};

// Build the directed reference graph and BFS from roots.
const buildGraph = () => {
  const allFiles = listFiles(SRC);
  const registry = loadAutoImportRegistry();

  // file → Set<file>
  const edges = new Map();
  for (const file of allFiles) {
    const source = loadFile(file);
    const refs = collectImportsFromCode(source, file);
    if (file.endsWith(".vue")) {
      // Auto-import: resolve PascalCase tag names through the registry.
      for (const name of collectTagNames(source)) {
        const target = registry.get(name);
        if (target) refs.add(target);
      }
    }
    edges.set(file, refs);
  }
  return { allFiles, edges, registry };
};

const seedRoots = (allFiles) => {
  const roots = new Set();
  for (const rel of ROOT_FILES) {
    const abs = resolve(FRONTEND_ROOT, rel);
    if (existsSync(abs)) roots.add(abs);
  }
  for (const file of allFiles) {
    if (isUnderAny(file, ROOT_DIRS)) roots.add(file);
    // Tests count as live importers.
    if (/\.test\.(?:ts|tsx|mts|cjs|js|mjs)$/.test(file)) roots.add(file);
  }
  return roots;
};

const traverse = (edges, roots) => {
  const live = new Set();
  const queue = [...roots];
  while (queue.length) {
    const file = queue.pop();
    if (live.has(file)) continue;
    live.add(file);
    const refs = edges.get(file);
    if (!refs) continue;
    for (const ref of refs) {
      if (!live.has(ref)) queue.push(ref);
    }
  }
  return live;
};

// Reverse-edge map: for each .vue, who imports it.
const buildReverseEdges = (edges) => {
  const rev = new Map();
  for (const [src, targets] of edges) {
    for (const t of targets) {
      if (!rev.has(t)) rev.set(t, new Set());
      rev.get(t).add(src);
    }
  }
  return rev;
};

const candidateVueFiles = (allFiles) =>
  allFiles.filter(
    (f) => f.endsWith(".vue") && f.startsWith(COMPONENTS_DIR + "/")
  );

const groupByDirectory = (files) => {
  const groups = new Map();
  for (const f of files) {
    const rel = toRel(f);
    const dir = rel.split("/").slice(0, -1).join("/");
    if (!groups.has(dir)) groups.set(dir, []);
    groups.get(dir).push(rel);
  }
  return [...groups.entries()].sort((a, b) => a[0].localeCompare(b[0]));
};

const printGroups = (label, files) => {
  if (files.length === 0) {
    console.log(`${label}: none`);
    return;
  }
  console.log(`${label} (${files.length}):`);
  for (const [dir, list] of groupByDirectory(files)) {
    console.log(`  ${dir}/  (${list.length})`);
    for (const f of list.sort()) {
      console.log(`    ${f}`);
    }
  }
};

const reportDead = (deadFiles) => {
  printGroups("Dead .vue files (no live importer)", deadFiles);
  console.log(`\nTotal dead: ${deadFiles.length}`);
};

const deleteDead = (deadFiles) => {
  for (const f of deadFiles) {
    unlinkSync(f);
    console.log(`deleted ${toRel(f)}`);
  }
  console.log(`\nDeleted ${deadFiles.length} files. Re-run without --delete to find newly-orphaned components.`);
};

const reactOnlyReport = (allVue, edges, reverseEdges) => {
  const reactRoot = resolve(SRC, "react") + "/";
  const candidates = [];
  for (const f of allVue) {
    const importers = reverseEdges.get(f);
    if (!importers || importers.size === 0) continue;
    let allReact = true;
    for (const i of importers) {
      if (!i.startsWith(reactRoot)) {
        allReact = false;
        break;
      }
    }
    if (allReact) candidates.push(f);
  }
  printGroups("Vue components imported only from src/react/", candidates);
};

const main = () => {
  const { allFiles, edges } = buildGraph();
  const roots = seedRoots(allFiles);
  const live = traverse(edges, roots);
  const candidates = candidateVueFiles(allFiles);

  if (MODE_REACT_ONLY) {
    const reverseEdges = buildReverseEdges(edges);
    reactOnlyReport(candidates, edges, reverseEdges);
    return;
  }

  const dead = candidates.filter((f) => !live.has(f));
  if (MODE_DELETE) {
    if (dead.length === 0) {
      console.log("No dead components to delete.");
      return;
    }
    deleteDead(dead);
    return;
  }

  reportDead(dead);
};

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  main();
}
