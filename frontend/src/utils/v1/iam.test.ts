import { describe, expect, test } from "vitest";
import type { Binding } from "@/types/proto-es/v1/iam_policy_pb";
import { bindingScopesResources } from "./iam";

const binding = (expression?: string) =>
  ({
    condition: expression === undefined ? undefined : { expression },
  }) as Binding;

// Mirrors the server's rule: any condition attribute other than request.time
// scopes the binding away from project-wide surfaces.
describe("bindingScopesResources", () => {
  test("no condition or expiry-only conditions are project-wide", () => {
    expect(bindingScopesResources(binding())).toBe(false);
    expect(bindingScopesResources(binding(""))).toBe(false);
    expect(
      bindingScopesResources(
        binding('request.time < timestamp("2099-01-01T00:00:00Z")')
      )
    ).toBe(false);
  });

  test("resource attributes scope the binding", () => {
    expect(
      bindingScopesResources(
        binding('resource.database == "instances/i/databases/d"')
      )
    ).toBe(true);
    expect(
      bindingScopesResources(
        binding(
          'resource.environment_id == "prod" && request.time < timestamp("2099-01-01T00:00:00Z")'
        )
      )
    ).toBe(true);
    expect(
      bindingScopesResources(binding('resource.schema_name == "public"'))
    ).toBe(true);
  });
});
