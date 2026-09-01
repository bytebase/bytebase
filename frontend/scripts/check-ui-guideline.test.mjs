import { existsSync, readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, test } from "vitest";
import {
  compareWithBaseline,
  getBaselineUpdateIssues,
  isScannableSourcePath,
  scanCssSource,
  scanSource,
} from "./check-ui-guideline.mjs";

const FEATURE_FILE = "src/routes/project/Feature.tsx";
const SHARED_FILE = "src/components/ui/control.tsx";

describe("check-ui-guideline", () => {
  test("stores legacy debt as documented repository state", () => {
    const debtPath = join(import.meta.dirname, "ui-guideline-legacy-debt.json");
    const oldBaselinePath = join(import.meta.dirname, "ui-guideline-baseline.json");

    expect(existsSync(debtPath)).toBe(true);
    expect(existsSync(oldBaselinePath)).toBe(false);

    const debt = JSON.parse(readFileSync(debtPath, "utf8"));
    expect(debt.description).toBe(
      "Committed legacy UX debt. New or increased violations are rejected."
    );
    expect(debt.updateCommand).toBe(
      "node frontend/scripts/check-ui-guideline.mjs --write-baseline"
    );
    expect(debt.removalCondition).toBe(
      "Delete this file and baseline handling after all violations are fixed."
    );
  });

  test("scans product sources but not tests or generated declarations", () => {
    expect(isScannableSourcePath("src/routes/project/Feature.tsx")).toBe(true);
    expect(isScannableSourcePath("src/routes/project/Feature.test.tsx")).toBe(false);
    expect(isScannableSourcePath("src/routes/project/Feature.spec.ts")).toBe(false);
    expect(isScannableSourcePath("src/env.d.ts")).toBe(false);
  });

  test("allows semantic utilities and shared components", () => {
    const violations = scanSource(
      `
import { Button } from "@/components/ui/button";

export function Feature() {
  return (
    <div className="flex gap-x-2 bg-background text-control">
      <Button>Save</Button>
    </div>
  );
}
`,
      FEATURE_FILE
    );

    expect(violations).toEqual([]);
  });

  test("flags objective utility violations", () => {
    const violations = scanSource(
      `
export function Feature() {
  return (
    <div className="space-x-2 bg-gray-100 dark:bg-gray-900 gap-[10px] text-[13px] leading-[18px]" />
  );
}
`,
      FEATURE_FILE
    );

    expect(violations).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ rule: "no-space-between", token: "space-x-2" }),
        expect.objectContaining({ rule: "no-raw-color", token: "bg-gray-100" }),
        expect.objectContaining({ rule: "no-manual-dark", token: "dark:bg-gray-900" }),
        expect.objectContaining({ rule: "no-arbitrary-gap", token: "gap-[10px]" }),
        expect.objectContaining({ rule: "no-arbitrary-type", token: "text-[13px]" }),
        expect.objectContaining({ rule: "no-arbitrary-type", token: "leading-[18px]" }),
      ])
    );
  });

  test("flags spacing values outside the product scale", () => {
    const violations = scanSource(
      '<div className="gap-0 gap-1 gap-x-1.5 gap-y-2 gap-3 gap-4 gap-6 gap-8 gap-0.5 md:gap-x-5" />;',
      FEATURE_FILE
    );

    expect(violations).toEqual([
      expect.objectContaining({ rule: "no-off-scale-gap", token: "gap-0.5" }),
      expect.objectContaining({ rule: "no-off-scale-gap", token: "md:gap-x-5" }),
    ]);
  });

  test("flags radii outside the approved corner vocabulary", () => {
    const violations = scanSource(
      '<div className="rounded-none rounded-xs rounded-sm rounded-full rounded-t-none rounded-r-xs rounded-b-sm rounded-l-full rounded rounded-t rounded-md rounded-lg rounded-3xl md:rounded-t-lg hover:!rounded-lg rounded-[3px] rounded-[length:3px] rounded-(--card-radius)" />;',
      FEATURE_FILE
    );

    expect(violations).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ rule: "no-off-scale-radius", token: "rounded" }),
        expect.objectContaining({ rule: "no-off-scale-radius", token: "rounded-t" }),
        expect.objectContaining({ rule: "no-off-scale-radius", token: "rounded-md" }),
        expect.objectContaining({ rule: "no-off-scale-radius", token: "rounded-lg" }),
        expect.objectContaining({ rule: "no-off-scale-radius", token: "rounded-3xl" }),
        expect.objectContaining({
          rule: "no-off-scale-radius",
          token: "md:rounded-t-lg",
        }),
        expect.objectContaining({
          rule: "no-off-scale-radius",
          token: "rounded-[3px]",
        }),
        expect.objectContaining({
          rule: "no-off-scale-radius",
          token: "hover:!rounded-lg",
        }),
        expect.objectContaining({
          rule: "no-off-scale-radius",
          token: "rounded-[length:3px]",
        }),
        expect.objectContaining({
          rule: "no-off-scale-radius",
          token: "rounded-(--card-radius)",
        }),
      ])
    );
    expect(violations).toHaveLength(10);
  });

  test("flags inline React radius properties", () => {
    const violations = scanSource(
      `
const styles = {
  borderRadius: "4px",
  borderTopLeftRadius: radius,
  "border-bottom-right-radius": "var(--radius-sm)",
  padding: "4px",
};
`,
      FEATURE_FILE
    );

    expect(violations).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          rule: "no-off-scale-radius",
          token: "borderRadius",
        }),
        expect.objectContaining({
          rule: "no-off-scale-radius",
          token: "borderTopLeftRadius",
        }),
        expect.objectContaining({
          rule: "no-off-scale-radius",
          token: "border-bottom-right-radius",
        }),
      ])
    );
    expect(violations).toHaveLength(3);
  });

  test("flags shorthand inline React radius properties", () => {
    const violations = scanSource(
      `
const borderRadius = "12px";
const styles = { borderRadius };
`,
      FEATURE_FILE
    );

    expect(violations).toEqual([
      expect.objectContaining({
        rule: "no-off-scale-radius",
        token: "borderRadius",
      }),
    ]);
  });

  test("enforces the CSS radius vocabulary", () => {
    const violations = scanCssSource(
      `
.allowed {
  border-radius: 0;
  border-top-left-radius: var(--radius-xs);
  border-top-right-radius: var(--radius-sm);
  border-bottom-radius: var(--radius-full);
}
.invalid {
  border-radius: 4px;
  border-top-left-radius: 50%;
  border-bottom-right-radius: 0 4px;
  border-bottom-left-radius: var(--card-radius);
  border-start-start-radius: var(--radius-md);
}
.utility {
  @apply rounded-sm rounded-lg;
}
`,
      "src/assets/css/example.css"
    );

    expect(violations).toEqual([
      expect.objectContaining({
        rule: "no-off-scale-radius",
        token: "border-bottom-left-radius: var(--card-radius)",
      }),
      expect.objectContaining({
        path: "src/assets/css/example.css",
        rule: "no-off-scale-radius",
        token: "border-bottom-right-radius: 0 4px",
      }),
      expect.objectContaining({
        rule: "no-off-scale-radius",
        token: "border-radius: 4px",
      }),
      expect.objectContaining({
        rule: "no-off-scale-radius",
        token: "border-start-start-radius: var(--radius-md)",
      }),
      expect.objectContaining({
        rule: "no-off-scale-radius",
        token: "border-top-left-radius: 50%",
      }),
      expect.objectContaining({ rule: "no-off-scale-radius", token: "rounded-lg" }),
    ]);
  });

  test("enforces radius in legacy adapters but still ignores generated code", () => {
    const source = '<div className="rounded-lg bg-gray-100" />;';

    expect(
      scanSource(source, "src/apps/explain-visualizer/Feature.tsx")
    ).toEqual([
      expect.objectContaining({
        rule: "no-off-scale-radius",
        token: "rounded-lg",
      }),
    ]);
    expect(scanSource(source, "src/types/proto-es/generated.ts")).toEqual([]);
  });

  test("allows responsive Button sizing but flags other dimension overrides", () => {
    const violations = scanSource(
      `
export function Feature() {
  return (
    <>
      <Button size="sm" className="w-full" />
      <Button
        className={cn(
          "h-8",
          compact ? "md:size-7" : "",
          "hover:h-9 2xl:h-9 max-xl:size-7"
        )}
      />
    </>
  );
}
`,
      FEATURE_FILE
    );

    expect(violations).toEqual([
      expect.objectContaining({
        rule: "no-button-dimension-override",
        token: "h-8",
      }),
      expect.objectContaining({
        rule: "no-button-dimension-override",
        token: "hover:h-9",
      }),
    ]);
  });

  test("flags raw feature tables but allows the shared table owner", () => {
    const source = '<table><tbody><tr><td>Value</td></tr></tbody></table>;';

    expect(scanSource(source, FEATURE_FILE)).toEqual([
      expect.objectContaining({ rule: "no-raw-table", token: "table" }),
    ]);
    expect(scanSource(source, "src/components/ui/table.tsx")).toEqual([]);
  });

  test("flags literal colors assigned to CSS color properties", () => {
    const violations = scanSource(
      `
const direct = { backgroundColor: "#3B82F6" };
const fixed = { color: "hsl(220, 55%, 55%)" };
const dynamicDataColor = { color: \`hsl(\${hue}, 55%, 55%)\` };
const camelCase = { borderTopColor: "rgb(1 2 3)" };
const semantic = { borderColor: "rgb(var(--color-control-border))" };
const data = { color: colorToHex("#4f46e5") };
`,
      FEATURE_FILE
    );

    expect(violations).toEqual([
      expect.objectContaining({ rule: "no-literal-color", token: "#3B82F6" }),
      expect.objectContaining({
        rule: "no-literal-color",
        token: "hsl(",
        count: 1,
      }),
      expect.objectContaining({ rule: "no-literal-color", token: "rgb(" }),
    ]);
  });

  test("scans static portions of template expressions", () => {
    const violations = scanSource(
      'const classes = `flex ${compact ? "gap-1" : "gap-2"} bg-gray-100`;',
      FEATURE_FILE
    );

    expect(violations).toEqual([
      expect.objectContaining({ rule: "no-raw-color", token: "bg-gray-100" }),
    ]);
  });

  test("flags native feature controls but allows primitive owners", () => {
    const source = `
export function Control() {
  return <button><input /><select /><textarea /></button>;
}
`;

    expect(scanSource(source, FEATURE_FILE)).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ rule: "no-native-control", token: "button" }),
        expect.objectContaining({ rule: "no-native-control", token: "input" }),
        expect.objectContaining({ rule: "no-native-control", token: "select" }),
        expect.objectContaining({ rule: "no-native-control", token: "textarea" }),
      ])
    );
    expect(scanSource(source, SHARED_FILE)).toEqual([]);
  });

  test("flags ad hoc SheetContent widths", () => {
    const source = `
export function Feature() {
  return <SheetContent className="w-[42rem] max-w-4xl" />;
}
`;

    expect(scanSource(source, FEATURE_FILE)).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          rule: "no-ad-hoc-sheet-width",
          token: "w-[42rem]",
        }),
        expect.objectContaining({
          rule: "no-ad-hoc-sheet-width",
          token: "max-w-4xl",
        }),
      ])
    );
    expect(
      scanSource('<SheetContent width="wide" />;', FEATURE_FILE)
    ).toEqual([]);
  });

  test("accepts exact legacy fingerprints", () => {
    const violations = scanSource(
      '<div className="space-x-2 bg-gray-100" />;',
      FEATURE_FILE
    );
    const baseline = { version: 1, violations };

    expect(compareWithBaseline(violations, baseline)).toEqual([]);
  });

  test("rejects new and changed fingerprints", () => {
    const baselineViolations = scanSource(
      '<div className="space-x-2" />;',
      FEATURE_FILE
    );
    const currentViolations = scanSource(
      '<div className="space-x-4" />;',
      FEATURE_FILE
    );

    expect(
      compareWithBaseline(currentViolations, {
        version: 1,
        violations: baselineViolations,
      })
    ).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ kind: "new", token: "space-x-4" }),
        expect.objectContaining({ kind: "stale", token: "space-x-2" }),
      ])
    );
  });

  test("rejects stale baseline entries after debt is removed", () => {
    const baselineViolations = scanSource(
      '<div className="space-x-2" />;',
      FEATURE_FILE
    );

    expect(
      compareWithBaseline([], { version: 1, violations: baselineViolations })
    ).toEqual([
      expect.objectContaining({ kind: "stale", token: "space-x-2" }),
    ]);
  });

  test("rejects baseline exceptions for shared primitives", () => {
    const sharedViolation = {
      path: SHARED_FILE,
      rule: "no-raw-color",
      token: "bg-gray-100",
      count: 1,
    };

    expect(
      compareWithBaseline([sharedViolation], {
        version: 1,
        violations: [sharedViolation],
      })
    ).toEqual([
      expect.objectContaining({ kind: "shared", token: "bg-gray-100" }),
    ]);
  });

  test("allows baseline updates only when debt shrinks", () => {
    const oldViolations = scanSource(
      '<div className="space-x-2 space-x-2 bg-gray-100" />;',
      FEATURE_FILE
    );
    const reducedViolations = scanSource(
      '<div className="space-x-2" />;',
      FEATURE_FILE
    );

    expect(
      getBaselineUpdateIssues(reducedViolations, {
        version: 1,
        violations: oldViolations,
      })
    ).toEqual([]);
    expect(
      getBaselineUpdateIssues(
        scanSource('<div className="space-x-4" />;', FEATURE_FILE),
        { version: 1, violations: oldViolations }
      )
    ).toEqual([
      expect.objectContaining({ kind: "new", token: "space-x-4" }),
    ]);
  });

  test("allows baseline updates to record only newly enforced rules", () => {
    const oldViolation = {
      path: FEATURE_FILE,
      rule: "no-space-between",
      token: "space-x-2",
      count: 1,
    };
    const newRuleViolation = {
      path: FEATURE_FILE,
      rule: "no-raw-table",
      token: "table",
      count: 1,
    };
    const baseline = {
      version: 2,
      rules: ["no-space-between"],
      violations: [oldViolation],
    };

    expect(
      getBaselineUpdateIssues([oldViolation, newRuleViolation], baseline)
    ).toEqual([]);
    expect(
      getBaselineUpdateIssues(
        [{ ...oldViolation, count: 2 }, newRuleViolation],
        baseline
      )
    ).toEqual([
      expect.objectContaining({ kind: "new", rule: "no-space-between" }),
    ]);
  });
});
