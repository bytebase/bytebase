// Enforces the objective subset of docs/agents/frontend-ux.md.
//
// Existing feature debt is recorded as exact file/rule/token fingerprints in
// ui-guideline-legacy-debt.json. Shared primitives cannot use debt exceptions.
// This is a conservative static scanner, not a complete Tailwind evaluator.

import {
  existsSync,
  readFileSync,
  readdirSync,
  writeFileSync,
} from "node:fs";
import { dirname, relative, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import postcss from "postcss";
import ts from "typescript-6";

const scriptDir = dirname(fileURLToPath(import.meta.url));
const frontendRoot = resolve(scriptDir, "..");
const sourceRoot = resolve(frontendRoot, "src");
const baselinePath = resolve(scriptDir, "ui-guideline-legacy-debt.json");
const sharedPrimitivePrefix = "src/components/ui/";
const baselineDescription =
  "Committed legacy UX debt. New or increased violations are rejected.";
const baselineUpdateCommand =
  "node frontend/scripts/check-ui-guideline.mjs --write-baseline";
const baselineRemovalCondition =
  "Delete this file and baseline handling after all violations are fixed.";

const legacyRules = [
  "no-ad-hoc-sheet-width",
  "no-arbitrary-gap",
  "no-arbitrary-type",
  "no-manual-dark",
  "no-native-control",
  "no-raw-color",
  "no-space-between",
];
const enforcedRules = [
  ...legacyRules,
  "no-button-dimension-override",
  "no-literal-color",
  "no-off-scale-gap",
  "no-off-scale-radius",
  "no-raw-table",
].sort();

const legacyIgnoredPrefixes = ["src/apps/explain-visualizer/"];
const generatedPrefixes = ["src/types/proto-es/"];
const nativeControls = new Set(["button", "input", "select", "textarea"]);
const rawColorFamilies = [
  "slate",
  "gray",
  "zinc",
  "neutral",
  "stone",
  "red",
  "orange",
  "amber",
  "yellow",
  "lime",
  "green",
  "emerald",
  "teal",
  "cyan",
  "sky",
  "blue",
  "indigo",
  "violet",
  "purple",
  "fuchsia",
  "pink",
  "rose",
  "black",
  "white",
].join("|");
const rawColorProperties = [
  "accent",
  "bg",
  "border",
  "caret",
  "divide",
  "fill",
  "outline",
  "placeholder",
  "ring",
  "shadow",
  "stroke",
  "text",
].join("|");
const rawColorPattern = new RegExp(
  `(?:^|:)(?:${rawColorProperties})-(?:${rawColorFamilies})(?:-|/|$)`
);
const approvedGapValues = new Set(["0", "1", "1.5", "2", "3", "4", "6", "8"]);
const approvedRadiusValues = new Set(["none", "xs", "sm", "full"]);
const approvedCssRadiusValues = new Set([
  "0",
  "var(--radius-xs)",
  "var(--radius-sm)",
  "var(--radius-full)",
]);
const cssColorPropertyPattern = /^(?:color|[A-Za-z]+Color)$/;
const literalColorPattern = /#[\da-f]{3,8}\b|\b(?:rgb|rgba|hsl|hsla)\s*\(/gi;
const cssRadiusPropertyPattern = /^border(?:-[a-z]+)*-radius$/;
const inlineRadiusPropertyPattern = /^border(?:[A-Z][a-z]*)*Radius$/;
const dashedInlineRadiusPropertyPattern = /^border(?:-[a-z]+)*-radius$/;

const isLegacyIgnoredPath = (path) =>
  legacyIgnoredPrefixes.some((prefix) => path.startsWith(prefix));

const isGeneratedPath = (path) =>
  generatedPrefixes.some((prefix) => path.startsWith(prefix));

const isSharedPrimitive = (path) => path.startsWith(sharedPrimitivePrefix);

const sourceKind = (path) =>
  path.endsWith(".tsx") ? ts.ScriptKind.TSX : ts.ScriptKind.TS;

const createSourceFile = (source, path) =>
  ts.createSourceFile(path, source, ts.ScriptTarget.Latest, true, sourceKind(path));

const getTagName = (tagName) => {
  if (ts.isIdentifier(tagName)) return tagName.text;
  if (ts.isPropertyAccessExpression(tagName)) return tagName.name.text;
  return tagName.getText();
};

const staticStrings = (node) => {
  if (ts.isStringLiteral(node) || ts.isNoSubstitutionTemplateLiteral(node)) {
    return [node.text];
  }
  if (ts.isTemplateExpression(node)) {
    return [
      node.head.text,
      ...node.templateSpans.flatMap((span) => [
        ...staticStrings(span.expression),
        span.literal.text,
      ]),
    ];
  }
  if (ts.isBinaryExpression(node) && node.operatorToken.kind === ts.SyntaxKind.PlusToken) {
    return [...staticStrings(node.left), ...staticStrings(node.right)];
  }
  if (ts.isCallExpression(node)) {
    return node.arguments.flatMap(staticStrings);
  }
  if (ts.isConditionalExpression(node)) {
    return [...staticStrings(node.whenTrue), ...staticStrings(node.whenFalse)];
  }
  if (ts.isParenthesizedExpression(node)) return staticStrings(node.expression);
  return [];
};

const classTokens = (value) => value.split(/\s+/).filter(Boolean);

const tailwindUtility = (token) => {
  let bracketDepth = 0;
  let utilityStart = 0;
  for (let index = 0; index < token.length; index++) {
    const character = token[index];
    if (character === "[") bracketDepth++;
    if (character === "]") bracketDepth--;
    if (character === ":" && bracketDepth === 0) utilityStart = index + 1;
  }
  return token.slice(utilityStart).replace(/^!|!$/g, "");
};

const attributeTokens = (node, attributeNames) => {
  const attribute = node.attributes.properties.find(
    (property) =>
      ts.isJsxAttribute(property) &&
      attributeNames.has(property.name.getText())
  );
  if (!attribute || !ts.isJsxAttribute(attribute) || !attribute.initializer) {
    return [];
  }

  const initializer = attribute.initializer;
  const values = ts.isJsxExpression(initializer) && initializer.expression
    ? staticStrings(initializer.expression)
    : staticStrings(initializer);
  return values.flatMap(classTokens);
};

const rulesForToken = (token) => {
  const rules = [];
  if (/(?:^|:)space-[xy]-/.test(token)) rules.push("no-space-between");
  if (rawColorPattern.test(token)) rules.push("no-raw-color");
  if (token.includes("dark:")) rules.push("no-manual-dark");
  if (/(?:^|:)(?:gap|gap-x|gap-y)-\[[^\]]+\]/.test(token)) {
    rules.push("no-arbitrary-gap");
  }
  const gapMatch = token.match(
    /(?:^|:)(?:gap|gap-x|gap-y)-(\d+(?:\.\d+)?)!?$/
  );
  if (gapMatch && !approvedGapValues.has(gapMatch[1])) {
    rules.push("no-off-scale-gap");
  }
  const utility = tailwindUtility(token);
  if (utility) {
    const bareRadius = /^rounded(?:-[trblse]{1,2})?$/.test(utility);
    const radiusMatch = utility.match(
      /^rounded(?:-[trblse]{1,2})?-(\[[^\]]+\]|[a-z0-9.]+)$/
    );
    if (
      bareRadius ||
      (radiusMatch && !approvedRadiusValues.has(radiusMatch[1]))
    ) {
      rules.push("no-off-scale-radius");
    }
  }
  if (/(?:^|:)(?:text|leading)-\[[^\]]+\]/.test(token)) {
    rules.push("no-arbitrary-type");
  }
  return rules;
};

