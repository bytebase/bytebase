// @vitest-environment node
import { describe, expect, test } from "vitest";
import enUS from "@/locales/en-US.json";
import esES from "@/locales/es-ES.json";
import jaJP from "@/locales/ja-JP.json";
import viVN from "@/locales/vi-VN.json";
import zhCN from "@/locales/zh-CN.json";
import * as queryFollowUps from "./queryFollowUps";
import {
  getQueryResponseActions as buildQueryResponseActions,
  getCurrentTurnQueryCalls,
} from "./queryFollowUps";
import {
  boundAgentQueryResult,
  omitAgentQueryRows,
} from "./queryResultRetention";
import type {
  AgentQueryMaskingEligibility,
  AgentResourceReference,
  AgentResponseAction,
  Message,
} from "./types";

const translator =
  (locale: unknown) =>
  (key: string, params?: { database: string }): string => {
    let value: unknown = locale;
    for (const part of key.split("."))
      value = (value as Record<string, unknown>)[part];
    if (typeof value !== "string")
      throw new Error(`Missing locale key: ${key}`);
    return value.replace("{{database}}", params?.database ?? "");
  };
const translate = translator(enUS);
const locales = [
  { name: "en-US", locale: enUS },
  { name: "es-ES", locale: esES },
  { name: "ja-JP", locale: jaJP },
  { name: "vi-VN", locale: viVN },
  { name: "zh-CN", locale: zhCN },
];
const governanceKeys = [
  "agent.follow-up.query-mask",
  "agent.follow-up.query-unmask",
  "agent.follow-up.query-mask-database",
  "agent.follow-up.query-unmask-database",
];
const getQueryResponseActions = (
  actions: AgentResponseAction[],
  messages: readonly Message[],
  references: readonly AgentResourceReference[] = [],
  eligibility?: AgentQueryMaskingEligibility
) =>
  buildQueryResponseActions(
    actions,
    messages,
    references,
    translate,
    eligibility
  );
const getQueryMaskingSetupTarget = (
  actions: AgentResponseAction[],
  messages: readonly Message[],
  references: readonly AgentResourceReference[] = []
) => {
  expect(queryFollowUps.getQueryMaskingSetupTarget).toBeTypeOf("function");
  return queryFollowUps.getQueryMaskingSetupTarget(
    actions,
    messages,
    references
  );
};

const database = "instances/main/databases/orders";
const project = "projects/orders";
const reference: AgentResourceReference = {
  kind: "database",
  name: database,
  project,
  label: "orders",
};
const setup: AgentResponseAction = {
  kind: "continue",
  prompt: "How can I mask these columns?",
};
const unmask: AgentResponseAction = {
  kind: "continue",
  prompt: "How can I request access to unmasked data?",
};
const invented: AgentResponseAction = {
  kind: "continue",
  prompt:
    "Should I check whether your masking exemption is configured correctly or has expired?",
};
const primary: AgentResponseAction = {
  kind: "open",
  target: { kind: "open-global-masking" },
};
const rowCount: AgentResponseAction = {
  kind: "continue",
  prompt: "How many rows are in this result?",
};
const summary: AgentResponseAction = {
  kind: "continue",
  prompt: "Can you summarize these rows?",
};
const groups: AgentResponseAction = {
  kind: "continue",
  prompt: "Can you group these results by status?",
};

const tool = (
  name: string,
  args: Record<string, unknown>,
  result: unknown,
  id = name
): Message[] => [
  {
    role: "assistant",
    toolCalls: [{ id, name, arguments: JSON.stringify(args) }],
  },
  {
    role: "tool",
    toolCallId: id,
    content: typeof result === "string" ? result : JSON.stringify(result),
  },
];
const envelope = (result: Record<string, unknown> = {}) => ({
  status: 200,
  response: {
    results: [
      {
        columnNames: ["email"],
        rows: [{ values: [{ stringValue: "visible" }] }],
        statement: "SELECT email FROM users",
        ...result,
      },
    ],
  },
});
const query = (
  result: unknown = envelope(),
  body: Record<string, unknown> = {},
  id = "query",
  operationId = "SQLService/Query"
) =>
  tool(
    "call_api",
    {
      operationId,
      body: { name: database, statement: "SELECT email FROM users", ...body },
    },
    result,
    id
  );
