// frontend/scripts/check-react-layering.mjs
//
// Enforces the React semantic overlay layering policy introduced by PR #20012.
// Feature code must not create global z-index overlays directly. Use shared UI
// primitives or portal into the semantic layer roots instead.
//
// This is a conservative policy scanner, not a complete TypeScript evaluator.
// It resolves direct string literals, unique static identifiers, simple static
// string concatenations, named createPortal imports, and simple
// document/document.body aliases. Complex, dynamic, imported, or shadowed
// values may be unresolved; passing this check does not permit bypassing the
// policy.

import { readFileSync, readdirSync } from "node:fs";
import { dirname, relative, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import ts from "typescript";

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
  "src/react/components/Watermark.tsx",
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

const getRawOverlayViolation = (block) => {
  const fixedMatch = block.match(/\bfixed\b/);
  const absoluteMatch = block.match(/\babsolute\b/);
  const zMatches = [...block.matchAll(/\bz-auto\b|\bz-(\d+)\b|\bz-\[([^\]]+)\]/g)];
  if (zMatches.length === 0 || (!fixedMatch && !absoluteMatch)) {
    return null;
  }

  const isAbsoluteGlobal = zMatches.some((zMatch) => {
    if (zMatch[2] !== undefined) {
      return true;
    }
    const value = Number(zMatch[1]);
    return Number.isFinite(value) && value >= 40;
  });
  if (!fixedMatch && !isAbsoluteGlobal) {
    return null;
  }

  const tokenMatch = fixedMatch ?? absoluteMatch ?? zMatches.find((zMatch) => {
    if (isRawZToken(zMatch)) {
      return true;
    }
    const value = Number(zMatch[1]);
    return Number.isFinite(value) && value >= 40;
  });

  return {
    reason: fixedMatch
      ? "feature-owned fixed overlay uses raw z-index"
      : "feature-owned absolute overlay uses high raw z-index",
    index: tokenMatch?.index ?? 0,
  };
};

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

const skipWhitespace = (source, startIndex) => {
  let index = startIndex;
  while (index < source.length && /\s/.test(source[index])) {
    index++;
  }
  return index;
};

const scanQuotedExpression = (source, index, quote) => {
  const start = index;
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

const scanBareExpression = (source, index) => {
  let current = index;
  while (current < source.length && !/[\s>]/.test(source[current])) {
    current++;
  }
  return { start: index, end: current };
};

const isStringQuote = (ch) => ch === "\"" || ch === "'" || ch === "`";

const scanStringState = (ch, stringQuote, escaped) => {
  if (escaped) {
    return { stringQuote, escaped: false };
  }
  if (ch === "\\") {
    return { stringQuote, escaped: true };
  }
  if (ch === stringQuote) {
    return { stringQuote: null, escaped: false };
  }
  return { stringQuote, escaped: false };
};

const getNextBraceDepth = (ch, depth) => {
  if (ch === "{") {
    return depth + 1;
  }
  if (ch === "}") {
    return depth - 1;
  }
  return depth;
};

const scanBalancedExpression = (source, index) => {
  let current = index + 1;
  let depth = 1;
  let stringQuote = null;
  let escaped = false;
  while (current < source.length) {
    const ch = source[current];
    if (stringQuote !== null) {
      ({ stringQuote, escaped } = scanStringState(ch, stringQuote, escaped));
      current++;
      continue;
    }

    if (isStringQuote(ch)) {
      stringQuote = ch;
      current++;
      continue;
    }
    depth = getNextBraceDepth(ch, depth);
    if (depth === 0) {
      return { start: index, end: current + 1 };
    }
    current++;
  }
  return { start: index, end: source.length };
};

const extractClassExpression = (source, startIndex) => {
  const index = skipWhitespace(source, startIndex);
  if (index >= source.length) {
    return null;
  }

  const startChar = source[index];
  if (startChar === "\"" || startChar === "'" || startChar === "`") {
    return scanQuotedExpression(source, index, startChar);
  }
  if (startChar === "{") {
    return scanBalancedExpression(source, index);
  }
  return scanBareExpression(source, index);
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
    const violation = getRawOverlayViolation(block);
    if (!violation) {
      continue;
    }

    const tokenIndex = expr.start + violation.index;
    const lineNumber = getLineNumber(lineStarts, tokenIndex);
    violations.push({
      rel,
      lineNumber,
      reason: violation.reason,
      line: lines[lineNumber - 1] ?? "",
    });
  }

  return violations;
};

