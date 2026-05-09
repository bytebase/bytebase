import { readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, test } from "vitest";

describe("InstanceFormBody", () => {
  test("uses boolean values for inert collapsible sections", () => {
    const source = readFileSync(
      join(process.cwd(), "src/react/components/instance/InstanceFormBody.tsx"),
      "utf-8"
    );

    expect(source).not.toContain(
      'inert={isEngineSelectorCollapsed ? "" : undefined}'
    );
    expect(source).not.toContain(
      'inert={isConnectionOptionsCollapsed ? "" : undefined}'
    );
    expect(source).toContain(
      "inert={isEngineSelectorCollapsed ? true : undefined}"
    );
    expect(source).toContain(
      "inert={isConnectionOptionsCollapsed ? true : undefined}"
    );
  });
});
