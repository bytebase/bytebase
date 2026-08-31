import { describe, expect, test } from "vitest";
import { buildAccessGrantFilter } from "./accessGrant";
import { buildGroupFilter } from "./group";
import { getListInstanceFilter } from "./instance";
import { buildPlanFilter } from "./plan";
import { getListProjectFilter } from "./project";
import { buildAccountListFilter } from "./serviceAccount";
import { buildUserFilter } from "./user";
import {
  buildDatabaseFilter,
  buildProjectFilter,
  getLabelFilter,
} from "./utils";

const QUOTED = 'SELECT * FROM "users"';
const ESCAPED = 'SELECT * FROM \\"users\\"';

// A `"` typed into any of these search boxes used to close the CEL string
// literal early, and the backend rejected the whole filter with
// InvalidArgument. SQL identifiers make this routine on the access-grant page.
describe("free-text search filters escape the quote", () => {
  const cases: [string, () => string][] = [
    [
      "access grant statement",
      () => buildAccessGrantFilter({ statement: QUOTED }),
    ],
    ["group", () => buildGroupFilter({ query: QUOTED })],
    ["instance", () => getListInstanceFilter({ query: QUOTED })],
    ["plan", () => buildPlanFilter({ project: "projects/p", query: QUOTED })],
    ["project list", () => getListProjectFilter({ query: QUOTED })],
    ["project picker", () => buildProjectFilter(QUOTED)],
    ["service account", () => buildAccountListFilter({ query: QUOTED })],
    ["user", () => buildUserFilter({ query: QUOTED })],
    ["database", () => buildDatabaseFilter({ query: QUOTED })],
  ];

  test.each(cases)("%s", (_name, build) => {
    const filter = build().toLowerCase();
    expect(filter).not.toContain(QUOTED.toLowerCase());
    expect(filter).toContain(ESCAPED.toLowerCase());
  });

  test("access grant keeps the statement readable", () => {
    expect(buildAccessGrantFilter({ statement: QUOTED })).toBe(
      `query.contains("${ESCAPED}")`
    );
  });

  test("instance host and port are escaped too", () => {
    expect(getListInstanceFilter({ host: 'h"1', port: 'p"2' })).toBe(
      'host.contains("h\\"1") && port.contains("p\\"2")'
    );
  });

  test("database table filter is escaped", () => {
    expect(buildDatabaseFilter({ table: 't"1' })).toBe(
      'table.contains("t\\"1")'
    );
  });
});

describe("getLabelFilter", () => {
  // Label keys allow dashes. `labels.cost-center` parses as subtraction, so
  // the backend never saw the key and rejected the whole filter.
  test("uses index syntax so a dashed key survives CEL parsing", () => {
    expect(getLabelFilter(["cost-center:eng"])).toEqual([
      'labels["cost-center"] == "eng"',
    ]);
  });

  test("uses index syntax for multi-value keys", () => {
    expect(getLabelFilter(["cost-center:eng,ops"])).toEqual([
      'labels["cost-center"] in ["eng", "ops"]',
    ]);
  });
});