const createSourceFile = (source, rel) =>
  ts.createSourceFile(
    rel,
    source,
    ts.ScriptTarget.Latest,
    true,
    rel.endsWith(".tsx") ? ts.ScriptKind.TSX : ts.ScriptKind.TS
  );

const getUniqueStaticString = (staticStrings, name) => {
  const entry = staticStrings.get(name);
  if (!entry || entry.count !== 1) {
    return null;
  }
  return entry.value;
};

const getDirectStaticString = (expression) => {
  const current = skipExpressionWrappers(expression);
  if (ts.isStringLiteral(current) || ts.isNoSubstitutionTemplateLiteral(current)) {
    return current.text;
  }
  return null;
};

const getStaticString = (expression, staticStrings = new Map()) => {
  const current = skipExpressionWrappers(expression);
  const directValue = getDirectStaticString(current);
  if (directValue !== null) {
    return directValue;
  }
  if (ts.isIdentifier(current)) {
    return getUniqueStaticString(staticStrings, current.text);
  }
  if (
    ts.isBinaryExpression(current) &&
    current.operatorToken.kind === ts.SyntaxKind.PlusToken
  ) {
    const left = getStaticString(current.left, staticStrings);
    const right = getStaticString(current.right, staticStrings);
    if (left !== null && right !== null) {
      return `${left}${right}`;
    }
  }
  return null;
};

const recordStaticString = (staticStrings, name, value) => {
  const current = staticStrings.get(name);
  staticStrings.set(name, {
    count: (current?.count ?? 0) + 1,
    value: current ? null : value,
  });
};

const collectStaticStrings = (sourceFile) => {
  const staticStrings = new Map();

  const visit = (node) => {
    if (
      ts.isVariableDeclaration(node) &&
      ts.isIdentifier(node.name)
    ) {
      recordStaticString(
        staticStrings,
        node.name.text,
        node.initializer ? getStaticString(node.initializer, staticStrings) : null
      );
    } else if (ts.isParameter(node) && ts.isIdentifier(node.name)) {
      recordStaticString(staticStrings, node.name.text, null);
    }
    ts.forEachChild(node, visit);
  };

  visit(sourceFile);
  return staticStrings;
};

const getTemplateParts = (sourceFile, node, staticStrings) => {
  const headPart = {
    text: node.head.text,
    start: node.head.getStart(sourceFile) + 1,
  };

  const spanParts = node.templateSpans.flatMap((span) => {
    const expressionValue = getStaticString(span.expression, staticStrings);
    const literalPart = {
      text: span.literal.text,
      start: span.literal.getStart(sourceFile) + 1,
    };
    if (expressionValue === null) {
      return [literalPart];
    }
    return [
      {
        text: expressionValue,
        start: span.expression.getStart(sourceFile),
      },
      literalPart,
    ];
  });

  return [
    headPart,
    ...spanParts,
  ];
};

const combineTemplateParts = (parts, fallbackIndex) => {
  const placeholder = " ";
  const combinedLineIndexes = [];
  const combinedText = parts
    .map((part) => part.text)
    .join(placeholder);

  let combinedOffset = 0;
  parts.forEach((part, partIndex) => {
    for (let index = 0; index < part.text.length; index++) {
      combinedLineIndexes[combinedOffset + index] = part.start + index;
    }
    combinedOffset += part.text.length;
    if (partIndex < parts.length - 1) {
      combinedLineIndexes[combinedOffset] = part.start;
      combinedOffset += placeholder.length;
    }
  });

  return {
    text: combinedText,
    getAbsoluteIndex: (index) => combinedLineIndexes[index] ?? fallbackIndex,
  };
};

