import { describe, expect, test } from "vitest";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type { AgentSqlEditorSnapshot } from "./sqlEditorContext";
import { bindAgentSqlBlockActions, extractAgentSqlBlocks } from "./sqlResponse";
import type { AgentMessage } from "./types";

describe("extractAgentSqlBlocks", () => {
  test("preserves exact raw whitespace and identifies duplicate fences by offset", () => {
    const first = "```SQL\r\n  SELECT 1;  \r\n\r\n```";
    const last = "~~~sql\n  SELECT 1;  \n~~~";
    const text = `Intro 😀\n\n${first}\n\n${last}`;
    const blocks = extractAgentSqlBlocks(text);
    expect(blocks).toEqual([
      {
        blockIndex: 0,
        start: text.indexOf(first),
        end: text.indexOf(first) + first.length,
        sql: "  SELECT 1;  \r\n",
      },
      {
        blockIndex: 1,
        start: text.indexOf(last),
        end: text.length,
        sql: "  SELECT 1;  ",
      },
    ]);
  });

  test("does not reassign an invalid SQL index to the next complete fence", () => {
    const text = [
      "> ```sql\n> SELECT 0;\n> ```",
      "    SELECT 1;",
      "```python\nprint(2)\n```",
      "```sql\nSELECT * FROM <table>;\n```",
      "  ```sql\n  SELECT 3;\n  ```",
      "```sql example\nSELECT 4;\n```",
      "```s&#113;l\nSELECT 4;\n```",
      "```sql\nSELECT 5;\n```",
      "```sql\nSELECT 6;",
    ].join("\n\n");
    expect(extractAgentSqlBlocks(text)).toEqual([
      {
        blockIndex: 4,
        start: text.indexOf("```sql\nSELECT 5"),
        end: text.indexOf("\n\n```sql\nSELECT 6"),
        sql: "SELECT 5;",
      },
    ]);
  });

  test("enforces a UTF-8 byte limit without truncating and rejects NUL", () => {
    const sql = "é".repeat(50_000);
    expect(extractAgentSqlBlocks(`~~~sql\n${sql}\n~~~`)[0]?.sql).toBe(sql);
    expect(extractAgentSqlBlocks(`~~~sql\n${sql}x\n~~~`)).toEqual([]);
    expect(extractAgentSqlBlocks("~~~sql\nSELECT '\0';\n~~~")).toEqual([]);
  });
});

const snapshot: AgentSqlEditorSnapshot = {
  version: 1,
  project: "projects/sample",
  tab: { id: "tab", mode: "SAVED_QUERY" },
  connection: {
    database: "instances/prod/databases/app",
    project: "projects/sample",
    engine: Engine.POSTGRES,
  },
  editorTarget: {
    actor: "users/member",
    workspace: "workspaces/sample",
    hostId: "host",
    mountId: "mount",
    tabId: "tab",
    tabMode: "SAVED_QUERY",
    project: "projects/sample",
    modelUri: "file:///query.sql",
    modelVersion: 3,
    connection: {
      database: "instances/prod/databases/app",
      project: "projects/sample",
      engine: Engine.POSTGRES,
    },
    selection: { start: 0, end: 0, caret: 0 },
  },
};
const user: AgentMessage = {
  id: "user",
  chatId: "chat",
  role: "user",
  content: "Write a query",
  createdTs: 1,
  metadata: { sqlEditor: snapshot },
};
const assistant: AgentMessage = {
  id: "answer",
  chatId: "chat",
  role: "assistant",
  content: "```sql\nSELECT 1;\n```\n\n```sql\n  SELECT 2;  \n```",
  createdTs: 2,
  metadata: {
    runId: "run",
    sqlEditor: snapshot,
    sqlActionProvenance: {
      version: 1,
      runId: "run",
      runMode: "default",
      diagnostic: false,
    },
    actions: [{ kind: "sql-editor-block", blockIndex: 1 }],
  },
};

describe("bindAgentSqlBlockActions", () => {
  test("binds only the marked fence to the answer, not the chat's later context", () => {
    const later = { ...user, id: "later", metadata: undefined };
    expect(
      bindAgentSqlBlockActions(assistant, [user, assistant, later])
    ).toEqual([
      {
        chatId: "chat",
        messageId: "answer",
        blockIndex: 1,
        sourceHash: expect.stringMatching(/^[a-f0-9]{64}$/),
        sql: "  SELECT 2;  ",
        snapshot,
      },
    ]);
  });

  test("does not offer an editor action for an answer based on multiple queried databases", () => {
    const query: AgentMessage = {
      chatId: "chat",
      id: "tool-call",
      createdTs: 1,
      role: "assistant",
      toolCalls: [
        "instances/prod/databases/app",
        "instances/prod/databases/other",
      ].map((name, index) => ({
        id: `query-${index}`,
        name: "call_api",
        arguments: JSON.stringify({
          operationId: "SQLService/Query",
          body: { name, statement: "SELECT 1" },
        }),
      })),
    };
    expect(
      bindAgentSqlBlockActions(assistant, [user, query, assistant])
    ).toEqual([]);
  });

  test.each([
    { runId: undefined },
    { pending: true },
    { sqlActionProvenance: undefined },
    {
      sqlActionProvenance: {
        version: 1,
        runId: "run",
        runMode: "read-only",
        diagnostic: false,
      },
    },
    {
      sqlActionProvenance: {
        version: 1,
        runId: "run",
        runMode: "default",
        diagnostic: true,
      },
    },
    {
      resourceReferences: [
        {
          kind: "database",
          name: "instances/prod/databases/other",
          project: "projects/sample",
          label: "other",
        },
      ],
    },
  ] as const)("rejects ineligible answer metadata %j", (metadata) => {
    const answer = {
      ...assistant,
      metadata: { ...assistant.metadata, ...metadata },
    } as AgentMessage;
    expect(bindAgentSqlBlockActions(answer, [user, answer])).toEqual([]);
  });
});
