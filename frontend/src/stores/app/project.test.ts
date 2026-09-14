// @vitest-environment node
import { describe, expect, test } from "vitest";
import { getListProjectFilter } from "./project";

const QUOTED = 'SELECT * FROM "users"';
const ESCAPED = 'SELECT * FROM \\"users\\"';

describe("getListProjectFilter", () => {
  test("escapes a quote in the free-text query", () => {
    const filter = getListProjectFilter({ query: QUOTED }).toLowerCase();
    expect(filter).not.toContain(QUOTED.toLowerCase());
    expect(filter).toContain(ESCAPED.toLowerCase());
  });
});
