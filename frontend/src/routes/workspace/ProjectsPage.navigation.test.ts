import { describe, expect, test } from "vitest";

import source from "./ProjectsPage.tsx?raw";

describe("ProjectsPage navigation", () => {
  test("opens a newly created project on the databases page with the connect instance intro", () => {
    expect(source).toContain("PROJECT_V1_ROUTE_DATABASES");
    expect(source).toContain(
      "query: { [PRODUCT_INTRO_QUERY_KEY]: CONNECT_DATABASE_PRODUCT_INTRO }"
    );
  });

  test("opens projects from the table on the issues page", () => {
    expect(source).toContain("PROJECT_V1_ROUTE_ISSUES");
    expect(source).toContain("projectIssuesRoute(project)");
    expect(source).toContain("e.ctrlKey || e.metaKey");
    expect(source).toContain('window.open(route.fullPath, "_blank")');
    expect(source).toContain("markListScrollRestorationEntry();");
    expect(source).not.toContain("PROJECT_V1_ROUTE_DETAIL");
    expect(source).not.toContain("router.push({ path: `/${project.name}` })");
  });
});
