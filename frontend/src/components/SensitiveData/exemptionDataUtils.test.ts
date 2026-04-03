import { describe, expect, test, vi } from "vitest";

vi.mock("@/plugins/i18n", () => ({
  locale: { value: "en-US" },
  t: (key: string, params?: Record<string, unknown>) => {
    const map: Record<string, string> = {
      "database.all": "All databases",
      "common.schema": "Schema",
      "common.table": "Table",
      "project.masking-exemption.level": "level",
    };
    if (key === "common.n-more" && params?.n !== undefined)
      return `+${params.n} more`;
    return map[key] ?? key;
  },
}));

import {
  generateGrantTitle,
  getConditionExpression,
  groupByMember,
  parseClassificationLevel,
  parseExpirationTimestamp,
  rewriteResourceDatabase,
} from "./exemptionDataUtils";
import type { AccessUser, ExemptionGrant } from "./types";

describe("parseClassificationLevel", () => {
  test("parses <= operator", () => {
    const result = parseClassificationLevel(
      "resource.classification_level <= 3"
    );
    expect(result).toEqual({ operator: "<=", value: 3 });
  });

  test("parses == operator", () => {
    const result = parseClassificationLevel(
      "resource.classification_level == 2"
    );
    expect(result).toEqual({ operator: "==", value: 2 });
  });

  test("parses < operator", () => {
    const result = parseClassificationLevel(
      "resource.classification_level < 4"
    );
    expect(result).toEqual({ operator: "<", value: 4 });
  });

  test("parses >= operator", () => {
    const result = parseClassificationLevel(
      "resource.classification_level >= 1"
    );
    expect(result).toEqual({ operator: ">=", value: 1 });
  });

  test("parses > operator", () => {
    const result = parseClassificationLevel(
      "resource.classification_level > 0"
    );
    expect(result).toEqual({ operator: ">", value: 0 });
  });

  test("parses != operator", () => {
    const result = parseClassificationLevel(
      "resource.classification_level != 5"
    );
    expect(result).toEqual({ operator: "!=", value: 5 });
  });

  test("handles extra whitespace around operator", () => {
    const result = parseClassificationLevel(
      "resource.classification_level  <=  3"
    );
    expect(result).toEqual({ operator: "<=", value: 3 });
  });

  test("extracts level from a combined CEL expression", () => {
    const result = parseClassificationLevel(
      'resource.instance_id == "prod" && resource.classification_level <= 3'
    );
    expect(result).toEqual({ operator: "<=", value: 3 });
  });

  test("returns undefined when no classification level present", () => {
    const result = parseClassificationLevel(
      'resource.instance_id == "prod" && resource.database_name == "hr"'
    );
    expect(result).toBeUndefined();
  });

  test("returns undefined for empty string", () => {
    expect(parseClassificationLevel("")).toBeUndefined();
  });

  test("handles multi-digit level values", () => {
    const result = parseClassificationLevel(
      "resource.classification_level <= 10"
    );
    expect(result).toEqual({ operator: "<=", value: 10 });
  });
});

describe("getConditionExpression", () => {
  test("strips request.time from expression", () => {
    const result = getConditionExpression(
      'resource.instance_id == "prod" && request.time < timestamp("2026-04-15T00:00:00Z")'
    );
    expect(result).toBe('resource.instance_id == "prod"');
  });

  test("returns full expression when no request.time", () => {
    const result = getConditionExpression(
      'resource.instance_id == "prod" && resource.database_name == "hr"'
    );
    expect(result).toBe(
      'resource.instance_id == "prod" && resource.database_name == "hr"'
    );
  });

  test("returns empty string for empty input", () => {
    expect(getConditionExpression("")).toBe("");
  });

  test("returns empty string when expression is only request.time", () => {
    const result = getConditionExpression(
      'request.time < timestamp("2026-04-15T00:00:00Z")'
    );
    expect(result).toBe("");
  });

  test("handles extra whitespace in request.time", () => {
    const result = getConditionExpression(
      'resource.instance_id == "prod" && request.time  <  timestamp("2026-04-15T00:00:00Z")'
    );
    expect(result).toBe('resource.instance_id == "prod"');
  });
});

