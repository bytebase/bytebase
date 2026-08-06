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
});