const addViolation = (counts, path, rule, token) => {
  const key = JSON.stringify([path, rule, token]);
  const current = counts.get(key);
  counts.set(key, {
    path,
    rule,
    token,
    count: (current?.count ?? 0) + 1,
  });
};

const sortViolations = (violations) =>
  [...violations].sort(
    (a, b) =>
      a.path.localeCompare(b.path) ||
      a.rule.localeCompare(b.rule) ||
      a.token.localeCompare(b.token)
  );

export const isScannableSourcePath = (path) =>
  /\.(?:ts|tsx)$/.test(path) &&
  !path.endsWith(".d.ts") &&
  !/\.(?:test|spec)\.(?:ts|tsx)$/.test(path);

const isScannableCssPath = (path) => path.endsWith(".css");

const isScannablePath = (path) =>
  isScannableSourcePath(path) || isScannableCssPath(path);

const scanSheetWidth = (node, path, counts) => {
  if (getTagName(node.tagName) !== "SheetContent") return;
  for (const token of attributeTokens(node, new Set(["className"]))) {
    if (/(?:^|:)(?:w|min-w|max-w)-/.test(token)) {
      addViolation(counts, path, "no-ad-hoc-sheet-width", token);
    }
  }
};

const scanButtonDimensions = (node, path, counts) => {
  if (isSharedPrimitive(path) || getTagName(node.tagName) !== "Button") return;
  for (const token of attributeTokens(node, new Set(["class", "className"]))) {
    const isDimensionOverride = /(?:^|:)(?:h|size)-/.test(token);
    const isResponsiveOverride =
      /^(?:sm|md|lg|xl|2xl|max-(?:sm|md|lg|xl|2xl)):(?:h|size)-/.test(
        token
      );
    if (isDimensionOverride && !isResponsiveOverride) {
      addViolation(counts, path, "no-button-dimension-override", token);
    }
  }
};

