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

const CLASS_ATTR_PATTERN = /\bclass(Name)?\s*=\s*/g;

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

const hasInlineZIndex = (line) => /\bzIndex\s*:/.test(line);

const buildLineStarts = (source) => {
  const lineStarts = [0];
  for (let index = source.indexOf("\n"); index !== -1; index = source.indexOf("\n", index + 1)) {
    lineStarts.push(index + 1);
  }
  return lineStarts;
};

const getLineNumber = (lineStarts, index) => {
  let low = 0;
  let high = lineStarts.length - 1;
  while (low <= high) {
    const mid = Math.floor((low + high) / 2);
    if (lineStarts[mid] <= index) {
      low = mid + 1;
    } else {
      high = mid - 1;
    }
  }
  return high + 1;
};

const extractClassExpression = (source, startIndex) => {
  let index = startIndex;
  while (index < source.length && /\s/.test(source[index])) {
    index++;
  }
  if (index >= source.length) {
    return null;
  }

  const start = index;
  const startChar = source[index];

  const scanQuoted = (quote) => {
    let current = index + 1;
    let escaped = false;
    while (current < source.length) {
      const ch = source[current];
      if (escaped) {
        escaped = false;
      } else if (ch === "\\") {
        escaped = true;
      } else if (ch === quote) {
        return { start, end: current + 1 };
      }
      current++;
    }
    return { start, end: source.length };
  };

  if (startChar === "\"" || startChar === "'" || startChar === "`") {
    return scanQuoted(startChar);
  }

  if (startChar !== "{") {
    let current = index;
    while (current < source.length && !/[\s>]/.test(source[current])) {
      current++;
    }
    return { start, end: current };
  }

  let current = index + 1;
  let depth = 1;
  let stringQuote = null;
  let escaped = false;
  while (current < source.length) {
    const ch = source[current];
    if (stringQuote !== null) {
      if (escaped) {
        escaped = false;
      } else if (ch === "\\") {
        escaped = true;
      } else if (ch === stringQuote) {
        stringQuote = null;
      }
      current++;
      continue;
    }

    if (ch === "\"" || ch === "'" || ch === "`") {
      stringQuote = ch;
      current++;
      continue;
    }
    if (ch === "{") {
      depth++;
    } else if (ch === "}") {
      depth--;
      if (depth === 0) {
        return { start, end: current + 1 };
      }
    }
    current++;
  }
  return { start, end: source.length };
};

const isRawZToken = (zMatch) =>
  zMatch[0] === "z-auto" ||
  zMatch[2] !== undefined ||
  (zMatch[1] !== undefined && Number.isFinite(Number(zMatch[1])));

const scanClassExpressions = (source, rel, lines) => {
  const violations = [];
  const lineStarts = buildLineStarts(source);

  CLASS_ATTR_PATTERN.lastIndex = 0;
  let match;
  while ((match = CLASS_ATTR_PATTERN.exec(source))) {
    const expr = extractClassExpression(source, match.index + match[0].length);
    if (!expr) {
      continue;
    }

    const block = source.slice(expr.start, expr.end);
    const fixedMatch = block.match(/\bfixed\b/);
    const absoluteMatch = block.match(/\babsolute\b/);
    const zMatches = [...block.matchAll(/\bz-auto\b|\bz-(\d+)\b|\bz-\[([^\]]+)\]/g)];
    if (zMatches.length === 0 || (!fixedMatch && !absoluteMatch)) {
      continue;
    }

    const isAbsoluteGlobal = zMatches.some((zMatch) => {
      if (zMatch[2] !== undefined) {
        return true;
      }
      const value = Number(zMatch[1]);
      return Number.isFinite(value) && value >= 40;
    });
    if (!fixedMatch && !isAbsoluteGlobal) {
      continue;
    }

    const reason = fixedMatch
      ? "feature-owned fixed overlay uses raw z-index"
      : "feature-owned absolute overlay uses high raw z-index";
    const tokenMatch = fixedMatch ?? absoluteMatch ?? zMatches.find((zMatch) => {
      if (isRawZToken(zMatch)) {
        return true;
      }
      const value = Number(zMatch[1]);
      return Number.isFinite(value) && value >= 40;
    });
    const tokenIndex = expr.start + (tokenMatch?.index ?? 0);
    const lineNumber = getLineNumber(lineStarts, tokenIndex);
    violations.push({
      rel,
      lineNumber,
      reason,
      line: lines[lineNumber - 1] ?? "",
    });
  }

  return violations;
};

const scanFile = (file) => {
  const rel = relative(ROOT, file);
  if (isApprovedPath(rel)) {
    return [];
  }

  const source = readFileSync(file, "utf-8");
  const violations = [];
  const lines = source.split("\n");
  violations.push(...scanClassExpressions(source, rel, lines));
  lines.forEach((line, index) => {
    const lineNumber = index + 1;
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
