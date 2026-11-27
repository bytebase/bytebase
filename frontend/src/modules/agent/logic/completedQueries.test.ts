import { describe, expect, test } from "vitest";
import { agentApiExecutionEvidence } from "./apiExecution";
import { getAgentCompletedQueries } from "./completedQueries";
import { omitAgentQueryRows } from "./queryResultRetention";
import type { AgentMessage } from "./types";

const database = "instances/example/databases/app";
const statement = " SELECT employee_id FROM employee; ";
const user: AgentMessage = {
  id: "user",
  chatId: "chat",
  role: "user",
  createdTs: 1,
  content: "Show employees",
};
const call: AgentMessage = {
  id: "call",
  chatId: "chat",
  role: "assistant",
  createdTs: 2,
  metadata: { runId: "run" },
  toolCalls: [
    {
      id: "query",
      name: "call_api",
      arguments: JSON.stringify({
        operationId: "SQLService/Query",
        body: { name: database, statement },
      }),
    },
  ],
};
const result: AgentMessage = {
  id: "result",
  chatId: "chat",
  role: "tool",
  createdTs: 3,
  toolCallId: "query",
  metadata: { runId: "run" },
  content: JSON.stringify({
    status: 200,
    agentExecution: agentApiExecutionEvidence("rpc-success", {
      httpStatus: 200,
    }),
    response: {
      results: [
        {
          columnNames: ["employee_id"],
          rows: [{ values: [{ stringValue: "******" }] }],
          masked: [{ maskingRuleId: "rule" }],
        },
      ],
    },
  }),
};
const answer: AgentMessage = {
  id: "answer",
  chatId: "chat",
  role: "assistant",
  createdTs: 4,
  metadata: { runId: "run" },
  content: "Returned an employee preview.",
};
const messages = [user, call, result, answer];
const withBody = (body: Record<string, unknown>) => ({
  ...call,
  toolCalls: [
    {
      ...call.toolCalls![0],
      arguments: JSON.stringify({ operationId: "SQLService/Query", body }),
    },
  ],
});