const scanTemplateExpression = (sourceFile, node, staticStrings, scanText) => {
  const combined = combineTemplateParts(
    getTemplateParts(sourceFile, node, staticStrings),
    node.getStart(sourceFile)
  );
  const violation = getRawOverlayViolation(combined.text);
  if (violation) {
    scanText(combined.text, node.getStart(sourceFile), combined.getAbsoluteIndex);
  }
};

const scanStringLiteralNode = (sourceFile, node, scanText) => {
  scanText(node.text, node.getStart(sourceFile) + 1);
};

const scanLiteralNode = (sourceFile, node, staticStrings, scanText) => {
  if (ts.isStringLiteral(node) || ts.isNoSubstitutionTemplateLiteral(node)) {
    scanStringLiteralNode(sourceFile, node, scanText);
    return;
  }
  if (ts.isTemplateExpression(node)) {
    scanTemplateExpression(sourceFile, node, staticStrings, scanText);
  }
};

const scanStringLiterals = (sourceFile, rel, lines) => {
  const violations = [];
  const staticStrings = collectStaticStrings(sourceFile);

  const scanText = (
    text,
    startIndex,
    getAbsoluteIndex = (index) => startIndex + index
  ) => {
    const violation = getRawOverlayViolation(text);
    if (!violation) {
      return;
    }

    const lineNumber =
      sourceFile.getLineAndCharacterOfPosition(
        getAbsoluteIndex(violation.index)
      ).line + 1;
    violations.push({
      rel,
      lineNumber,
      reason: violation.reason,
      line: lines[lineNumber - 1] ?? "",
    });
  };

  const visit = (node) => {
    if (ts.isVariableDeclaration(node) && node.initializer) {
      const staticValue = getStaticString(node.initializer, staticStrings);
      if (staticValue !== null) {
        scanText(staticValue, node.initializer.getStart(sourceFile));
      }
    }
    scanLiteralNode(sourceFile, node, staticStrings, scanText);
    ts.forEachChild(node, visit);
  };

  visit(sourceFile);
  return violations;
};

const skipExpressionWrappers = (expression) => {
  let current = expression;
  while (
    ts.isParenthesizedExpression(current) ||
    ts.isAsExpression(current) ||
    ts.isNonNullExpression(current) ||
    ts.isTypeAssertionExpression(current)
  ) {
    current = current.expression;
  }
  return current;
};

const isDocumentExpression = (expression, documentAliases) => {
  const current = skipExpressionWrappers(expression);
  return (
    ts.isIdentifier(current) &&
    (current.text === "document" || documentAliases.has(current.text))
  );
};

const isDocumentBodyExpression = (
  expression,
  documentAliases,
  documentBodyAliases
) => {
  const current = skipExpressionWrappers(expression);
  if (ts.isIdentifier(current)) {
    return documentBodyAliases.has(current.text);
  }
  return (
    ts.isPropertyAccessExpression(current) &&
    current.name.text === "body" &&
    isDocumentExpression(current.expression, documentAliases)
  );
};

const collectCreatePortalAliases = (sourceFile) => {
  const aliases = new Set(["createPortal"]);

  const visit = (node) => {
    if (
      ts.isImportDeclaration(node) &&
      ts.isStringLiteral(node.moduleSpecifier) &&
      node.moduleSpecifier.text === "react-dom" &&
      node.importClause?.namedBindings &&
      ts.isNamedImports(node.importClause.namedBindings)
    ) {
      node.importClause.namedBindings.elements.forEach((element) => {
        const importedName = element.propertyName?.text ?? element.name.text;
        if (importedName === "createPortal") {
          aliases.add(element.name.text);
        }
      });
    }

    ts.forEachChild(node, visit);
  };

  visit(sourceFile);
  return aliases;
};