const scanLiteralColor = (node, path, counts) => {
  if (!ts.isPropertyAssignment(node)) return;
  const propertyName = ts.isIdentifier(node.name) || ts.isStringLiteral(node.name)
    ? node.name.text
    : "";
  if (!cssColorPropertyPattern.test(propertyName)) return;

  const initializer = node.initializer;
  const values = ts.isStringLiteral(initializer) ||
      ts.isNoSubstitutionTemplateLiteral(initializer)
    ? [initializer.text]
    : [];
  for (const value of values) {
    if (value.includes("var(--")) continue;
    for (const match of value.matchAll(literalColorPattern)) {
      addViolation(counts, path, "no-literal-color", match[0]);
    }
  }
};

const propertyName = (node) =>
  ts.isIdentifier(node) || ts.isStringLiteral(node) ? node.text : "";

const scanInlineRadius = (node, path, counts) => {
  if (!ts.isPropertyAssignment(node)) return;
  const name = propertyName(node.name);
  if (
    !inlineRadiusPropertyPattern.test(name) &&
    !dashedInlineRadiusPropertyPattern.test(name)
  ) {
    return;
  }
  addViolation(counts, path, "no-off-scale-radius", name);
};

export function scanSource(source, path) {
  if (isGeneratedPath(path)) return [];

  const sourceFile = createSourceFile(source, path);
  const counts = new Map();
  const scanTokenValue = (value) => {
    for (const token of classTokens(value)) {
      for (const rule of rulesForToken(token)) {
        if (isLegacyIgnoredPath(path) && rule !== "no-off-scale-radius") {
          continue;
        }
        addViolation(counts, path, rule, token);
      }
    }
  };
  const visit = (node) => {
    if (ts.isStringLiteral(node) || ts.isNoSubstitutionTemplateLiteral(node)) {
      scanTokenValue(node.text);
    }
    if (ts.isTemplateExpression(node)) {
      scanTokenValue(node.head.text);
      for (const span of node.templateSpans) scanTokenValue(span.literal.text);
    }

    if (ts.isJsxOpeningElement(node) || ts.isJsxSelfClosingElement(node)) {
      const tagName = getTagName(node.tagName);
      if (!isSharedPrimitive(path) && nativeControls.has(tagName)) {
        addViolation(counts, path, "no-native-control", tagName);
      }
      if (!isSharedPrimitive(path) && tagName === "table") {
        addViolation(counts, path, "no-raw-table", tagName);
      }
      scanSheetWidth(node, path, counts);
      scanButtonDimensions(node, path, counts);
    }
    scanLiteralColor(node, path, counts);
    scanInlineRadius(node, path, counts);
    ts.forEachChild(node, visit);
  };
  visit(sourceFile);
  return sortViolations(counts.values());
}

export function scanCssSource(source, path) {
  if (isGeneratedPath(path)) return [];

  const counts = new Map();
  const root = postcss.parse(source, { from: path });
  root.walkDecls((declaration) => {
    if (!cssRadiusPropertyPattern.test(declaration.prop)) return;
    const value = declaration.value.trim();
    if (!approvedCssRadiusValues.has(value)) {
      addViolation(
        counts,
        path,
        "no-off-scale-radius",
        `${declaration.prop}: ${value}`
      );
    }
  });
  root.walkAtRules("apply", (atRule) => {
    for (const token of classTokens(atRule.params)) {
      if (rulesForToken(token).includes("no-off-scale-radius")) {
        addViolation(counts, path, "no-off-scale-radius", token);
      }
    }
  });
  return sortViolations(counts.values());
}

const violationKey = ({ path, rule, token }) =>
  JSON.stringify([path, rule, token]);

export function compareWithBaseline(violations, baseline) {
  if (
    ![1, 2].includes(baseline.version) ||
    !Array.isArray(baseline.violations) ||
    (baseline.version === 2 && !Array.isArray(baseline.rules))
  ) {
    return [{ kind: "invalid", rule: "baseline", token: "version" }];
  }

  const issues = [];
  const currentByKey = new Map();
  const baselineByKey = new Map();

  for (const violation of violations) {
    if (isSharedPrimitive(violation.path)) {
      issues.push({ kind: "shared", ...violation });
      continue;
    }
    currentByKey.set(violationKey(violation), violation);
  }
  for (const violation of baseline.violations) {
    if (!isSharedPrimitive(violation.path)) {
      baselineByKey.set(violationKey(violation), violation);
    }
  }

  for (const [key, violation] of currentByKey) {
    const expected = baselineByKey.get(key);
    if (!expected || expected.count !== violation.count) {
      issues.push({ kind: "new", ...violation });
    }
  }
  for (const [key, violation] of baselineByKey) {
    const current = currentByKey.get(key);
    if (!current || current.count !== violation.count) {
      issues.push({ kind: "stale", ...violation });
    }
  }

  return sortViolations(issues);
}

