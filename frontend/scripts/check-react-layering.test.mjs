import { describe, expect, test } from "vitest";
import { scanSource } from "./check-react-layering.mjs";

const FEATURE_FILE = "src/react/pages/project/Feature.tsx";
const APPROVED_FILE = "src/react/components/ui/overlay.tsx";

describe("check-react-layering", () => {
  test("flags raw overlay classes stored in local constants", () => {
    const violations = scanSource(
      `
const overlayClass = "fixed inset-0 z-50";

export function FeatureOverlay() {
  return <div className={overlayClass} />;
}
`,
      FEATURE_FILE
    );

    expect(violations).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          lineNumber: 2,
          reason: "feature-owned fixed overlay uses raw z-index",
        }),
      ])
    );
  });

  test("flags raw overlay classes split across template literal expressions", () => {
    const violations = scanSource(
      [
        'const overlayClass = `fixed ${condition ? "opacity-100" : ""} inset-0 z-50`;',
        "",
        "export function FeatureOverlay() {",
        "  return <div className={overlayClass} />;",
        "}",
        "",
      ].join("\n"),
      FEATURE_FILE
    );

    expect(violations).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          lineNumber: 1,
          reason: "feature-owned fixed overlay uses raw z-index",
        }),
      ])
    );
  });

  test("flags createPortal targets aliased from document.body", () => {
    const violations = scanSource(
      `
import { createPortal } from "react-dom";

const target = document.body;

export function FeatureOverlay() {
  return createPortal(<div />, target);
}
`,
      FEATURE_FILE
    );

    expect(violations).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          lineNumber: 7,
          reason: "feature-owned portal targets document.body directly",
        }),
      ])
    );
  });

  test("allows semantic layer owners to manage raw layers", () => {
    const violations = scanSource(
      `
import { createPortal } from "react-dom";

const target = document.body;
const overlayClass = "fixed inset-0 z-50";

export function OverlayRoot() {
  return createPortal(<div className={overlayClass} />, target);
}
`,
      APPROVED_FILE
    );

    expect(violations).toEqual([]);
  });
});