describe("parseExpirationTimestamp", () => {
  test("extracts timestamp from request.time expression", () => {
    const result = parseExpirationTimestamp(
      'resource.instance_id == "prod" && request.time < timestamp("2026-04-15T11:30:00.000Z")'
    );
    expect(result).toBe(new Date("2026-04-15T11:30:00.000Z").getTime());
  });

  test("returns undefined when no request.time present", () => {
    const result = parseExpirationTimestamp('resource.instance_id == "prod"');
    expect(result).toBeUndefined();
  });

  test("returns undefined for empty string", () => {
    expect(parseExpirationTimestamp("")).toBeUndefined();
  });

  test("extracts timestamp when request.time is the only expression", () => {
    const result = parseExpirationTimestamp(
      'request.time < timestamp("2026-12-31T23:59:59.000Z")'
    );
    expect(result).toBe(new Date("2026-12-31T23:59:59.000Z").getTime());
  });
});

describe("groupByMember", () => {
  const makeAccessUser = (overrides: Partial<AccessUser>): AccessUser => ({
    type: "user",
    key: "default-key",
    member: "user:test@bytebase.com",
    rawExpression: "",
    description: "",
    conditionExpression: "",
    ...overrides,
  });

  test("groups multiple exemptions for the same member", () => {
    const users: AccessUser[] = [
      makeAccessUser({
        member: "user:admin@bytebase.com",
        key: "admin:1",
        description: "Grant 1",
        conditionExpression: 'resource.instance_id == "prod"',
      }),
      makeAccessUser({
        member: "user:admin@bytebase.com",
        key: "admin:2",
        description: "Grant 2",
        conditionExpression: "resource.classification_level <= 3",
      }),
    ];

    const result = groupByMember(users);
    expect(result).toHaveLength(1);
    expect(result[0].member).toBe("user:admin@bytebase.com");
    expect(result[0].grants).toHaveLength(2);
    // Latest created grant first (reversed order)
    expect(result[0].grants[0].description).toBe("Grant 2");
    expect(result[0].grants[1].description).toBe("Grant 1");
  });

  test("separates different members", () => {
    const users: AccessUser[] = [
      makeAccessUser({ member: "user:alice@bytebase.com", key: "a:1" }),
      makeAccessUser({ member: "user:bob@bytebase.com", key: "b:1" }),
    ];

    const result = groupByMember(users);
    expect(result).toHaveLength(2);
    expect(
      result.map((m) => m.member).sort((a, b) => a.localeCompare(b))
    ).toEqual(["user:alice@bytebase.com", "user:bob@bytebase.com"]);
  });

  test("detects group type from member prefix", () => {
    const users: AccessUser[] = [
      makeAccessUser({
        type: "group",
        member: "group:analysts@bytebase.com",
        key: "g:1",
      }),
    ];

    const result = groupByMember(users);
    expect(result[0].type).toBe("group");
  });

  test("computes neverExpires=true when any grant has no expiration", () => {
    const users: AccessUser[] = [
      makeAccessUser({
        member: "user:admin@bytebase.com",
        key: "a:1",
        expirationTimestamp: new Date("2026-04-15").getTime(),
      }),
      makeAccessUser({
        member: "user:admin@bytebase.com",
        key: "a:2",
        expirationTimestamp: undefined,
      }),
    ];

    const result = groupByMember(users);
    expect(result[0].neverExpires).toBe(true);
  });

  test("computes neverExpires=false when all grants have expiration", () => {
    const users: AccessUser[] = [
      makeAccessUser({
        member: "user:admin@bytebase.com",
        key: "a:1",
        expirationTimestamp: new Date("2026-04-15").getTime(),
      }),
      makeAccessUser({
        member: "user:admin@bytebase.com",
        key: "a:2",
        expirationTimestamp: new Date("2026-06-30").getTime(),
      }),
    ];

    const result = groupByMember(users);
    expect(result[0].neverExpires).toBe(false);
  });

  test("computes nearestExpiration as earliest timestamp", () => {
    const apr = new Date("2026-04-15").getTime();
    const jun = new Date("2026-06-30").getTime();
    const users: AccessUser[] = [
      makeAccessUser({
        member: "user:admin@bytebase.com",
        key: "a:1",
        expirationTimestamp: jun,
      }),
      makeAccessUser({
        member: "user:admin@bytebase.com",
        key: "a:2",
        expirationTimestamp: apr,
      }),
    ];

    const result = groupByMember(users);
    expect(result[0].nearestExpiration).toBe(apr);
  });

  test("nearestExpiration is undefined when no grants have expiration", () => {
    const users: AccessUser[] = [
      makeAccessUser({
        member: "user:admin@bytebase.com",
        key: "a:1",
      }),
    ];

    const result = groupByMember(users);
    expect(result[0].nearestExpiration).toBeUndefined();
  });

  test("databaseNames has empty sentinel when no grants have databaseResources", () => {
    const users: AccessUser[] = [
      makeAccessUser({
        member: "user:admin@bytebase.com",
        key: "a:1",
        conditionExpression: "resource.classification_level <= 3",
      }),
    ];

    const result = groupByMember(users);
    // Empty sentinel indicates "all databases" (grant has no specific resources)
    expect(result[0].databaseNames).toEqual([""]);
  });

  test("extracts unique database names from resources", () => {
    const users: AccessUser[] = [
      makeAccessUser({
        member: "user:admin@bytebase.com",
        key: "a:1",
        databaseResources: [
          {
            databaseFullName: "instances/prod/databases/hr_prod",
            schema: "public",
            table: "employee",
          },
          {
            databaseFullName: "instances/prod/databases/hr_prod",
            schema: "public",
            table: "audit",
          },
          {
            databaseFullName: "instances/dev/databases/test",
            schema: "",
            table: "",
          },
        ],
      }),
    ];

    const result = groupByMember(users);
    expect(result[0].databaseNames.sort((a, b) => a.localeCompare(b))).toEqual([
      "hr_prod",
      "test",
    ]);
  });

  test("parses classification level into grant", () => {
    const users: AccessUser[] = [
      makeAccessUser({
        member: "user:admin@bytebase.com",
        key: "a:1",
        conditionExpression:
          'resource.instance_id == "prod" && resource.classification_level <= 3',
      }),
    ];

    const result = groupByMember(users);
    expect(result[0].grants[0].classificationLevel).toEqual({
      operator: "<=",
      value: 3,
    });
  });

  test("classificationLevel is undefined when not in expression", () => {
    const users: AccessUser[] = [
      makeAccessUser({
        member: "user:admin@bytebase.com",
        key: "a:1",
        conditionExpression: 'resource.instance_id == "prod"',
      }),
    ];

    const result = groupByMember(users);
    expect(result[0].grants[0].classificationLevel).toBeUndefined();
  });

  test("handles empty input array", () => {
    expect(groupByMember([])).toEqual([]);
  });

  test("does not merge grants — each exemption is its own card", () => {
    const exp = new Date("2026-04-15").getTime();
    const users: AccessUser[] = [
      makeAccessUser({
        member: "user:admin@bytebase.com",
        key: "a:1",
        description: "Production audit",
        expirationTimestamp: exp,
        databaseResources: [
          {
            databaseFullName: "instances/prod/databases/hr_prod",
            schema: "public",
            table: "audit",
          },
        ],
      }),
      makeAccessUser({
        member: "user:admin@bytebase.com",
        key: "a:2",
        description: "Production audit",
        expirationTimestamp: exp,
        databaseResources: [
          {
            databaseFullName: "instances/prod/databases/hr_prod",
            schema: "public",
            table: "department",
          },
        ],
      }),
      makeAccessUser({
        member: "user:admin@bytebase.com",
        key: "a:3",
        description: "Production audit",
        expirationTimestamp: exp,
        databaseResources: [
          {
            databaseFullName: "instances/prod/databases/hr_prod",
            schema: "public",
            table: "employee",
          },
        ],
      }),
    ];

    const result = groupByMember(users);
    expect(result).toHaveLength(1);
    // No merging — each exemption is its own grant card
    expect(result[0].grants).toHaveLength(3);
  });

  test("keeps overlapping resources as separate grants", () => {
    const users: AccessUser[] = [
      makeAccessUser({
        member: "user:admin@bytebase.com",
        key: "a:1",
        description: "",
        databaseResources: [
          {
            databaseFullName: "instances/prod/databases/hr_prod",
            schema: "public",
            table: "employee",
          },
          {
            databaseFullName: "instances/prod/databases/hr_prod",
            schema: "public",
            table: "audit",
          },
        ],
      }),
      makeAccessUser({
        member: "user:admin@bytebase.com",
        key: "a:2",
        description: "",
        databaseResources: [
          {
            databaseFullName: "instances/prod/databases/hr_prod",
            schema: "public",
            table: "employee",
          },
          {
            databaseFullName: "instances/prod/databases/hr_prod",
            schema: "public",
            table: "department",
          },
        ],
      }),
    ];

    const result = groupByMember(users);
    // No merging — each exemption keeps its own resources
    expect(result[0].grants).toHaveLength(2);
    expect(result[0].grants[0].databaseResources).toHaveLength(2);
    expect(result[0].grants[1].databaseResources).toHaveLength(2);
  });

  test("keeps grants with different descriptions as separate cards", () => {
    const users: AccessUser[] = [
      makeAccessUser({
        member: "user:admin@bytebase.com",
        key: "a:1",
        description: "Audit access",
        databaseResources: [
          {
            databaseFullName: "instances/prod/databases/hr_prod",
            schema: "public",
            table: "audit",
          },
        ],
      }),
      makeAccessUser({
        member: "user:admin@bytebase.com",
        key: "a:2",
        description: "Dev access",
        databaseResources: [
          {
            databaseFullName: "instances/prod/databases/hr_prod",
            schema: "public",
            table: "employee",
          },
        ],
      }),
    ];

    const result = groupByMember(users);
    expect(result[0].grants).toHaveLength(2);
  });

  test("does NOT merge grants with different expirations", () => {
    const users: AccessUser[] = [
      makeAccessUser({
        member: "user:admin@bytebase.com",
        key: "a:1",
        description: "Same reason",
        expirationTimestamp: new Date("2026-04-15").getTime(),
        databaseResources: [
          {
            databaseFullName: "instances/prod/databases/hr_prod",
            schema: "public",
            table: "audit",
          },
        ],
      }),
      makeAccessUser({
        member: "user:admin@bytebase.com",
        key: "a:2",
        description: "Same reason",
        expirationTimestamp: new Date("2026-06-30").getTime(),
        databaseResources: [
          {
            databaseFullName: "instances/prod/databases/hr_prod",
            schema: "public",
            table: "employee",
          },
        ],
      }),
    ];

    const result = groupByMember(users);
    expect(result[0].grants).toHaveLength(2);
  });

  test("does NOT merge grants with different classification levels", () => {
    const users: AccessUser[] = [
      makeAccessUser({
        member: "user:admin@bytebase.com",
        key: "a:1",
        description: "",
        conditionExpression:
          'resource.instance_id == "prod" && resource.classification_level <= 3',
        databaseResources: [
          {
            databaseFullName: "instances/prod/databases/hr_prod",
            schema: "public",
            table: "audit",
          },
        ],
      }),
      makeAccessUser({
        member: "user:admin@bytebase.com",
        key: "a:2",
        description: "",
        conditionExpression:
          'resource.instance_id == "prod" && resource.classification_level <= 5',
        databaseResources: [
          {
            databaseFullName: "instances/prod/databases/hr_prod",
            schema: "public",
            table: "employee",
          },
        ],
      }),
    ];

    const result = groupByMember(users);
    expect(result[0].grants).toHaveLength(2);
  });

  test("keeps level-only grants as separate cards even with same description", () => {
    const users: AccessUser[] = [
      makeAccessUser({
        member: "user:admin@bytebase.com",
        key: "a:1",
        description: "Level access",
        conditionExpression: "resource.classification_level <= 3",
      }),
      makeAccessUser({
        member: "user:admin@bytebase.com",
        key: "a:2",
        description: "Level access",
        conditionExpression: "resource.classification_level <= 3",
      }),
    ];

    const result = groupByMember(users);
    // No merging — each exemption is its own card
    expect(result[0].grants).toHaveLength(2);
  });
});

