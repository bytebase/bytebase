import { describe, expect, test } from "vitest";
import {
  normalizeAgentDiagnosticCapability,
  sameAgentDiagnosticCapability,
} from "./diagnosticCapability";

describe("sameAgentDiagnosticCapability", () => {
  const capability = {
    kind: "diagnostic",
    resource: {
      kind: "database",
      name: "instances/example/databases/app",
      project: "projects/example",
      label: "Application",
    },
  };

  test("matches ordinary chats without granting a diagnostic capability", () => {
    expect(sameAgentDiagnosticCapability(undefined, undefined)).toBe(true);
    expect(sameAgentDiagnosticCapability(undefined, capability)).toBe(false);
    expect(sameAgentDiagnosticCapability(capability, undefined)).toBe(false);
  });

  test("compares the normalized resource binding without its display label", () => {
    expect(
      sameAgentDiagnosticCapability(capability, {
        kind: "diagnostic",
        resource: {
          ...capability.resource,
          name: " instances/example/databases/app ",
          project: " projects/example ",
          label: "Another display label",
        },
      })
    ).toBe(true);
  });

  test.each([
    { kind: "plan", name: "projects/example/plans/42" },
    { name: "instances/example/databases/other" },
    { project: "projects/other" },
  ])("rejects a changed resource binding %j", (resource) => {
    expect(
      sameAgentDiagnosticCapability(capability, {
        kind: "diagnostic",
        resource: { ...capability.resource, ...resource },
      })
    ).toBe(false);
  });

  test.each(
    [null, {}, [], { kind: "unknown" }, { kind: "diagnostic" }].map(
      (invalid) => ({ invalid })
    )
  )("never equates malformed supplied capability $invalid", ({ invalid }) => {
    expect(sameAgentDiagnosticCapability(invalid, invalid)).toBe(false);
    expect(sameAgentDiagnosticCapability(invalid, undefined)).toBe(false);
    expect(sameAgentDiagnosticCapability(undefined, invalid)).toBe(false);
    expect(sameAgentDiagnosticCapability(capability, invalid)).toBe(false);
    expect(sameAgentDiagnosticCapability(invalid, capability)).toBe(false);
  });
});

describe("normalizeAgentDiagnosticCapability", () => {
  test.each([
    ["plan", "projects/create/plans/42", "projects/create"],
    ["issue", "projects/create/issues/42", "projects/create"],
    ["database", "instances/create/databases/app", "projects/example"],
    ["database", "instances/example/databases/create", "projects/example"],
    [
      "database",
      "projects/create/instances/create/databases/create",
      "projects/create",
    ],
  ])("accepts a valid named %s resource %s", (kind, name, project) => {
    expect(
      normalizeAgentDiagnosticCapability({
        kind: "diagnostic",
        resource: { kind, name, project, label: name },
      })
    ).toEqual({
      kind: "diagnostic",
      resource: { kind, name, project, label: name },
    });
  });

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
