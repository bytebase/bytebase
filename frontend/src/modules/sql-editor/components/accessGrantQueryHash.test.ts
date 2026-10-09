import { describe, expect, test } from "vitest";
import vectors from "../../../../../testdata/access_grant_query_hash.json";
import { hashAccessGrantQuery } from "./accessGrantQueryHash";

describe("hashAccessGrantQuery", () => {
  test.each(vectors)("$name", ({ input, hash }) => {
    expect(hashAccessGrantQuery(input)).toBe(hash);
  });
});
