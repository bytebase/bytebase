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
});
