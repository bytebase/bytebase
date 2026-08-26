import { readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, test } from "vitest";

describe("InstanceDashboard", () => {
  test("does not expose force archive paths for project instances", () => {
    const source = readFileSync(
      join(process.cwd(), "src/components/instance/InstanceDashboard.tsx"),
      "utf-8"
    );

    expect(source).toMatch(
      /archiveInstance\(\s*instance,\s*project \? false : forceArchive\s*\)/
    );
    expect(source).toContain("{!project && (");
    expect(source).toContain("forceArchive={!project}");
  });

  test("resets scoped state when the project parent changes", () => {
    const source = readFileSync(
      join(process.cwd(), "src/components/instance/InstanceDashboard.tsx"),
      "utf-8"
    );

    expect(source).toContain("const previousParentRef = useRef(parent);");
    expect(source).toContain("fetchIdRef.current += 1;");
    expect(source).toContain("setInstances([]);");
    expect(source).toContain('nextPageTokenRef.current = "";');
    expect(source).toContain("setHasMore(false);");
    expect(source).toContain("setSelectedNames(new Set());");
    expect(source).toContain("}, [parent]);");
  });

  test("allows the connect-instance intro on project instance pages", () => {
    const source = readFileSync(
      join(process.cwd(), "src/components/instance/InstanceDashboard.tsx"),
      "utf-8"
    );

    expect(source).toContain("disabled: !canCreate,");
    expect(source).not.toContain(
      'disabled: layout === "project" || !canCreate,'
    );
  });
});
