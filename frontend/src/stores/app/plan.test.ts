// @vitest-environment node
import { describe, expect, test } from "vitest";
import { buildPlanFilter } from "./plan";

const QUOTED = 'SELECT * FROM "users"';
const ESCAPED = 'SELECT * FROM \\"users\\"';

describe("buildPlanFilter", () => {
  test("escapes a quote in the free-text query", () => {
    const filter = buildPlanFilter({
      project: "projects/p",
      query: QUOTED,
    }).toLowerCase();
    expect(filter).not.toContain(QUOTED.toLowerCase());
    expect(filter).toContain(ESCAPED.toLowerCase());
  });
});