describe("getAgentCompletedQueries", () => {
  test("uses actual completed Query bytes and bounded retained rows rather than answer prose", () => {
    const completed = getAgentCompletedQueries(answer, messages);
    expect(completed).toHaveLength(1);
    expect(completed[0]).toMatchObject({
      callMessageId: "call",
      resultMessageId: "result",
      toolCallIndex: 0,
      database,
      statement,
      preview: {
        resultSets: [{ columns: ["employee_id"], returnedRowCount: 1 }],
      },
    });
    expect(completed[0].preview.resultSets[0].rows[0].values[0].kind).toEqual({
      case: "stringValue",
      value: "******",
    });
  });

  test("never attaches a later turn or result to a historical answer", () => {
    const early = { ...answer, id: "early", createdTs: 1 };
    expect(
      getAgentCompletedQueries(early, [user, early, call, result])
    ).toEqual([]);
    expect(
      getAgentCompletedQueries(answer, [
        user,
        call,
        { ...user, id: "later-user" },
        result,
        answer,
      ])
    ).toEqual([]);
    expect(
      getAgentCompletedQueries(answer, [user, call, answer, result])
    ).toEqual([]);
  });

  test.each(["call", "result", "answer"] as const)(
    "rejects foreign run/chat on %s",
    (kind) => {
      const index = { call: 1, result: 2, answer: 3 }[kind]!;
      for (const override of [
        { chatId: "other" },
        { metadata: { runId: "other" } },
      ]) {
        const modified = messages.map((message, i) =>
          i === index ? { ...message, ...override } : message
        );
        expect(getAgentCompletedQueries(modified[3], modified)).toEqual([]);
      }
    }
  );

  test("rejects pending, errored, duplicated, or detached answers", () => {
    for (const metadata of [
      { runId: "run", pending: true },
      { runId: "run", error: "stopped" },
      {},
    ]) {
      const modified = { ...answer, metadata };
      expect(
        getAgentCompletedQueries(modified, [user, call, result, modified])
      ).toEqual([]);
    }
    expect(getAgentCompletedQueries(answer, [...messages, answer])).toEqual([]);
    expect(
      getAgentCompletedQueries({ ...answer, content: "fabricated" }, messages)
    ).toEqual([]);
  });

  test("rejects ambiguous call IDs or duplicate tool results instead of guessing", () => {
    expect(
      getAgentCompletedQueries(answer, [
        user,
        { ...call, toolCalls: [...call.toolCalls!, ...call.toolCalls!] },
        result,
        answer,
      ])
    ).toEqual([]);
    expect(
      getAgentCompletedQueries(answer, [
        user,
        call,
        result,
        { ...result, id: "duplicate" },
        answer,
      ])
    ).toEqual([]);
    expect(
      getAgentCompletedQueries(answer, [
        user,
        call,
        { ...call, id: "intervening", toolCalls: [] },
        result,
        answer,
      ])
    ).toEqual([]);
  });

  test("preserves independently owned queries including reused IDs in successive calls", () => {
    const second = {
      ...call,
      id: "second-call",
      toolCalls: [
        {
          ...call.toolCalls![0],
          arguments: JSON.stringify({
            operationId: "bytebase.v1.SQLService.Query",
            body: {
              name: "projects/example/instances/example/databases/other",
              statement: "SELECT 2;",
            },
          }),
        },
      ],
    };
    const secondResult = { ...result, id: "second-result" };
    const completed = getAgentCompletedQueries(answer, [
      user,
      call,
      result,
      second,
      secondResult,
      answer,
    ]);
    expect(
      completed.map(({ database, statement, resultMessageId }) => ({
        database,
        statement,
        resultMessageId,
      }))
    ).toEqual([
      { database, statement, resultMessageId: "result" },
      {
        database: "projects/example/instances/example/databases/other",
        statement: "SELECT 2;",
        resultMessageId: "second-result",
      },
    ]);
  });

  test("retains omitted preview counts without restoring or querying row values", () => {
    const omitted = { ...result, content: omitAgentQueryRows(result.content!) };
    expect(
      getAgentCompletedQueries(answer, [user, call, omitted, answer])[0].preview
        .resultSets[0]
    ).toMatchObject({ previewLevel: "omitted", returnedRowCount: 1, rows: [] });
  });

  test.each([
    { name: database, statement, explain: true },
    { name: "app", statement },
    { name: database, statement: "" },
    { name: database, statement: "SELECT '\0';" },
  ])("does not promote an invalid or explain request %j", (body) => {
    expect(
      getAgentCompletedQueries(answer, [user, withBody(body), result, answer])
    ).toEqual([]);
  });

  test.each([
    { status: 200, response: { results: [{ columnNames: ["id"], rows: [] }] } },
    {
      status: 200,
      agentExecution: agentApiExecutionEvidence("fetch-failed"),
      response: { results: [{ columnNames: ["id"], rows: [] }] },
    },
    {
      status: 200,
      agentExecution: agentApiExecutionEvidence("query-result-error"),
      response: { results: [{ columnNames: ["id"], error: "denied" }] },
    },
    {
      status: 200,
      agentExecution: agentApiExecutionEvidence("rpc-success"),
      response: { results: [{ columnNames: ["id"], error: "denied" }] },
    },
    {
      status: 200,
      agentExecution: agentApiExecutionEvidence("rpc-success"),
      response: { results: [{ columnNames: ["id"], permissionDenied: {} }] },
    },
    {
      status: 200,
      agentExecution: agentApiExecutionEvidence("rpc-success"),
      response: { results: [{ columnNames: ["id"], rows: "invalid" }] },
    },
  ])(
    "does not promote failed, unknown, or malformed results %j",
    (envelope) => {
      expect(
        getAgentCompletedQueries(answer, [
          user,
          call,
          { ...result, content: JSON.stringify(envelope) },
          answer,
        ])
      ).toEqual([]);
    }
  );
});
