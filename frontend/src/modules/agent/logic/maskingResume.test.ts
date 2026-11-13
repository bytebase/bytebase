import { describe, expect, test } from "vitest";
import { getMaskingResumeArgs } from "./maskingResume";
import type { Message } from "./types";

const maskingTurn: Message[] = [
  { role: "user", content: "Help me mask email addresses." },
  {
    role: "assistant",
    toolCalls: [
      {
        id: "overview",
        name: "get_masking_overview",
        arguments: JSON.stringify({ project: "projects/orders" }),
      },
    ],
  },
  {
    role: "tool",
    toolCallId: "overview",
    content: '{"openTargets":[{"kind":"open-global-masking"}]}',
  },
  {
    role: "assistant",
    toolCalls: [
      {
        id: "question",
        name: "ask_user",
        arguments: '{"prompt":"Which visibility?"}',
      },
    ],
  },
  {
    role: "tool",
    toolCallId: "question",
    content: '{"kind":"choose","answer":"Fully hidden","value":"full"}',
  },
];

const exemptionScopeTurn = (
  answer: Record<string, unknown>,
  {
    overviewArgs = {
      project: "projects/orders",
      database: "instances/prod/databases/orders",
    },
    overviewResult = {
      project: "projects/orders",
      catalog: { database: "instances/prod/databases/orders" },
    },
    askArgs = {
      kind: "choose",
      purpose: "masking-exemption-scope",
      prompt: "Which databases?",
    },
  }: {
    overviewArgs?: Record<string, unknown>;
    overviewResult?: Record<string, unknown>;
    askArgs?: Record<string, unknown>;
  } = {}
): Message[] => [
  maskingTurn[0],
  {
    role: "assistant",
    toolCalls: [
      {
        id: "overview",
        name: "get_masking_overview",
        arguments: JSON.stringify(overviewArgs),
      },
    ],
  },
  {
    role: "tool",
    toolCallId: "overview",
    content: JSON.stringify(overviewResult),
  },
  {
    role: "assistant",
    toolCalls: [
      { id: "question", name: "ask_user", arguments: JSON.stringify(askArgs) },
    ],
  },
  { role: "tool", toolCallId: "question", content: JSON.stringify(answer) },
];

const unmaskMethodTurn = (
  args: Record<string, unknown> = {},
  answer: Record<string, unknown> = {
    kind: "choose",
    answer: "Masking exemption",
    value: "masking-exemption",
  }
): Message[] => [
  { role: "user", content: "Help me view unmasked data." },
  {
    role: "assistant",
    toolCalls: [
      {
        id: "method",
        name: "ask_unmask_method",
        arguments: JSON.stringify(args),
      },
    ],
  },
  { role: "tool", toolCallId: "method", content: JSON.stringify(answer) },
];

