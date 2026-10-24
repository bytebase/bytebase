// @vitest-environment node
import { describe, expect, test } from "vitest";
import { searchApi } from "./searchApi";

describe("searchApi", () => {
  test("finds the database-listing endpoint from a natural-language query", async () => {
    const result = await searchApi({
      query: "list databases in this project",
    });

    expect(result).toMatch(
      /^## Matching endpoints\n\n1\. DatabaseService\/ListDatabases/m
    );
  });

  test("uses response-schema terms when ranking an endpoint", async () => {
    const result = await searchApi({
      query: "view project members",
    });

    expect(result).toMatch(
      /^## Matching endpoints\n\n1\. ProjectService\/GetIamPolicy/m
    );
  });

  test("shows all supported parent scopes when inspecting a listing endpoint", async () => {
    const result = await searchApi({
      operationId: "DatabaseService/ListDatabases",
    });

    expect(result).toContain("projects/{project}: list databases in a project");
    expect(result).toContain(
      "workspaces/{id}: list databases in the workspace"
    );
    expect(result).toContain(
      "instances/{instance}: list databases in an instance"
    );
    expect(result).toContain(
      "projects/{project}/instances/{instance}: list databases in a project instance"
    );
  });
});
