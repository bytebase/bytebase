// @vitest-environment node
import { describe, expect, test } from "vitest";
import { buildGroupFilter } from "./group";

const QUOTED = 'SELECT * FROM "users"';
const ESCAPED = 'SELECT * FROM \\"users\\"';

describe("buildGroupFilter", () => {
  test("escapes a quote in the free-text query", () => {
    const filter = buildGroupFilter({ query: QUOTED }).toLowerCase();
    expect(filter).not.toContain(QUOTED.toLowerCase());
    expect(filter).toContain(ESCAPED.toLowerCase());
  });
});