export function getBaselineUpdateIssues(violations, baseline) {
  if (
    ![1, 2].includes(baseline.version) ||
    !Array.isArray(baseline.violations) ||
    (baseline.version === 2 && !Array.isArray(baseline.rules))
  ) {
    return [{ kind: "invalid", rule: "baseline", token: "version" }];
  }

  const previouslyEnforcedRules = new Set(
    baseline.version === 1 ? legacyRules : baseline.rules
  );

  const expectedByKey = new Map(
    baseline.violations
      .filter((violation) => !isSharedPrimitive(violation.path))
      .map((violation) => [violationKey(violation), violation])
  );
  const issues = [];
  for (const violation of violations) {
    if (isSharedPrimitive(violation.path)) {
      issues.push({ kind: "shared", ...violation });
      continue;
    }
    if (!previouslyEnforcedRules.has(violation.rule)) continue;
    const expected = expectedByKey.get(violationKey(violation));
    if (!expected || violation.count > expected.count) {
      issues.push({ kind: "new", ...violation });
    }
  }
  return sortViolations(issues);
}

const findSourceFiles = (directory) =>
  readdirSync(directory, { withFileTypes: true }).flatMap((entry) => {
    const path = resolve(directory, entry.name);
    if (entry.isDirectory()) return findSourceFiles(path);
    if (!isScannablePath(path)) return [];
    return [path];
  });

const scanFrontend = () =>
  sortViolations(
    findSourceFiles(sourceRoot).flatMap((file) => {
      const path = relative(frontendRoot, file);
      const source = readFileSync(file, "utf8");
      return isScannableCssPath(path)
        ? scanCssSource(source, path)
        : scanSource(source, path);
    })
  );

const printViolation = (violation) => {
  const count = violation.count > 1 ? ` (${violation.count} occurrences)` : "";
  console.error(
    `- ${violation.path}: ${violation.rule}: ${violation.token}${count}`
  );
};

const run = () => {
  const writeBaseline = process.argv.includes("--write-baseline");
  const reportOnly = process.argv.includes("--report-only");
  const violations = scanFrontend();

  if (writeBaseline) {
    const previousBaseline = existsSync(baselinePath)
      ? JSON.parse(readFileSync(baselinePath, "utf8"))
      : { version: 1, violations: [] };
    const updateIssues = existsSync(baselinePath)
      ? getBaselineUpdateIssues(violations, previousBaseline)
      : violations
          .filter((violation) => isSharedPrimitive(violation.path))
          .map((violation) => ({ kind: "shared", ...violation }));
    if (updateIssues.length > 0) {
      console.error(
        "The UI guideline baseline may only remove existing debt. Fix new and shared violations first:\n"
      );
      updateIssues.forEach(printViolation);
      process.exit(1);
    }
    const baseline = {
      description: baselineDescription,
      updateCommand: baselineUpdateCommand,
      removalCondition: baselineRemovalCondition,
      version: 2,
      rules: enforcedRules,
      violations,
    };
    writeFileSync(baselinePath, `${JSON.stringify(baseline, null, 2)}\n`);
    console.log(`Wrote ${violations.length} UI guideline fingerprints`);
    return;
  }

  if (reportOnly) {
    if (violations.length === 0) {
      console.log("No UI guideline violations found");
      return;
    }
    console.error(`Found ${violations.length} UI guideline fingerprints:\n`);
    violations.forEach(printViolation);
    return;
  }

  if (!existsSync(baselinePath)) {
    console.error(
      "UI guideline baseline is missing. Run with --write-baseline after fixing shared violations."
    );
    process.exit(1);
  }

  const baseline = JSON.parse(readFileSync(baselinePath, "utf8"));
  const issues = compareWithBaseline(violations, baseline);
  if (issues.length > 0) {
    console.error("UI guideline ratchet failed:\n");
    for (const issue of issues) {
      const prefix =
        issue.kind === "stale"
          ? "remove stale baseline"
          : issue.kind === "shared"
            ? "fix shared primitive"
            : issue.kind === "new"
              ? "new violation"
              : "invalid baseline";
      console.error(`${prefix}:`);
      printViolation(issue);
    }
    process.exit(1);
  }

  console.log(`UI guideline check passed (${violations.length} legacy fingerprints)`);
};

if (process.argv[1] && fileURLToPath(import.meta.url) === resolve(process.argv[1])) {
  run();
}
