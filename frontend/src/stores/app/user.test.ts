// @vitest-environment node
import { describe, expect, test } from "vitest";
import { buildUserFilter } from "./user";

const QUOTED = 'SELECT * FROM "users"';
const ESCAPED = 'SELECT * FROM \\"users\\"';

describe("buildUserFilter", () => {
  test("escapes a quote in the free-text query", () => {
    const filter = buildUserFilter({ query: QUOTED }).toLowerCase();
    expect(filter).not.toContain(QUOTED.toLowerCase());
    expect(filter).toContain(ESCAPED.toLowerCase());
  });
});