describe("generateGrantTitle", () => {
  const makeGrant = (overrides: Partial<ExemptionGrant>): ExemptionGrant => ({
    id: "test",
    description: "",
    rawExpression: "",
    conditionExpression: "",
    ...overrides,
  });

  // --- No databases ---

  test("no resources, no level → All databases", () => {
    expect(generateGrantTitle(makeGrant({}))).toBe("All databases");
  });

  test("no resources, with level", () => {
    expect(
      generateGrantTitle(
        makeGrant({ classificationLevel: { operator: "<=", value: 3 } })
      )
    ).toBe("All databases, level ≤ 3");
  });

  // --- Multiple databases ---

  test("2 databases", () => {
    expect(
      generateGrantTitle(
        makeGrant({
          databaseResources: [
            {
              databaseFullName: "instances/a/databases/bbdev",
              schema: "",
              table: "",
            },
            {
              databaseFullName: "instances/a/databases/hr_prod",
              schema: "",
              table: "",
            },
          ],
        })
      )
    ).toBe("bbdev, hr_prod");
  });

  test("3 databases", () => {
    expect(
      generateGrantTitle(
        makeGrant({
          databaseResources: [
            {
              databaseFullName: "instances/a/databases/bbdev",
              schema: "",
              table: "",
            },
            {
              databaseFullName: "instances/a/databases/hr_prod",
              schema: "",
              table: "",
            },
            {
              databaseFullName: "instances/a/databases/test",
              schema: "",
              table: "",
            },
          ],
        })
      )
    ).toBe("bbdev, hr_prod +1 more");
  });

  test("2 databases with level", () => {
    expect(
      generateGrantTitle(
        makeGrant({
          databaseResources: [
            {
              databaseFullName: "instances/a/databases/bbdev",
              schema: "",
              table: "",
            },
            {
              databaseFullName: "instances/a/databases/hr_prod",
              schema: "",
              table: "",
            },
          ],
          classificationLevel: { operator: "<=", value: 3 },
        })
      )
    ).toBe("bbdev, hr_prod, level ≤ 3");
  });

  // --- Single database, single path (drill down) ---

  test("1 db, no schema, no table", () => {
    expect(
      generateGrantTitle(
        makeGrant({
          databaseResources: [
            {
              databaseFullName: "instances/a/databases/hr_prod",
              schema: "",
              table: "",
            },
          ],
        })
      )
    ).toBe("hr_prod");
  });

  test("1 db, 1 schema, no table → show schema keyword", () => {
    expect(
      generateGrantTitle(
        makeGrant({
          databaseResources: [
            {
              databaseFullName: "instances/a/databases/hr_prod",
              schema: "public",
              table: "",
            },
          ],
        })
      )
    ).toBe("hr_prod (schema public)");
  });

  test("1 db, 1 schema, 1 table → dot notation", () => {
    expect(
      generateGrantTitle(
        makeGrant({
          databaseResources: [
            {
              databaseFullName: "instances/a/databases/hr_prod",
              schema: "public",
              table: "audit",
            },
          ],
        })
      )
    ).toBe("hr_prod (public.audit)");
  });

  test("1 db, 1 schema, 1 table, 1 column → full dot path", () => {
    expect(
      generateGrantTitle(
        makeGrant({
          databaseResources: [
            {
              databaseFullName: "instances/a/databases/hr_prod",
              schema: "public",
              table: "employee",
              columns: ["emp_no"],
            },
          ],
        })
      )
    ).toBe("hr_prod (public.employee.emp_no)");
  });

  test("1 db, 1 schema, 1 table, 2 columns → colon then list", () => {
    expect(
      generateGrantTitle(
        makeGrant({
          databaseResources: [
            {
              databaseFullName: "instances/a/databases/hr_prod",
              schema: "public",
              table: "employee",
              columns: ["emp_no", "name"],
            },
          ],
        })
      )
    ).toBe("hr_prod (public.employee: emp_no, name)");
  });

  test("1 db, 1 schema, 1 table, 3+ columns → colon then 2 + more", () => {
    expect(
      generateGrantTitle(
        makeGrant({
          databaseResources: [
            {
              databaseFullName: "instances/a/databases/hr_prod",
              schema: "public",
              table: "employee",
              columns: ["emp_no", "name", "salary", "dept"],
            },
          ],
        })
      )
    ).toBe("hr_prod (public.employee: emp_no, name +2 more)");
  });

  test("1 db, no schema, 1 table → table keyword", () => {
    expect(
      generateGrantTitle(
        makeGrant({
          databaseResources: [
            {
              databaseFullName: "instances/a/databases/hr_prod",
              schema: "",
              table: "employee",
            },
          ],
        })
      )
    ).toBe("hr_prod (table employee)");
  });

  test("1 db, no schema, 1 table, 1 column → table.column", () => {
    expect(
      generateGrantTitle(
        makeGrant({
          databaseResources: [
            {
              databaseFullName: "instances/a/databases/hr_prod",
              schema: "",
              table: "employee",
              columns: ["emp_no"],
            },
          ],
        })
      )
    ).toBe("hr_prod (employee.emp_no)");
  });

  // --- Single database, branching (multiple at some level) ---

  test("1 db, 2 schemas → comma list, stop", () => {
    expect(
      generateGrantTitle(
        makeGrant({
          databaseResources: [
            {
              databaseFullName: "instances/a/databases/hr_prod",
              schema: "public",
              table: "",
            },
            {
              databaseFullName: "instances/a/databases/hr_prod",
              schema: "test",
              table: "",
            },
          ],
        })
      )
    ).toBe("hr_prod (public, test)");
  });

  test("1 db, 3 schemas → 2 + more", () => {
    expect(
      generateGrantTitle(
        makeGrant({
          databaseResources: [
            {
              databaseFullName: "instances/a/databases/hr_prod",
              schema: "public",
              table: "",
            },
            {
              databaseFullName: "instances/a/databases/hr_prod",
              schema: "test",
              table: "",
            },
            {
              databaseFullName: "instances/a/databases/hr_prod",
              schema: "audit",
              table: "",
            },
          ],
        })
      )
    ).toBe("hr_prod (public, test +1 more)");
  });

  test("1 db, 1 schema, 2 tables → colon then list", () => {
    expect(
      generateGrantTitle(
        makeGrant({
          databaseResources: [
            {
              databaseFullName: "instances/a/databases/hr_prod",
              schema: "public",
              table: "audit",
            },
            {
              databaseFullName: "instances/a/databases/hr_prod",
              schema: "public",
              table: "dept",
            },
          ],
        })
      )
    ).toBe("hr_prod (public: audit, dept)");
  });

  test("1 db, 1 schema, 3 tables → colon then 2 + more", () => {
    expect(
      generateGrantTitle(
        makeGrant({
          databaseResources: [
            {
              databaseFullName: "instances/a/databases/hr_prod",
              schema: "public",
              table: "audit",
            },
            {
              databaseFullName: "instances/a/databases/hr_prod",
              schema: "public",
              table: "dept",
            },
            {
              databaseFullName: "instances/a/databases/hr_prod",
              schema: "public",
              table: "employee",
            },
          ],
        })
      )
    ).toBe("hr_prod (public: audit, dept +1 more)");
  });

  test("1 db, no schema, 2 tables → comma list", () => {
    expect(
      generateGrantTitle(
        makeGrant({
          databaseResources: [
            {
              databaseFullName: "instances/a/databases/hr_prod",
              schema: "",
              table: "audit",
            },
            {
              databaseFullName: "instances/a/databases/hr_prod",
              schema: "",
              table: "dept",
            },
          ],
        })
      )
    ).toBe("hr_prod (audit, dept)");
  });

  test("1 db, no schema, 4 tables → 2 + more", () => {
    expect(
      generateGrantTitle(
        makeGrant({
          databaseResources: [
            {
              databaseFullName: "instances/a/databases/hr_prod",
              schema: "",
              table: "a",
            },
            {
              databaseFullName: "instances/a/databases/hr_prod",
              schema: "",
              table: "b",
            },
            {
              databaseFullName: "instances/a/databases/hr_prod",
              schema: "",
              table: "c",
            },
            {
              databaseFullName: "instances/a/databases/hr_prod",
              schema: "",
              table: "d",
            },
          ],
        })
      )
    ).toBe("hr_prod (a, b +2 more)");
  });

  // --- With level suffix ---

  test("single path with level", () => {
    expect(
      generateGrantTitle(
        makeGrant({
          databaseResources: [
            {
              databaseFullName: "instances/a/databases/hr_prod",
              schema: "public",
              table: "audit",
            },
          ],
          classificationLevel: { operator: "<=", value: 3 },
        })
      )
    ).toBe("hr_prod (public.audit), level ≤ 3");
  });

  test("branching with level", () => {
    expect(
      generateGrantTitle(
        makeGrant({
          databaseResources: [
            {
              databaseFullName: "instances/a/databases/hr_prod",
              schema: "public",
              table: "audit",
            },
            {
              databaseFullName: "instances/a/databases/hr_prod",
              schema: "public",
              table: "dept",
            },
          ],
          classificationLevel: { operator: "==", value: 1 },
        })
      )
    ).toBe("hr_prod (public: audit, dept), level = 1");
  });
});

