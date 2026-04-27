import { describe, expect, test } from "vitest";

import source from "./ProjectsPage.tsx?raw";

describe("ProjectsPage navigation", () => {
  test("opens projects from the table on the issues page", () => {
    expect(source).toContain("PROJECT_V1_ROUTE_ISSUES");
    expect(source).toContain("projectIssuesRoute(project)");
    expect(source).not.toContain("PROJECT_V1_ROUTE_DETAIL");
    expect(source).not.toContain("router.push({ path: `/${project.name}` })");
  });
});
