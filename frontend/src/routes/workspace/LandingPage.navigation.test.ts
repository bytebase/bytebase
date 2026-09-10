// @vitest-environment node
import { describe, expect, test } from "vitest";

import source from "./LandingPage.tsx?raw";

describe("LandingPage navigation", () => {
  test("routes the visit projects quick link to the projects page", () => {
    expect(source).toContain('id: "visit-projects"');
    expect(source).toContain("route: PROJECT_V1_ROUTE_DASHBOARD");
    expect(source).not.toContain("ProjectSwitchDialog");
    expect(source).not.toContain("setShowProjectSwitchDialog");
  });

  test("uses shared controls and semantic colors for quick-link interactions", () => {
    expect(source).toContain('import { Button } from "@/components/ui/button"');
    expect(source).not.toContain("<button");
    expect(source).not.toContain("hover:bg-gray-100");
    expect(source).not.toContain("text-gray-500");
  });
});
