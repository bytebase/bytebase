import { readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, test } from "vitest";

const frameworkDir = join(import.meta.dirname, "../../tests/e2e/framework");
const apiClientSource = readFileSync(
  join(frameworkDir, "api-client.ts"),
  "utf8"
);
const bootstrapSource = readFileSync(
  join(frameworkDir, "mode-start-new-bytebase.ts"),
  "utf8"
);
const setupProjectSource = readFileSync(
  join(frameworkDir, "setup-project.ts"),
  "utf8"
);

describe("E2E sample bootstrap", () => {
  test("uses the project-scoped sample API and its single-instance result", () => {
    expect(apiClientSource).not.toContain(
      ["/v1/actuator", ["setup", "Sample"].join("")].join(":")
    );
    expect(apiClientSource).toContain("instances:prepareSampleProjectInstance");

    expect(bootstrapSource).not.toContain(
      ["api", ["setup", "Sample()"].join("")].join(".")
    );
    expect(bootstrapSource).toContain("api.prepareSampleProjectInstance(");
    expect(bootstrapSource).not.toContain("instances/prod-sample-instance");
    expect(bootstrapSource).not.toContain("[port + 3, port + 4]");

    expect(setupProjectSource).toContain("updateDatabaseEnvironment(");
    expect(setupProjectSource).toContain('"environments/prod"');
  });
});