const turn = (...messages: Message[]): Message[] => [
  { role: "user", content: "Query these records." },
  ...messages,
];
const overview = () => ({
  project,
  setup: {
    features: { masking: "available" },
    steps: [
      {
        id: "global-masking",
        readAccess: { status: "allowed" },
        configure: { status: "allowed" },
      },
      { id: "semantic-types", readAccess: { status: "allowed" } },
    ],
  },
  maskingPolicy: { status: "available", configured: false },
  semanticTypes: { status: "available", configured: false },
  catalog: { status: "available", database },
});
const overviewCall = (result: unknown = overview(), id = "overview") =>
  tool("get_masking_overview", { project, database }, result, id);

describe("getCurrentTurnQueryCalls", () => {
  test.each(["SQLService/Query", "bytebase.v1.SQLService.Query"])(
    "pairs persisted Query results for %s across an interrupted execution",
    (operationId) => {
      const messages = turn(...query(envelope(), {}, "approved", operationId));
      expect(getCurrentTurnQueryCalls(messages)).toEqual([
        {
          args: {
            operationId,
            body: { name: database, statement: "SELECT email FROM users" },
          },
          result: JSON.stringify(envelope()),
        },
      ]);
    }
  );

  test("does not reuse a previous user turn or an unpaired tool result", () => {
    const previous = query();
    expect(
      getCurrentTurnQueryCalls([
        ...turn(...previous),
        { role: "user", content: "Another question." },
        previous[1],
      ])
    ).toEqual([]);
    expect(getCurrentTurnQueryCalls(previous)).toEqual([]);
  });

  test("keeps unanswered Query attempts but ignores malformed and unrelated calls", () => {
    expect(
      getCurrentTurnQueryCalls(
        turn(
          {
            role: "assistant",
            toolCalls: [
              { id: "malformed", name: "call_api", arguments: "not JSON" },
              {
                id: "other",
                name: "call_api",
                arguments: '{"operationId":"DatabaseService/GetDatabase"}',
              },
            ],
          },
          query()[0]
        )
      )
    ).toEqual([
      {
        args: {
          operationId: "SQLService/Query",
          body: { name: database, statement: "SELECT email FROM users" },
        },
      },
    ]);
  });
});

describe("getQueryMaskingSetupTarget", () => {
  test("targets a successful unmasked Query without a model overview call", () => {
    expect(
      getQueryMaskingSetupTarget([rowCount, groups], turn(...query()), [
        reference,
      ])
    ).toBe(database);
    expect(
      getQueryMaskingSetupTarget([primary, rowCount], turn(...query()), [
        reference,
      ])
    ).toBe(database);
  });

  test("counts retained ordinary questions rather than unsupported governance wording", () => {
    expect(
      getQueryMaskingSetupTarget(
        [rowCount, invented, groups],
        turn(...query()),
        [reference]
      )
    ).toBe(database);
    expect(
      getQueryMaskingSetupTarget([invented], turn(...query()), [reference])
    ).toBe(database);
  });

  test("accepts verified project-only scope and self-contained discovered database scope", () => {
    expect(getQueryMaskingSetupTarget([rowCount], turn(...query()))).toBe(
      database
    );
    expect(
      getQueryMaskingSetupTarget([rowCount], turn(...query()), [
        {
          kind: "project",
          name: project,
          project,
          label: "Orders",
        },
      ])
    ).toBe(database);
  });

  test("accepts empty data results and BigQuery sidecar plans", () => {
    expect(
      getQueryMaskingSetupTarget(
        [rowCount],
        turn(...query(envelope({ rows: [] }))),
        [reference]
      )
    ).toBe(database);
    expect(
      getQueryMaskingSetupTarget(
        [rowCount],
        turn(
          ...query(
            envelope({
              queryPlan: {
                format: "JSON",
                executed: true,
                content: '[{"name":"stage"}]',
              },
            })
          )
        ),
        [reference]
      )
    ).toBe(database);
  });

  test.each([
    {
      name: "masked data",
      messages: turn(
        ...query(envelope({ masked: [{ algorithm: "Full mask" }] }))
      ),
    },
    {
      name: "failed Query",
      messages: turn(...query(envelope({ error: "failed" }))),
    },
    {
      name: "HTTP failure",
      messages: turn(...query({ status: 403, response: {} })),
    },
    {
      name: "EXPLAIN",
      messages: turn(...query(envelope(), { explain: true })),
    },
    {
      name: "row plan",
      messages: turn(...query(envelope({ queryPlan: { executed: true } }))),
    },
    {
      name: "metadata-only response",
      messages: turn(...query(envelope({ columnNames: [] }))),
    },
    { name: "unanswered Query", messages: turn(query()[0]) },
    { name: "no Query", messages: turn(...overviewCall()) },
    {
      name: "previous-turn Query",
      messages: [
        ...turn(...query()),
        { role: "user" as const, content: "Another request." },
      ],
    },
    {
      name: "different targets",
      messages: turn(
        ...query(),
        ...query(
          envelope(),
          { name: "instances/main/databases/other" },
          "other"
        )
      ),
    },
    {
      name: "mixed failure",
      messages: turn(
        ...query(),
        ...query(envelope({ error: "failed" }), {}, "failed")
      ),
    },
  ])("does not fetch setup eligibility for $name", ({ messages }) => {
    expect(
      getQueryMaskingSetupTarget([rowCount, groups], messages, [reference])
    ).toBeUndefined();
  });

  test("does not fetch setup eligibility after decline, a prepared workflow, or three ordinary questions", () => {
    const messages = turn(...query());
    expect(
      getQueryMaskingSetupTarget([primary], messages, [reference])
    ).toBeUndefined();
    expect(
      getQueryMaskingSetupTarget(
        [{ kind: "draft-masking-rule", draftId: "prepared" }, rowCount],
        messages,
        [reference]
      )
    ).toBeUndefined();
    expect(
      getQueryMaskingSetupTarget([rowCount, summary, groups], messages, [
        reference,
      ])
    ).toBeUndefined();
  });

  test("does not fetch setup eligibility for conflicting or unrelated references", () => {
    const messages = turn(...query());
    expect(
      getQueryMaskingSetupTarget([rowCount], messages, [
        {
          ...reference,
          name: "instances/main/databases/other",
        },
      ])
    ).toBeUndefined();
    expect(
      getQueryMaskingSetupTarget([rowCount], messages, [
        {
          ...reference,
          kind: "issue",
          name: `${project}/issues/1`,
        },
      ])
    ).toBeUndefined();
    expect(
      getQueryMaskingSetupTarget(
        [rowCount],
        turn(
          ...query(envelope(), {
            name: "projects/other/instances/main/databases/orders",
          })
        ),
        [reference]
      )
    ).toBeUndefined();
  });
});

