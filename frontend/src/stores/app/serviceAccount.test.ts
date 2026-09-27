// @vitest-environment node
import { describe, expect, test } from "vitest";
import { buildAccountListFilter } from "./serviceAccount";

const QUOTED = 'SELECT * FROM "users"';
const ESCAPED = 'SELECT * FROM \\"users\\"';

describe("buildAccountListFilter", () => {
  test("escapes a quote in the free-text query", () => {
    const filter = buildAccountListFilter({ query: QUOTED }).toLowerCase();
    expect(filter).not.toContain(QUOTED.toLowerCase());
    expect(filter).toContain(ESCAPED.toLowerCase());
  });
});
