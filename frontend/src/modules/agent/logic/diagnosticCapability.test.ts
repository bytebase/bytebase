import { describe, expect, test } from "vitest";
import { normalizeAgentDiagnosticCapability } from "./diagnosticCapability";

describe("normalizeAgentDiagnosticCapability", () => {
  test.each([
    ["plan", "projects/example/plans/42"],
    ["issue", "projects/example/issues/42"],
    ["database", "instances/example/databases/app"],
    ["database", "projects/example/instances/example/databases/app"],
  ])("copies a concrete %s reference", (kind, name) => {
    const raw = {
      kind: "diagnostic",
      resource: { kind, name, project: "projects/example", label: " Example " },
    };

    const capability = normalizeAgentDiagnosticCapability(raw);

    expect(capability).toEqual({
      kind: "diagnostic",
      resource: { ...raw.resource, label: name },
    });
    expect(capability).not.toBe(raw);
    expect(capability?.resource).not.toBe(raw.resource);
  });

  test.each([
    undefined,
    null,
    [],
    { kind: "unknown" },
    { kind: "diagnostic" },
    { kind: "diagnostic", resource: null },
    { kind: "project", name: "projects/example" },
    ...[
      ["project", "projects/example", "projects/example"],
      ["plan", "projects/other/plans/42", "projects/example"],
      ["issue", "projects/other/issues/42", "projects/example"],
      [
        "database",
        "projects/other/instances/example/databases/app",
        "projects/example",
      ],
      ["plan", "projects/example/plans/-", "projects/example"],
      ["plan", "projects/example/plans/create", "projects/example"],
      ["issue", "projects/example/issues/create", "projects/example"],
      ["plan", "projects/example/plans/password=secret", "projects/example"],
      ["issue", "projects/example/issues/not-an-id", "projects/example"],
      ["plan", "projects/example/plans/0", "projects/example"],
      ["issue", "projects/example/issues/-", "projects/example"],
      ["issue", "projects/example/issues/*", "projects/example"],
      ["database", "instances/-/databases/app", "projects/example"],
      ["database", "instances/example/databases/*", "projects/example"],
      ["database", "instances/example/databases/create", "projects/example"],
      [
        "database",
        "instances/example/databases/password=secret",
        "projects/example",
      ],
      [
        "database",
        "instances/example/databases/access_token=secret",
        "projects/example",
      ],
      [
        "database",
        "instances/example/databases/postgres%3A%2F%2Fuser%3Asecret%40example.com",
        "projects/example",
      ],
      [
        "database",
        "instances/example/databases/-----BEGIN PRIVATE KEY-----secret",
        "projects/example",
      ],
      ["database", "instances/example/databases/app", "projects/-"],
      ["database", "instances/example/databases/app", "projects/*"],
      ["database", "instances/example/databases/app", "projects/not a project"],
      [
        "database",
        "instances/invalid instance/databases/app",
        "projects/example",
      ],
      ["plan", "projects/example/plans/42/tasks/1", "projects/example"],
    ].map(([kind, name, project]) => ({
      kind: "diagnostic",
      resource: { kind, name, project, label: "Example" },
    })),
  ])("rejects invalid or unbounded capability %j", (raw) => {
    expect(normalizeAgentDiagnosticCapability(raw)).toBeUndefined();
  });
});