describe("getQueryResponseActions", () => {
  test("an existing Query response offers masking setup with runtime permission evidence and no model overview", () => {
    const messages = turn(...query());
    expect(
      getQueryMaskingSetupTarget([rowCount, groups], messages, [reference])
    ).toBe(database);
    expect(
      getQueryResponseActions([rowCount, groups], messages, [reference], {
        database,
        allowed: true,
      })
    ).toEqual([rowCount, groups, setup]);
  });

  test("fresh runtime denial overrides an older positive overview", () => {
    expect(
      getQueryResponseActions(
        [rowCount, groups],
        turn(...query(), ...overviewCall()),
        [reference],
        {
          database,
          allowed: false,
        }
      )
    ).toEqual([rowCount, groups]);
  });

  test.each([
    { database: "instances/main/databases/other", allowed: true },
    { database, allowed: "true" },
    { database },
    { allowed: true },
    null,
    [],
  ])(
    "does not accept mismatched or malformed runtime permission evidence: %j",
    (eligibility) => {
      for (const messages of [
        turn(...query()),
        turn(...query(), ...overviewCall()),
      ]) {
        expect(
          getQueryResponseActions(
            [rowCount, groups],
            messages,
            [reference],
            eligibility as unknown as AgentQueryMaskingEligibility
          )
        ).toEqual([rowCount, groups]);
      }
    }
  );

  test("runtime setup eligibility does not replace positive unmask evidence", () => {
    expect(
      getQueryResponseActions(
        [rowCount, groups],
        turn(
          ...query(
            envelope({
              masked: [{ algorithm: "Full mask" }],
            })
          )
        ),
        [reference],
        { database, allowed: false }
      )
    ).toEqual([rowCount, groups, unmask]);
  });

  test("runtime eligibility respects declined follow-ups and the existing ordinary-question limit", () => {
    const declined = [primary];
    expect(
      getQueryResponseActions(declined, turn(...query()), [reference], {
        database,
        allowed: true,
      })
    ).toBe(declined);
    expect(
      getQueryResponseActions(
        [rowCount, summary, groups],
        turn(...query()),
        [reference],
        { database, allowed: true }
      )
    ).toEqual([rowCount, summary, groups]);
  });

  test("preserves ordinary screenshot questions while removing the invented exemption question", () => {
    expect(
      getQueryResponseActions([rowCount, groups, invented], turn(...query()), [
        reference,
      ])
    ).toEqual([rowCount, groups]);
  });

  test("preserves ordinary wording and order while appending one verified governance question", () => {
    expect(
      getQueryResponseActions(
        [groups, rowCount, invented],
        turn(...query(envelope({ masked: [{ algorithm: "Full mask" }] }))),
        [reference]
      )
    ).toEqual([groups, rowCount, unmask]);
    expect(
      getQueryResponseActions(
        [groups, rowCount, invented],
        turn(...query(), ...overviewCall()),
        [reference]
      )
    ).toEqual([groups, rowCount, setup]);
  });

  test("keeps three ordinary questions instead of displacing one for optional governance", () => {
    expect(
      getQueryResponseActions(
        [rowCount, summary, groups],
        turn(...query(envelope({ masked: [{ algorithm: "Full mask" }] }))),
        [reference]
      )
    ).toEqual([rowCount, summary, groups]);
  });

  test("keeps ordinary questions for failed and ambiguous Query results", () => {
    expect(
      getQueryResponseActions(
        [rowCount, groups, invented],
        turn(...query(envelope({ error: "query failed" }))),
        [reference]
      )
    ).toEqual([rowCount, groups]);
    expect(
      getQueryResponseActions(
        [rowCount, groups, invented],
        turn(
          ...query(),
          ...query(
            envelope(),
            { name: "instances/main/databases/other" },
            "other-query"
          )
        ),
        [reference]
      )
    ).toEqual([rowCount, groups]);
  });

  test("keeps ordinary questions and unmask help for a successful BigQuery data result with a sidecar plan", () => {
    expect(
      getQueryResponseActions(
        [rowCount, groups, invented],
        turn(
          ...query(
            envelope({
              masked: [{ algorithm: "Full mask" }],
              queryPlan: {
                format: "JSON",
                executed: true,
                content: '[{"name":"stage"}]',
              },
            })
          )
        ),
        [reference]
      )
    ).toEqual([rowCount, groups, unmask]);
  });

  test("keeps ordinary questions without governance for explicit EXPLAIN and row plans", () => {
    const sidecar = envelope({
      masked: [{ algorithm: "Full mask" }],
      queryPlan: {
        format: "JSON",
        executed: true,
        content: '[{"name":"stage"}]',
      },
    });
    expect(
      getQueryResponseActions(
        [rowCount, groups, invented],
        turn(...query(sidecar, { explain: true })),
        [reference]
      )
    ).toEqual([rowCount, groups]);
    expect(
      getQueryResponseActions(
        [rowCount, groups, invented],
        turn(
          ...query(envelope({ queryPlan: { format: "JSON", executed: true } }))
        ),
        [reference]
      )
    ).toEqual([rowCount, groups]);
  });

  test.each([
    "Can you mask these columns?",
    "Which masks are active?",
    "Why are these values masked?",
    "How does masking affect this query?",
    "Can you unmask these results?",
    "Who unmasks these values?",
    "Why were these values unmasked?",
    "What does unmasking require?",
    "Does my exemption apply?",
    "Are my exemptions active?",
    "Can you MASK these columns?",
    "How can I enable masking for this column?",
    "Can you apply masking to these results?",
    "How can I access unmasked data?",
    "Can you show me unmasked values?",
    "When does my exemption expire?",
    "Should I check when my exemption expires?",
    "Can I request a masking exemption?",
    "Can you check whether my masking exemption is active?",
    "Which masking rules apply to these columns?",
    "Is masking enabled for these results?",
    "Are these values masked?",
  ])("drops explicit unsupported English governance wording: %s", (prompt) => {
    expect(
      getQueryResponseActions(
        [rowCount, { kind: "continue", prompt }, groups],
        turn(...query()),
        [reference]
      )
    ).toEqual([rowCount, groups]);
  });

  test.each([
    "How can I inspect access to this project?",
    "Which permissions are needed to export these rows?",
    "When will the current session expire?",
    "Which protection settings apply to exports?",
    "How many tax exemptions are in this table?",
    "Can you count the tax exemptions by year?",
    "Can you summarize tax exemptions by category?",
    "Can you group tax exemptions by application year?",
    "Can you count tax exemptions by applicant?",
    "Can you group tax exemptions by grantor?",
    "Can you summarize the mask_column values?",
    "Can you summarize the masked_email values?",
    "Can you group unmasking_total by status?",
    "Can you summarize exemption_count for each group?",
    "Can you compare maskColumn values?",
  ])("preserves unrelated wording and SQL identifiers: %s", (prompt) => {
    const action: AgentResponseAction = { kind: "continue", prompt };
    expect(
      getQueryResponseActions([action], turn(...query()), [reference])
    ).toEqual([action]);
  });

  test.each([
    { column: "masked", prompt: "How many records have masked set to true?" },
    {
      column: "masked",
      prompt: "Can you group these rows by the masked column?",
    },
    { column: "unmasked", prompt: "Can you summarize the unmasked column?" },
    { column: "status", prompt: "Which rows have status = 'masked'?" },
    { column: "status", prompt: "How many rows have status = 'unmasked'?" },
    {
      column: "masking",
      prompt: "Can you compare the masking and exemption columns?",
    },
    {
      column: "masking",
      prompt: "Can you count records where masking = true?",
    },
    {
      column: "exemption",
      prompt: "Can you count records where exemption is null?",
    },
    {
      column: "status",
      prompt: "What is the distribution of the 'masked' status?",
    },
    {
      column: "status",
      prompt: "Can you show rows whose status is 'unmasked'?",
    },
    {
      column: "exemption",
      prompt: "How many active tax exemptions were recorded?",
    },
  ])(
    "preserves an ordinary data question about $column: $prompt",
    ({ column, prompt }) => {
      const action: AgentResponseAction = { kind: "continue", prompt };
      expect(
        getQueryResponseActions(
          [action],
          turn(
            ...query(envelope({ columnNames: [column], masked: [{}] }), {
              statement: `SELECT ${column} FROM records`,
            })
          ),
          [reference],
          { database, allowed: false }
        )
      ).toEqual([action]);
    }
  );

  test("retains positive masking evidence in the existing Query response", () => {
    const messages = turn(
      ...query(envelope({ masked: [{ algorithm: "Full mask" }] }))
    );
    expect(
      getQueryMaskingSetupTarget([rowCount, setup], messages, [reference])
    ).toBeUndefined();
    expect(
      getQueryResponseActions([rowCount, setup], messages, [reference], {
        database,
        allowed: false,
      })
    ).toEqual([rowCount, unmask]);
  });

  test("offers setup for multiple successful results in the existing Query response", () => {
    const messages = turn(
      ...query({
        status: 200,
        response: {
          results: [
            ...envelope().response.results,
            {
              columnNames: ["other"],
              rows: [],
              statement: "SELECT other FROM records",
            },
          ],
        },
      })
    );
    expect(
      getQueryMaskingSetupTarget([rowCount, setup], messages, [reference])
    ).toBe(database);
    expect(
      getQueryResponseActions([rowCount, setup], messages, [reference], {
        database,
        allowed: true,
      })
    ).toEqual([rowCount, setup]);
  });

  test.each(
    locales.flatMap(({ name, locale }) =>
      governanceKeys.map((key) => ({
        name,
        locale,
        key,
      }))
    )
  )(
    "recognizes previous $name governance template $key without current eligibility",
    ({ locale, key }) => {
      const prior = translator(locale)(key, {
        database: "instances/old/databases/archived",
      });
      expect(
        getQueryResponseActions(
          [rowCount, { kind: "continue", prompt: prior }, groups],
          turn(...query()),
          [reference]
        )
      ).toEqual([rowCount, groups]);
    }
  );

  test("does not inject a fallback when all proposed questions are unsupported", () => {
    expect(
      getQueryResponseActions([invented, setup, unmask], turn(...query()), [
        reference,
      ])
    ).toEqual([]);
  });

  test("is idempotent while preserving ordinary questions and one verified governance question", () => {
    const messages = turn(
      ...query(envelope({ masked: [{ algorithm: "Full mask" }] }))
    );
    const once = getQueryResponseActions(
      [groups, invented, rowCount],
      messages,
      [reference]
    );
    expect(once).toEqual([groups, rowCount, unmask]);
    expect(getQueryResponseActions(once, messages, [reference])).toEqual(once);
  });

  test.each(
    locales.flatMap(({ name, locale }) =>
      [false, true].map((bound) => ({ name, locale, bound }))
    )
  )(
    "strips previously eligible $name governance after a language switch and loss of evidence with bound references=$bound",
    ({ locale, bound }) => {
      const references = bound ? [reference] : [];
      for (const earlier of [
        turn(...query(), ...overviewCall()),
        turn(...query(envelope({ masked: [{ algorithm: "Full mask" }] }))),
      ]) {
        const prior = buildQueryResponseActions(
          [rowCount],
          earlier,
          references,
          translator(locale)
        );
        expect(prior).toHaveLength(2);
        expect(
          getQueryResponseActions(prior, turn(...query()), references)
        ).toEqual([rowCount]);
      }
    }
  );

  test("replaces the screenshot's invented exemption question and preserves primary actions", () => {
    expect(
      getQueryResponseActions([primary, invented], turn(...query()), [
        reference,
      ])
    ).toEqual([primary]);
  });

  test("leaves non-Query answers and declined follow-ups unchanged", () => {
    const actions = [primary, invented];
    expect(getQueryResponseActions(actions, turn(...overviewCall()))).toBe(
      actions
    );
    const declined = [primary];
    expect(getQueryResponseActions(declined, turn(...query()))).toBe(declined);
    expect(
      getQueryResponseActions(actions, [
        ...turn(...query()),
        { role: "user", content: "Explain masking." },
      ])
    ).toBe(actions);
  });

  test.each([
    { algorithm: "Full mask", context: "Multiple source columns" },
    { semanticTypeId: "private" },
    { maskingRuleId: "rule" },
    { classificationLevel: 2 },
  ])(
    "offers unmasked-access help from positive masking evidence: %j",
    (reason) => {
      expect(
        getQueryResponseActions(
          [invented],
          turn(...query(envelope({ masked: [reason] }))),
          [reference]
        )
      ).toEqual([unmask]);
    }
  );

  test.each([
    undefined,
    [],
    [{}],
    [null],
    [{ algorithm: "", semanticTypeId: "", classificationLevel: 0 }],
  ])(
    "does not infer masking from visible or mask-like values with reasons %j",
    (masked) => {
      expect(
        getQueryResponseActions(
          [invented, unmask],
          turn(
            ...query(
              envelope({
                rows: [{ values: [{ stringValue: "******" }] }],
                ...(masked === undefined ? {} : { masked }),
              })
            )
          ),
          [reference]
        )
      ).toEqual([]);
    }
  );

  test.each([boundAgentQueryResult, omitAgentQueryRows])(
    "uses masking metadata even when row previews are reduced",
    (retain) => {
      const raw = JSON.stringify(
        envelope({
          columnNames: Array.from(
            { length: 25 },
            (_, index) => `column_${index}`
          ),
          masked: Array.from({ length: 25 }, (_, index) =>
            index === 24 ? { algorithm: "Full mask" } : {}
          ),
          rows: [{ values: [{ stringValue: "x".repeat(70_000) }] }],
        })
      );
      expect(
        getQueryResponseActions([invented], turn(...query(retain(raw))), [
          reference,
        ])
      ).toEqual([unmask]);
    }
  );

  test.each([
    { status: 403, response: { results: [] } },
    { error: "Not called" },
    { ...envelope(), error: "response could not be read" },
    "not JSON",
    envelope({ error: "query failed" }),
    envelope({ permissionDenied: {} }),
    envelope({ syntaxError: {} }),
    envelope({ postgresError: {} }),
    envelope({ commandError: {} }),
    envelope({ queryPlan: { executed: true } }),
    envelope({ columnNames: [] }),
    { status: 200, response: { results: [] } },
    {
      status: 200,
      response: {
        results: [
          { columnNames: ["email"] },
          { error: "second statement failed" },
        ],
      },
    },
  ])(
    "removes unsupported governance questions when no successful data result is established: %j",
    (result) => {
      expect(
        getQueryResponseActions([primary, invented], turn(...query(result)), [
          reference,
        ])
      ).toEqual([primary]);
    }
  );

  test("excludes explicit EXPLAIN and unexecuted confirmation", () => {
    expect(
      getQueryResponseActions(
        [invented],
        turn(...query(envelope(), { explain: true })),
        [reference]
      )
    ).toEqual([]);
    expect(
      getQueryResponseActions([invented], turn(query()[0]), [reference])
    ).toEqual([]);
  });

  test("does not invent fallback questions for empty results or ambiguous targets", () => {
    expect(
      getQueryResponseActions(
        [invented],
        turn(...query(envelope({ rows: [] }))),
        [reference]
      )
    ).toEqual([]);
    expect(
      getQueryResponseActions(
        [invented],
        turn(
          ...query(),
          ...query(
            envelope(),
            { name: "instances/main/databases/other" },
            "other-query"
          )
        ),
        [reference]
      )
    ).toEqual([]);
    expect(
      getQueryResponseActions(
        [invented],
        turn(...query(), ...query(envelope(), {}, "same-target")),
        [reference]
      )
    ).toEqual([]);
  });

  test.each(["failed", "unanswered"])(
    "does not reuse an earlier successful Query after a later %s Query",
    (kind) => {
      const later = query(envelope({ error: "failed" }), {}, "later-query");
      expect(
        getQueryResponseActions(
          [invented],
          turn(
            ...query(envelope({ masked: [{ algorithm: "Full mask" }] })),
            ...(kind === "failed" ? later : [later[0]])
          ),
          [reference]
        )
      ).toEqual([]);
    }
  );

  test("offers setup only with a matching verified overview", () => {
    const messages = turn(...query(), ...overviewCall());
    expect(
      getQueryResponseActions([setup, invented], messages, [reference])
    ).toEqual([setup]);
    expect(getQueryResponseActions([invented], messages, [reference])).toEqual([
      setup,
    ]);
    expect(
      getQueryResponseActions([setup], turn(...query()), [reference])
    ).toEqual([]);
  });

  test("does not require catalog-read permission for a verified explicit database", () => {
    const value = { ...overview(), catalog: { status: "unavailable" } };
    expect(
      getQueryResponseActions(
        [invented],
        turn(...query(), ...overviewCall(value)),
        [reference]
      )
    ).toEqual([setup]);
    expect(
      getQueryResponseActions(
        [invented],
        turn(...query(), ...tool("get_masking_overview", {}, value)),
        [reference]
      )
    ).toEqual([setup]);
    expect(
      getQueryResponseActions(
        [invented],
        turn(...query(), ...tool("get_masking_overview", { project }, value))
      )
    ).toEqual([]);
  });

  test.each([
    (value: ReturnType<typeof overview>) => {
      value.setup.features.masking = "unknown";
    },
    (value: ReturnType<typeof overview>) => {
      value.setup.steps[0].configure!.status = "denied";
    },
    (value: ReturnType<typeof overview>) => {
      value.setup.steps[0].readAccess.status = "unknown";
    },
    (value: ReturnType<typeof overview>) => {
      value.setup.steps[1].readAccess.status = "denied";
    },
    (value: ReturnType<typeof overview>) => {
      value.maskingPolicy.status = "unavailable";
    },
    (value: ReturnType<typeof overview>) => {
      value.semanticTypes.status = "unavailable";
    },
    (value: ReturnType<typeof overview>) => {
      value.catalog.database = "instances/main/databases/other";
    },
    (value: ReturnType<typeof overview>) => {
      value.project = "projects/other";
    },
  ])(
    "omits setup when a required permission, facet, or scope is unverified",
    (change) => {
      const value = overview();
      change(value);
      expect(
        getQueryResponseActions(
          [setup],
          turn(...query(), ...overviewCall(value)),
          [reference]
        )
      ).toEqual([]);
    }
  );

  test("latest matching overview supersedes older permissions, including unavailable responses", () => {
    const denied = overview();
    denied.setup.steps[0].configure!.status = "denied";
    for (const latest of [denied, { error: "unavailable" }]) {
      expect(
        getQueryResponseActions(
          [setup],
          turn(
            ...query(),
            ...overviewCall(),
            ...overviewCall(latest, "latest")
          ),
          [reference]
        )
      ).toEqual([]);
    }
    expect(
      getQueryResponseActions(
        [setup],
        [...turn(...overviewCall()), ...turn(...query())],
        [reference]
      )
    ).toEqual([]);
  });

  test("an unrelated overview does not replace the latest evidence for this database", () => {
    expect(
      getQueryResponseActions(
        [invented],
        turn(
          ...query(),
          ...overviewCall(),
          ...tool(
            "get_masking_overview",
            { project, database: "instances/main/databases/other" },
            { error: "unavailable" },
            "other-overview"
          )
        ),
        [reference]
      )
    ).toEqual([setup]);
  });

  test.each([
    { name: "call_api", args: { operationId: "OrgPolicyService/SetPolicy" } },
    { name: "call_api", args: { operationId: "DatabaseService/GetDatabase" } },
    { name: "dom_action", args: { type: "click", ref: "save" } },
  ])(
    "requires a fresh overview after a potentially invalidating tool: $name $args",
    ({ name, args }) => {
      const mutation = tool(name, args, { status: 200 }, "mutation");
      const messages = turn(...query(), ...overviewCall(), ...mutation);
      expect(
        getQueryResponseActions([invented], messages, [reference])
      ).toEqual([]);
      expect(
        getQueryResponseActions(
          [invented],
          [...messages, ...overviewCall(overview(), "refreshed")],
          [reference]
        )
      ).toEqual([setup]);
    }
  );

  test.each([
    { ...overview(), databaseSelection: { status: "unavailable" } },
    { ...overview(), setup: { ...overview().setup, project: null } },
    { ...overview(), setup: { ...overview().setup, project } },
    {
      ...overview(),
      setup: { ...overview().setup, project: { name: "projects/other" } },
    },
  ])(
    "omits setup with explicitly unavailable selection or malformed project evidence: %j",
    (value) => {
      expect(
        getQueryResponseActions(
          [invented],
          turn(...query(), ...overviewCall(value)),
          [reference]
        )
      ).toEqual([]);
    }
  );

  test("prefers unmask over setup without requiring configuration permission", () => {
    expect(
      getQueryResponseActions(
        [setup],
        turn(...query(envelope({ masked: [{ context: "Matched rule" }] }))),
        [reference]
      )
    ).toEqual([unmask]);
  });

  test("makes discovered-database governance questions self-contained", () => {
    expect(
      getQueryResponseActions(
        [invented],
        turn(...query(envelope({ masked: [{ algorithm: "Full mask" }] })))
      )
    ).toEqual([
      {
        kind: "continue",
        prompt: `How can I request unmasked access to ${database}?`,
      },
    ]);
    expect(
      getQueryResponseActions([setup], turn(...query(), ...overviewCall()))
    ).toEqual([
      { kind: "continue", prompt: `How can I mask columns in ${database}?` },
    ]);
  });

  test("omits governance when selected scope conflicts or the identity cannot fit", () => {
    const masked = envelope({ masked: [{ algorithm: "Full mask" }] });
    expect(
      getQueryResponseActions([invented], turn(...query(masked)), [
        { ...reference, name: "instances/main/databases/other" },
      ])
    ).toEqual([]);
    expect(
      getQueryResponseActions([invented], turn(...query(masked)), [
        reference,
        { ...reference, project: "projects/other" },
      ])
    ).toEqual([]);
    expect(
      getQueryResponseActions(
        [invented],
        turn(
          ...query(masked, {
            name: `instances/main/databases/${"a".repeat(130)}`,
          })
        )
      )
    ).toEqual([]);
    expect(
      getQueryResponseActions(
        [invented],
        turn(
          ...query(masked, {
            name: "projects/other/instances/main/databases/orders",
          })
        ),
        [
          {
            ...reference,
            name: "projects/other/instances/main/databases/orders",
          },
        ]
      )
    ).toEqual([]);
  });

  test("uses a self-contained database question for project-only references", () => {
    const selectedProject: AgentResourceReference = {
      kind: "project",
      name: project,
      project,
      label: "Orders",
    };
    const masked = envelope({ masked: [{ algorithm: "Full mask" }] });
    expect(
      getQueryResponseActions([invented], turn(...query(masked)), [
        selectedProject,
      ])
    ).toEqual([
      {
        kind: "continue",
        prompt: `How can I request unmasked access to ${database}?`,
      },
    ]);
    expect(
      getQueryResponseActions(
        [invented],
        turn(
          ...query(masked, {
            name: "projects/other/instances/main/databases/orders",
          })
        ),
        [selectedProject]
      )
    ).toEqual([]);
    expect(
      getQueryResponseActions([invented], turn(...query(masked)), [
        { ...selectedProject, kind: "issue", name: `${project}/issues/1` },
      ])
    ).toEqual([]);
  });

  test("leaves later prepared-draft workflow answers unchanged", () => {
    const draft: AgentResponseAction = {
      kind: "draft-masking-rule",
      draftId: "prepared",
    };
    const actions = [draft, invented];
    expect(
      getQueryResponseActions(
        actions,
        turn(...query(envelope({ masked: [{ algorithm: "Full mask" }] }))),
        [reference]
      )
    ).toBe(actions);
  });

  test.each([false, true])(
    "is idempotent for runtime-owned governance with bound references=%s",
    (bound) => {
      const references = bound ? [reference] : [];
      for (const messages of [
        turn(...query(), ...overviewCall()),
        turn(...query(envelope({ masked: [{ algorithm: "Full mask" }] }))),
      ]) {
        const actions = getQueryResponseActions([setup], messages, references);
        expect(getQueryResponseActions(actions, messages, references)).toEqual(
          actions
        );
      }
    }
  );

  test.each([false, true])(
    "rebuilds setup after a language switch with bound references=%s",
    (bound) => {
      const references = bound ? [reference] : [];
      const messages = turn(...query(), ...overviewCall());
      const english = getQueryResponseActions([invented], messages, references);
      const chinese = translator(zhCN);
      expect(
        buildQueryResponseActions(english, messages, references, chinese)
      ).toEqual([
        {
          kind: "continue",
          prompt: chinese(
            `agent.follow-up.query-mask${bound ? "" : "-database"}`,
            { database }
          ),
        },
      ]);
    }
  );
});
