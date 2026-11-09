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

describe("getMaskingResumeArgs", () => {
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
