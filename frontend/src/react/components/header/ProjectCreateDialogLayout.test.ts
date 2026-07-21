import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, test } from "vitest";

const componentDir = dirname(fileURLToPath(import.meta.url));

describe("ProjectCreateDialog layout", () => {
  test("uses shared form primitives in the create project drawer", () => {
    const source = readFileSync(
      join(componentDir, "ProjectCreateDialog.tsx"),
      "utf8"
    );

    expect(source).toContain("<FormFieldGroup>");
    expect(source).toContain("<FormField>");
    expect(source).toContain("FormTitle");
    expect(source).not.toContain("FormLabel");
    expect(source).toContain('id="create-project-title"');
    expect(source).not.toContain('className="flex flex-col gap-y-6"');
    expect(source).not.toContain("<label");
  });

  test("opens the database page after creating a project", () => {
    const source = readFileSync(
      join(componentDir, "ProjectCreateDialog.tsx"),
      "utf8"
    );

    expect(source).toContain("PROJECT_V1_ROUTE_DATABASES");
    expect(source).toContain("name: PROJECT_V1_ROUTE_DATABASES");
    expect(source).toContain(
      "projectId: extractProjectResourceName(createdProject.name)"
    );
    expect(source).toContain(
      "query: { [PRODUCT_INTRO_QUERY_KEY]: CONNECT_DATABASE_PRODUCT_INTRO }"
    );
    expect(source).not.toContain("path: `/${createdProject.name}`");
  });
});