describe("rewriteResourceDatabase", () => {
  test("rewrites single resource.database condition", () => {
    const input =
      'resource.database == "instances/prod-instance/databases/hr_prod"';
    expect(rewriteResourceDatabase(input)).toBe(
      'resource.instance_id == "prod-instance" && resource.database_name == "hr_prod"'
    );
  });

  test("rewrites multiple resource.database conditions", () => {
    const input =
      '(resource.database == "instances/a/databases/db1") || (resource.database == "instances/b/databases/db2")';
    expect(rewriteResourceDatabase(input)).toBe(
      '(resource.instance_id == "a" && resource.database_name == "db1") || (resource.instance_id == "b" && resource.database_name == "db2")'
    );
  });

  test("preserves expressions without resource.database", () => {
    const input = "resource.classification_level <= 3";
    expect(rewriteResourceDatabase(input)).toBe(input);
  });

  test("handles mixed conditions", () => {
    const input =
      'resource.database == "instances/prod/databases/users" && resource.classification_level <= 2';
    expect(rewriteResourceDatabase(input)).toBe(
      'resource.instance_id == "prod" && resource.database_name == "users" && resource.classification_level <= 2'
    );
  });

  test("handles empty string", () => {
    expect(rewriteResourceDatabase("")).toBe("");
  });
});
