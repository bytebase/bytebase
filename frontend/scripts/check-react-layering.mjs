// frontend/scripts/check-react-layering.mjs
//
// Enforces the React semantic overlay layering policy introduced by PR #20012.
// Feature code must not create global z-index overlays directly. Use shared UI
// primitives or portal into the semantic layer roots instead.

import { readFileSync, readdirSync } from "fs";
import { relative, resolve } from "path";
import { fileURLToPath } from "url";
import { dirname } from "path";

const __dirname = dirname(fileURLToPath(import.meta.url));
const ROOT = resolve(__dirname, "..");
const REACT_DIR = resolve(ROOT, "src/react");
const REPORT_ONLY = process.argv.includes("--report-only");

const APPROVED_PREFIXES = [
  "src/react/components/ui/",
  "src/react/plugins/agent/",
];

const APPROVED_FILES = new Set([
  "src/react/components/auth/SessionExpiredSurface.tsx",
]);

const LOCAL_PAINT_ORDER_EXCEPTIONS = new Map([
  [
    "src/react/components/monaco/MonacoEditor.tsx",
    "Monaco action buttons are local to the editor surface.",
  ],
  [
    "src/react/plugins/agent/components/AgentInput.tsx",
    "Approved agent-owned layer family.",
  ],
  [
    "src/react/plugins/agent/components/AgentWindow.tsx",
    "Approved agent-owned layer family.",
  ],
]);

const findFiles = (dir) => {
  const files = [];
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const full = resolve(dir, entry.name);
    if (entry.isDirectory()) {
      files.push(...findFiles(full));
    } else if (/\.(ts|tsx)$/.test(entry.name)) {
      files.push(full);
    }
  }
  return files;
};

const isApprovedPath = (path) =>
  APPROVED_FILES.has(path) ||
  APPROVED_PREFIXES.some((prefix) => path.startsWith(prefix));

const hasRawZClass = (line) =>
  /\bz-\d+\b/.test(line) || /\bz-\[[^\]]+\]/.test(line);

const hasGlobalFixedZ = (line) => /\bfixed\b/.test(line) && hasRawZClass(line);

const hasHighAbsoluteZ = (line) =>
  /\babsolute\b/.test(line) && /\b(?:z-4\d|z-5\d|z-\[[^\]]+\])\b/.test(line);

const hasInlineZIndex = (line) => /\bzIndex\s*:/.test(line);

const scanFile = (file) => {
  const rel = relative(ROOT, file);
  if (isApprovedPath(rel)) {
    return [];
  }
  if (LOCAL_PAINT_ORDER_EXCEPTIONS.has(rel)) {
    return [];
  }

  const source = readFileSync(file, "utf-8");
  const violations = [];
  const lines = source.split("\n");

  lines.forEach((line, index) => {
    const lineNumber = index + 1;
    if (hasGlobalFixedZ(line)) {
      violations.push({
        rel,
        lineNumber,
        reason: "feature-owned fixed overlay uses raw z-index",
        line,
      });
    }
    if (hasHighAbsoluteZ(line)) {
      violations.push({
        rel,
        lineNumber,
        reason: "feature-owned absolute overlay uses high raw z-index",
        line,
      });
    }
    if (hasInlineZIndex(line)) {
      violations.push({
        rel,
        lineNumber,
        reason: "inline zIndex bypasses semantic layer ownership",
        line,
      });
    }
  });

  const portalToBody = /createPortal\([\s\S]*?document\.body/g;
  let match;
  while ((match = portalToBody.exec(source))) {
    const lineNumber = source.slice(0, match.index).split("\n").length;
    violations.push({
      rel,
      lineNumber,
      reason: "feature-owned portal targets document.body directly",
      line: lines[lineNumber - 1] ?? "",
    });
  }

  return violations;
};

const violations = findFiles(REACT_DIR).flatMap(scanFile);

if (violations.length > 0) {
  console.error(
    `React layering policy violations (${violations.length}). ` +
      "Use shared overlay primitives or getLayerRoot(\"overlay\").\n"
  );
  for (const violation of violations) {
    console.error(
      `${violation.rel}:${violation.lineNumber}: ${violation.reason}\n` +
        `  ${violation.line.trim()}\n`
    );
  }
  if (!REPORT_ONLY) {
    process.exit(1);
  }
}

console.log(
  violations.length === 0
    ? "React layering policy: all checks passed."
    : "React layering policy: report-only mode completed with violations."
);