describe("getMaskingResumeArgs", () => {
  test.each(["masking-exemption", "just-in-time"])(
    "resumes %s without a preceding masking overview",
    (value) => {
      expect(
        getMaskingResumeArgs(
          unmaskMethodTurn(
            { queryAccessMissing: true, forCurrentUser: false },
            { kind: "choose", answer: "Selected method", value }
          )
        )
      ).toEqual({});
    }
  );

  test.each([
    { project: "projects/orders" },
    { database: "instances/prod/databases/orders" },
    { database: "projects/orders/instances/prod/databases/orders" },
    {
      project: "projects/orders",
      database: "instances/prod/databases/orders",
    },
    {
      project: "projects/orders",
      database: "projects/orders/instances/prod/databases/orders",
    },
  ])("preserves explicit unmask method scope %j", (scope) => {
    const messages = unmaskMethodTurn({
      ...scope,
      queryAccessMissing: false,
      forCurrentUser: true,
    });
    const before = structuredClone(messages);
    expect(getMaskingResumeArgs(messages)).toEqual(scope);
    expect(messages).toEqual(before);
  });

  test.each(["", " \t "])(
    "treats blank optional method scope %j as omitted",
    (blank) => {
      expect(
        getMaskingResumeArgs(
          unmaskMethodTurn({ project: "projects/orders", database: blank })
        )
      ).toEqual({ project: "projects/orders" });
      expect(
        getMaskingResumeArgs(
          unmaskMethodTurn({
            project: blank,
            database: "instances/prod/databases/orders",
          })
        )
      ).toEqual({ database: "instances/prod/databases/orders" });
      expect(
        getMaskingResumeArgs(
          unmaskMethodTurn({ project: blank, database: blank })
        )
      ).toEqual({});
    }
  );

  test("does not revive an overview scope or interpret the method as a resource", () => {
    expect(
      getMaskingResumeArgs([
        ...maskingTurn.slice(0, 3),
        ...unmaskMethodTurn().slice(1),
      ])
    ).toEqual({});
  });

  test.each([
    { kind: "cancelled", reason: "user" },
    { kind: "input", answer: "Masking exemption", value: "masking-exemption" },
    {
      kind: "confirm",
      answer: "Masking exemption",
      value: "masking-exemption",
    },
    { kind: "choose", answer: "Masking exemption" },
    { kind: "choose", answer: "Masking exemption", value: 7 },
    { kind: "choose", answer: "Masking exemption", value: "projects/orders" },
    {
      kind: "choose",
      answer: "Masking exemption",
      value: "instances/prod/databases/orders",
    },
    {
      kind: "choose",
      answer: "Masking exemption",
      value: "semantic-type:email",
    },
    { kind: "choose", answer: "Masking exemption", value: "MASKING-EXEMPTION" },
    { kind: "choose", answer: "Masking exemption", value: "just-in-time " },
  ])("rejects an invalid unmask method answer %j", (answer) => {
    expect(getMaskingResumeArgs(unmaskMethodTurn({}, answer))).toBeUndefined();
  });

  test.each([
    { project: 7 },
    { database: 7 },
    { queryAccessMissing: "true" },
    { forCurrentUser: "false" },
    { project: "orders" },
    { project: "projects/-" },
    { project: "projects/*" },
    { project: "projects/orders/" },
    { project: " projects/orders " },
    { database: "orders" },
    { database: " instances/prod/databases/orders " },
    { database: "instances/-/databases/orders" },
    { database: "instances/*/databases/orders" },
    { database: "instances/prod/databases/-" },
    { database: "instances/prod/databases/*" },
    { database: "projects/-/instances/prod/databases/orders" },
    { database: "projects/*/instances/prod/databases/orders" },
    { database: "projects/orders/instances/-/databases/orders" },
    { database: "instances/prod/databases/orders/metadata" },
    {
      project: "projects/orders",
      database: "projects/other/instances/prod/databases/orders",
    },
    { semanticTypeId: "email" },
  ])("rejects malformed or conflicting unmask method arguments %j", (args) => {
    expect(getMaskingResumeArgs(unmaskMethodTurn(args))).toBeUndefined();
  });

  test("does not resume an unmask method after the run has completed", () => {
    const messages = unmaskMethodTurn();
    expect(
      getMaskingResumeArgs([
        messages[0]!,
        {
          role: "assistant",
          toolCalls: [{ id: "done", name: "done", arguments: "{}" }],
        },
        { role: "tool", toolCallId: "done", content: '{"success":true}' },
        ...messages.slice(1),
      ])
    ).toBeUndefined();
  });

  test.each([
    {
      name: "another rule database",
      tool: "ask_masking_rule_scope",
      askArgs: { project: "projects/orders" },
      answer: {
        kind: "choose",
        answer: "Another database",
        value: "another-database",
      },
      expected: { project: "projects/orders" },
    },
    {
      name: "workspace rule scope",
      tool: "ask_masking_rule_scope",
      askArgs: { project: "projects/orders" },
      answer: {
        kind: "choose",
        answer: "All workspace databases",
        value: "workspace",
      },
      expected: {},
    },
    {
      name: "another exemption database",
      tool: "ask_masking_exemption_scope",
      askArgs: {},
      answer: {
        kind: "choose",
        answer: "Another database",
        value: "another-database",
      },
      expected: { project: "projects/orders" },
    },
    {
      name: "semantic type selection",
      tool: "ask_masking_semantic_type",
      askArgs: {},
      answer: {
        kind: "choose",
        answer: "Email",
        value: "semantic-type:custom-email",
      },
      expected: {
        project: "projects/orders",
        database: "instances/prod/databases/orders",
      },
    },
    {
      name: "expiration clarification",
      tool: "ask_user",
      askArgs: { kind: "input", prompt: "When should it expire?" },
      answer: { kind: "input", answer: "No expiration" },
      expected: {
        project: "projects/orders",
        database: "instances/prod/databases/orders",
      },
    },
  ])(
    "resumes $name after an exact semantic type lookup",
    ({ tool, askArgs, answer, expected }) => {
      const messages = exemptionScopeTurn(answer, {
        overviewArgs: {
          project: "projects/orders",
          database: "instances/prod/databases/orders",
          semanticTypeId: "custom-email",
        },
        askArgs,
      });
      messages[3].toolCalls![0].name = tool;
      expect(getMaskingResumeArgs(messages)).toEqual(expected);
    }
  );

  test.each([null, 7, true, {}])(
    "rejects a malformed semantic type filter %j",
    (semanticTypeId) => {
      const messages = exemptionScopeTurn(
        {
          kind: "choose",
          answer: "Another database",
          value: "another-database",
        },
        {
          overviewArgs: { project: "projects/orders", semanticTypeId },
          askArgs: {},
        }
      );
      messages[3].toolCalls![0].name = "ask_masking_rule_scope";
      expect(getMaskingResumeArgs(messages)).toBeUndefined();
    }
  );

  test("still rejects unknown overview arguments alongside an exact type lookup", () => {
    const messages = exemptionScopeTurn(
      { kind: "choose", answer: "Another database", value: "another-database" },
      {
        overviewArgs: {
          project: "projects/orders",
          semanticTypeId: "custom-email",
          unexpected: true,
        },
        askArgs: {},
      }
    );
    messages[3].toolCalls![0].name = "ask_masking_rule_scope";
    expect(getMaskingResumeArgs(messages)).toBeUndefined();
  });

  test.each([
    ["workspace", {}],
    ["choose-project", {}],
    ["projects/billing", { project: "projects/billing" }],
    ["another-database", { project: "projects/orders" }],
    [
      "instances/staging/databases/selected",
      {
        project: "projects/orders",
        database: "instances/staging/databases/selected",
      },
    ],
  ])(
    "refreshes the global rule scope %s without reviving the old database",
    (value, expected) => {
      const messages = exemptionScopeTurn(
        { kind: "choose", answer: "Selected rule scope", value },
        { askArgs: {} }
      );
      messages[3].toolCalls![0].name = "ask_masking_rule_scope";
      expect(getMaskingResumeArgs(messages)).toEqual(expected);
    }
  );

  test.each([
    "bb.default",
    "custom-email",
    "create-semantic-type",
    "semantic-type:bb.default",
    "semantic-type:create-semantic-type",
    "action:create-semantic-type",
  ])("refreshes current eligibility after semantic type choice %s", (value) => {
    const messages = exemptionScopeTurn(
      { kind: "choose", answer: "Selected method", value },
      { askArgs: {} }
    );
    messages[3].toolCalls![0].name = "ask_masking_semantic_type";
    expect(getMaskingResumeArgs(messages)).toEqual({
      project: "projects/orders",
      database: "instances/prod/databases/orders",
    });
  });

  test.each([
    "orders",
    "projects/-",
    "instances/-/databases/orders",
    "projects/other/instances/prod/databases/orders",
  ])("rejects an invalid or conflicting global scope answer %s", (value) => {
    const messages = exemptionScopeTurn(
      { kind: "choose", answer: "Selected scope", value },
      { askArgs: {} }
    );
    messages[3].toolCalls![0].name = "ask_masking_rule_scope";
    expect(getMaskingResumeArgs(messages)).toBeUndefined();
  });

  test("uses the explicit verified rule question project for a selected database", () => {
    const messages = exemptionScopeTurn(
      {
        kind: "choose",
        answer: "Selected scope",
        value: "instances/staging/databases/billing",
      },
      { askArgs: { project: "projects/billing" } }
    );
    messages[3].toolCalls![0].name = "ask_masking_rule_scope";
    expect(getMaskingResumeArgs(messages)).toEqual({
      project: "projects/billing",
      database: "instances/staging/databases/billing",
    });
  });

  test.each([
    ["projects/orders", { project: "projects/orders" }],
    ["another-database", { project: "projects/orders" }],
    [
      "instances/prod/databases/orders",
      {
        project: "projects/orders",
        database: "instances/prod/databases/orders",
      },
    ],
    [
      "projects/orders/instances/prod/databases/orders",
      {
        project: "projects/orders",
        database: "projects/orders/instances/prod/databases/orders",
      },
    ],
  ])("restores the dedicated scope answer %s", (value, expected) => {
    const messages = exemptionScopeTurn(
      { kind: "choose", answer: "Selected scope", value },
      { askArgs: {} }
    );
    messages[3].toolCalls![0].name = "ask_masking_exemption_scope";
    expect(getMaskingResumeArgs(messages)).toEqual(expected);
  });

  test.each([
    ["projects/billing", { project: "projects/billing" }],
    ["another-database", { project: "projects/billing" }],
    [
      "instances/prod/databases/billing",
      {
        project: "projects/billing",
        database: "instances/prod/databases/billing",
      },
    ],
    [
      "projects/billing/instances/prod/databases/billing",
      {
        project: "projects/billing",
        database: "projects/billing/instances/prod/databases/billing",
      },
    ],
  ])(
    "uses the explicit exemption question project for scope %s",
    (value, expected) => {
      const messages = exemptionScopeTurn(
        { kind: "choose", answer: "Selected scope", value },
        { askArgs: { project: "projects/billing" } }
      );
      messages[3].toolCalls![0].name = "ask_masking_exemption_scope";
      expect(getMaskingResumeArgs(messages)).toEqual(expected);
    }
  );

  test.each([
    "projects/orders",
    "projects/orders/instances/prod/databases/orders",
    "instances/-/databases/billing",
    "instances/prod/databases/-",
    "instances/prod/databases/*",
  ])("rejects an invalid or foreign explicit exemption scope %s", (value) => {
    const messages = exemptionScopeTurn(
      { kind: "choose", answer: "Selected scope", value },
      { askArgs: { project: "projects/billing" } }
    );
    messages[3].toolCalls![0].name = "ask_masking_exemption_scope";
    expect(getMaskingResumeArgs(messages)).toBeUndefined();
  });

  test.each([
    null,
    7,
    "billing",
    "projects/-",
    "projects/*",
    "projects/billing/",
    " projects/billing ",
  ])("rejects an invalid explicit exemption question project %j", (project) => {
    const messages = exemptionScopeTurn(
      { kind: "choose", answer: "Another database", value: "another-database" },
      { askArgs: { project } }
    );
    messages[3].toolCalls![0].name = "ask_masking_exemption_scope";
    expect(getMaskingResumeArgs(messages)).toBeUndefined();
  });

  test.each(["", " \t "])(
    "uses the overview project when the optional exemption question project is blank %j",
    (project) => {
      const messages = exemptionScopeTurn(
        {
          kind: "choose",
          answer: "Another database",
          value: "another-database",
        },
        { askArgs: { project } }
      );
      messages[3].toolCalls![0].name = "ask_masking_exemption_scope";
      expect(getMaskingResumeArgs(messages)).toEqual({
        project: "projects/orders",
      });
    }
  );

  test("still requires a preceding overview for an explicit exemption question project", () => {
    const messages = exemptionScopeTurn(
      { kind: "choose", answer: "Selected project", value: "projects/billing" },
      { askArgs: { project: "projects/billing" } }
    );
    messages[3].toolCalls![0].name = "ask_masking_exemption_scope";
    expect(
      getMaskingResumeArgs([messages[0]!, ...messages.slice(3)])
    ).toBeUndefined();
  });

  test.each([
    { kind: "input", answer: "No expiration" },
    { kind: "choose", answer: "Another project", value: "projects/other" },
    {
      kind: "choose",
      answer: "Another database",
      value: "projects/other/instances/prod/databases/orders",
    },
    { kind: "choose", answer: "Orders", value: "instances/-/databases/orders" },
  ])("rejects an invalid dedicated scope answer %j", (answer) => {
    const messages = exemptionScopeTurn(answer, { askArgs: {} });
    messages[3].toolCalls![0].name = "ask_masking_exemption_scope";
    expect(getMaskingResumeArgs(messages)).toBeUndefined();
  });

  test("refreshes the latest overview after answering a clarification", () => {
    expect(getMaskingResumeArgs(maskingTurn)).toEqual({
      project: "projects/orders",
    });
  });

  test("replays the most recent selected scope without using old destinations", () => {
    expect(
      getMaskingResumeArgs([
        ...maskingTurn.slice(0, 3),
        {
          role: "assistant",
          toolCalls: [
            {
              id: "latest-overview",
              name: "get_masking_overview",
              arguments: "{}",
            },
          ],
        },
        {
          role: "tool",
          toolCallId: "latest-overview",
          content: JSON.stringify({
            project: "projects/billing",
            catalog: { database: "instances/prod/databases/billing" },
            openTargets: [{ kind: "open-global-masking" }],
          }),
        },
        ...maskingTurn.slice(3),
      ])
    ).toEqual({
      project: "projects/billing",
      database: "instances/prod/databases/billing",
    });
  });

  test.each([
    ["All databases in this project", "projects/orders"],
    ["Choose another database", "another-database"],
  ])("drops the previous database after choosing %s", (answer, value) => {
    expect(
      getMaskingResumeArgs(
        exemptionScopeTurn({ kind: "choose", answer, value })
      )
    ).toEqual({ project: "projects/orders" });
  });

  test.each([
    "instances/prod/databases/orders",
    "instances/staging/databases/other",
    "projects/orders/instances/prod/databases/orders",
  ])("refreshes the exact selected database %s", (database) => {
    expect(
      getMaskingResumeArgs(
        exemptionScopeTurn({
          kind: "choose",
          answer: "Selected database",
          value: database,
        })
      )
    ).toEqual({ project: "projects/orders", database });
  });

  test("uses a selected database when the previous overview was project-only", () => {
    expect(
      getMaskingResumeArgs(
        exemptionScopeTurn(
          {
            kind: "choose",
            answer: "Orders",
            value: "instances/prod/databases/orders",
          },
          { overviewArgs: {}, overviewResult: { project: "projects/orders" } }
        )
      )
    ).toEqual({
      project: "projects/orders",
      database: "instances/prod/databases/orders",
    });
  });

  test.each([
    { kind: "cancelled", reason: "user" },
    { kind: "confirm", answer: "No", confirmed: false },
    { kind: "input", answer: "All databases in this project" },
    { kind: "choose", answer: "All databases in this project" },
    { kind: "choose", answer: "Orders", value: 7 },
    { kind: "choose", answer: "Orders", value: "orders" },
    { kind: "choose", answer: "Orders", value: "projects/billing" },
    { kind: "choose", answer: "Orders", value: "projects/-" },
    {
      kind: "choose",
      answer: "Orders",
      value: "projects/billing/instances/prod/databases/orders",
    },
    {
      kind: "choose",
      answer: "Orders",
      value: "instances/prod/databases/orders/metadata",
    },
    { kind: "choose", answer: "Orders", value: "instances/-/databases/orders" },
  ])(
    "does not restore an old scope for an invalid scope answer %j",
    (answer) => {
      expect(getMaskingResumeArgs(exemptionScopeTurn(answer))).toBeUndefined();
    }
  );

  test("requires choose for the masking exemption scope purpose", () => {
    expect(
      getMaskingResumeArgs(
        exemptionScopeTurn(
          { kind: "choose", answer: "Orders", value: "projects/orders" },
          {
            askArgs: {
              kind: "input",
              purpose: "masking-exemption-scope",
              prompt: "Which databases?",
            },
          }
        )
      )
    ).toBeUndefined();
  });

  test.each(["projects/-", "projects/orders/", "orders", ""])(
    "does not resume a scope choice without an exact project %s",
    (project) => {
      expect(
        getMaskingResumeArgs(
          exemptionScopeTurn(
            {
              kind: "choose",
              answer: "Choose another database",
              value: "another-database",
            },
            { overviewArgs: { project }, overviewResult: { project } }
          )
        )
      ).toBeUndefined();
    }
  );

  test("does not refresh masking from a completed or older user turn", () => {
    expect(
      getMaskingResumeArgs([
        ...maskingTurn,
        {
          role: "assistant",
          toolCalls: [{ id: "done", name: "done", arguments: "{}" }],
        },
        { role: "tool", toolCallId: "done", content: '{"success":true}' },
        { role: "user", content: "Help me connect an instance." },
        ...maskingTurn.slice(3),
      ])
    ).toBeUndefined();
    expect(
      getMaskingResumeArgs([
        ...maskingTurn.slice(0, 3),
        {
          role: "assistant",
          toolCalls: [{ id: "done", name: "done", arguments: "{}" }],
        },
        { role: "tool", toolCallId: "done", content: '{"success":true}' },
        ...maskingTurn.slice(3),
      ])
    ).toBeUndefined();
  });

  test("ignores canceled asks, unfinished calls, and malformed overview arguments", () => {
    expect(
      getMaskingResumeArgs([
        ...maskingTurn.slice(0, -1),
        {
          role: "tool",
          toolCallId: "question",
          content: '{"kind":"cancelled","reason":"user"}',
        },
      ])
    ).toBeUndefined();
    expect(getMaskingResumeArgs(maskingTurn.slice(0, -1))).toBeUndefined();
    expect(
      getMaskingResumeArgs(
        maskingTurn.map((message) =>
          message.role === "assistant" &&
          message.toolCalls?.[0]?.name === "get_masking_overview"
            ? {
                ...message,
                toolCalls: [
                  { ...message.toolCalls[0], arguments: '{"project":7}' },
                ],
              }
            : message
        )
      )
    ).toBeUndefined();
  });
});