const isCreatePortalCall = (node, createPortalAliases) => {
  const expression = skipExpressionWrappers(node.expression);
  return (
    (ts.isIdentifier(expression) && createPortalAliases.has(expression.text)) ||
    (ts.isPropertyAccessExpression(expression) &&
      expression.name.text === "createPortal")
  );
};

const recordDocumentAliases = (
  declaration,
  documentAliases,
  documentBodyAliases
) => {
  if (!declaration.initializer) {
    return;
  }

  if (ts.isIdentifier(declaration.name)) {
    if (isDocumentExpression(declaration.initializer, documentAliases)) {
      documentAliases.add(declaration.name.text);
    }
    if (
      isDocumentBodyExpression(
        declaration.initializer,
        documentAliases,
        documentBodyAliases
      )
    ) {
      documentBodyAliases.add(declaration.name.text);
    }
    return;
  }

  if (
    !ts.isObjectBindingPattern(declaration.name) ||
    !isDocumentExpression(declaration.initializer, documentAliases)
  ) {
    return;
  }

  declaration.name.elements.forEach((element) => {
    if (!ts.isIdentifier(element.name)) {
      return;
    }
    const propertyName = element.propertyName;
    const sourceName =
      propertyName && ts.isIdentifier(propertyName)
        ? propertyName.text
        : element.name.text;
    if (sourceName === "body") {
      documentBodyAliases.add(element.name.text);
    }
  });
};

const scanPortalTargets = (sourceFile, rel, lines) => {
  const violations = [];
  const createPortalAliases = collectCreatePortalAliases(sourceFile);
  const documentAliases = new Set();
  const documentBodyAliases = new Set();

  const visit = (node) => {
    if (ts.isVariableDeclaration(node)) {
      recordDocumentAliases(node, documentAliases, documentBodyAliases);
    }

    if (
      ts.isCallExpression(node) &&
      isCreatePortalCall(node, createPortalAliases) &&
      node.arguments[1] &&
      isDocumentBodyExpression(
        node.arguments[1],
        documentAliases,
        documentBodyAliases
      )
    ) {
      const lineNumber =
        sourceFile.getLineAndCharacterOfPosition(node.getStart(sourceFile))
          .line + 1;
      violations.push({
        rel,
        lineNumber,
        reason: "feature-owned portal targets document.body directly",
        line: lines[lineNumber - 1] ?? "",
      });
    }

    ts.forEachChild(node, visit);
  };

  visit(sourceFile);
  return violations;
};

const dedupeViolations = (violations) => {
  const seen = new Set();
  return violations.filter((violation) => {
    const key = `${violation.rel}:${violation.lineNumber}:${violation.reason}`;
    if (seen.has(key)) {
      return false;
    }
    seen.add(key);
    return true;
  });
};

const scanInlineZIndex = (rel, lines) =>
  lines.flatMap((line, index) => {
    if (!hasInlineZIndex(line)) {
      return [];
    }
    return [
      {
        rel,
        lineNumber: index + 1,
        reason: "inline zIndex bypasses semantic layer ownership",
        line,
      },
    ];
  });

export const scanSource = (source, rel) => {
  if (isApprovedPath(rel)) {
    return [];
  }

  const violations = [];
  const lines = source.split("\n");
  const sourceFile = createSourceFile(source, rel);
  violations.push(
    ...scanClassExpressions(source, rel, lines),
    ...scanStringLiterals(sourceFile, rel, lines),
    ...scanInlineZIndex(rel, lines),
    ...scanPortalTargets(sourceFile, rel, lines)
  );

  return dedupeViolations(violations);
};

export const scanFile = (file) => {
  const rel = relative(ROOT, file);
  const source = readFileSync(file, "utf-8");
  return scanSource(source, rel);
};

export const scanReactLayering = () => findFiles(REACT_DIR).flatMap(scanFile);

const main = () => {
  const violations = scanReactLayering();

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
};

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  main();
}
