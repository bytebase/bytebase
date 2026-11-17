// @vitest-environment node
import { describe, expect, test } from "vitest";
import enUS from "@/locales/en-US.json";
import zhCN from "@/locales/zh-CN.json";
import {
  getQueryResponseActions as buildQueryResponseActions,
  getCurrentTurnQueryCalls,
} from "./queryFollowUps";
import {
  boundAgentQueryResult,
  omitAgentQueryRows,
} from "./queryResultRetention";
import type {
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
const getQueryResponseActions = (
  actions: AgentResponseAction[],
  messages: readonly Message[],
  references: readonly AgentResourceReference[] = []
) => buildQueryResponseActions(actions, messages, references, translate);

const database = "instances/main/databases/orders";
const project = "projects/orders";
const reference: AgentResourceReference = {
  kind: "database",
  name: database,
  project,
  label: "orders",
};
const analysis: AgentResponseAction = {
  kind: "continue",
  prompt: "How can I filter or sort these results?",
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

describe("getQueryResponseActions", () => {
  test("replaces the screenshot's invented exemption question and preserves primary actions", () => {
    expect(
      getQueryResponseActions([primary, invented], turn(...query()), [
        reference,
      ])
    ).toEqual([primary, analysis]);
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
      ).toEqual([analysis, unmask]);
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
      ).toEqual([analysis]);
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
      ).toEqual([analysis, unmask]);
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
    "removes all query follow-ups when no successful data result is established: %j",
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

  test("keeps empty data results useful and fails closed for ambiguous targets", () => {
    expect(
      getQueryResponseActions(
        [invented],
        turn(...query(envelope({ rows: [] }))),
        [reference]
      )
    ).toEqual([analysis]);
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
    ).toEqual([analysis]);
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
    ).toEqual([analysis, setup]);
    expect(getQueryResponseActions([invented], messages, [reference])).toEqual([
      analysis,
      setup,
    ]);
    expect(
      getQueryResponseActions([setup], turn(...query()), [reference])
    ).toEqual([analysis]);
  });

  test("does not require catalog-read permission for a verified explicit database", () => {
    const value = { ...overview(), catalog: { status: "unavailable" } };
    expect(
      getQueryResponseActions(
        [invented],
        turn(...query(), ...overviewCall(value)),
        [reference]
      )
    ).toEqual([analysis, setup]);
    expect(
      getQueryResponseActions(
        [invented],
        turn(...query(), ...tool("get_masking_overview", {}, value)),
        [reference]
      )
    ).toEqual([analysis, setup]);
    expect(
      getQueryResponseActions(
        [invented],
        turn(...query(), ...tool("get_masking_overview", { project }, value))
      )
    ).toEqual([analysis]);
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
      ).toEqual([analysis]);
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
      ).toEqual([analysis]);
    }
    expect(
      getQueryResponseActions(
        [setup],
        [...turn(...overviewCall()), ...turn(...query())],
        [reference]
      )
    ).toEqual([analysis]);
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
    ).toEqual([analysis, setup]);
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
      ).toEqual([analysis]);
      expect(
        getQueryResponseActions(
          [invented],
          [...messages, ...overviewCall(overview(), "refreshed")],
          [reference]
        )
      ).toEqual([analysis, setup]);
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
      ).toEqual([analysis]);
    }
  );

  test("prefers unmask over setup without requiring configuration permission", () => {
    expect(
      getQueryResponseActions(
        [setup],
        turn(...query(envelope({ masked: [{ context: "Matched rule" }] }))),
        [reference]
      )
    ).toEqual([analysis, unmask]);
  });

  test("makes discovered-database governance questions self-contained", () => {
    expect(
      getQueryResponseActions(
        [invented],
        turn(...query(envelope({ masked: [{ algorithm: "Full mask" }] })))
      )
    ).toEqual([
      analysis,
      {
        kind: "continue",
        prompt: `How can I request unmasked access to ${database}?`,
      },
    ]);
    expect(
      getQueryResponseActions([setup], turn(...query(), ...overviewCall()))
    ).toEqual([
      analysis,
      { kind: "continue", prompt: `How can I mask columns in ${database}?` },
    ]);
  });

  test("omits governance when selected scope conflicts or the identity cannot fit", () => {
    const masked = envelope({ masked: [{ algorithm: "Full mask" }] });
    expect(
      getQueryResponseActions([invented], turn(...query(masked)), [
        { ...reference, name: "instances/main/databases/other" },
      ])
    ).toEqual([analysis]);
    expect(
      getQueryResponseActions([invented], turn(...query(masked)), [
        reference,
        { ...reference, project: "projects/other" },
      ])
    ).toEqual([analysis]);
    expect(
      getQueryResponseActions(
        [invented],
        turn(
          ...query(masked, {
            name: `instances/main/databases/${"a".repeat(130)}`,
          })
        )
      )
    ).toEqual([analysis]);
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
    ).toEqual([analysis]);
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
      analysis,
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
    ).toEqual([analysis]);
    expect(
      getQueryResponseActions([invented], turn(...query(masked)), [
        { ...selectedProject, kind: "issue", name: `${project}/issues/1` },
      ])
    ).toEqual([analysis]);
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
        { kind: "continue", prompt: chinese("agent.follow-up.query-analysis") },
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
