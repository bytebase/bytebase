import { describe, expect, test, vi } from "vitest";
import { Engine } from "@/types/proto-es/v1/common_pb";
import {
  QueryOption_ExplainFormat,
  QueryOption_RedisRunCommandsOn,
} from "@/types/proto-es/v1/sql_service_pb";
import { useAgentStore } from "../store/agent";
import { agentApiExecutionEvidence } from "./apiExecution";
import type { AgentSqlEditorSnapshot } from "./sqlEditorContext";
import {
  bindAgentCompletedQueryActions,
  bindAgentSqlBlockActions,
  extractAgentSqlBlocks,
  getCurrentAgentSqlActionSource,
} from "./sqlResponse";
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

describe("bindAgentCompletedQueryActions", () => {
  const query: AgentMessage = {
    id: "query-call",
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
          body: {
            name: snapshot.connection!.database,
            statement: " SELECT 20; ",
          },
        }),
      },
    ],
  };
  const result: AgentMessage = {
    id: "query-result",
    chatId: "chat",
    role: "tool",
    createdTs: 3,
    toolCallId: "query",
    metadata: { runId: "run" },
    content: JSON.stringify({
      status: 200,
      agentExecution: agentApiExecutionEvidence("rpc-success"),
      response: {
        results: [
          { columnNames: ["value"], rows: [{ values: [{ int32Value: 20 }] }] },
        ],
      },
    }),
  };
  const completion: AgentMessage = {
    ...assistant,
    content: "Returned one row.",
    createdTs: 4,
    metadata: { ...assistant.metadata, actions: undefined },
  };
  const completedTurn = [user, query, result, completion];
  const queryWith = (parameters: Record<string, unknown>) => ({
    ...query,
    toolCalls: [
      {
        ...query.toolCalls![0],
        arguments: JSON.stringify({
          operationId: "SQLService/Query",
          body: {
            name: snapshot.connection!.database,
            statement: " SELECT 20; ",
            ...parameters,
          },
        }),
      },
    ],
  });

  test.each([
    { schema: "another_schema" },
    { container: "another_container" },
    { dataSourceId: "another_reader" },
    { data_source_id: "another_reader" },
    { queryOption: { redisRunCommandsOn: "ALL_NODES" } },
    { query_option: { explain_format: "JSON" } },
    { queryOption: { explainFormat: "JSON" } },
    { queryOption: { unknownOption: true } },
    { queryOption: "invalid" },
    { schema: 42 },
  ])(
    "withholds completed-query actions for different or invalid parameters %j",
    (parameters) => {
      expect(
        bindAgentCompletedQueryActions(completion, [
          user,
          queryWith(parameters),
          result,
          completion,
        ])
      ).toEqual([]);
    }
  );

  test.each([
    {},
    { schema: "", container: "", dataSourceId: "" },
    { schema: null, container: null, queryOption: null },
    { queryOption: {} },
    {
      queryOption: {
        redisRunCommandsOn: "REDIS_RUN_COMMANDS_ON_UNSPECIFIED",
        explainFormat: "EXPLAIN_FORMAT_UNSPECIFIED",
      },
    },
    { queryOption: { redisRunCommandsOn: "SINGLE_NODE" } },
    { queryOption: { redisRunCommandsOn: 0, explainFormat: 0 } },
  ])(
    "accepts equivalent empty/default execution parameters %j",
    (parameters) => {
      expect(
        bindAgentCompletedQueryActions(completion, [
          user,
          queryWith(parameters),
          result,
          completion,
        ])
      ).toHaveLength(1);
    }
  );

  test("accepts matching explicit execution parameters with protobuf enum normalization", () => {
    const connection = {
      ...snapshot.connection!,
      querySchema: "public",
      container: "partition",
      dataSourceId: "reader",
    };
    const configured = {
      ...snapshot,
      connection,
      editorTarget: { ...snapshot.editorTarget!, connection },
      queryOptions: {
        redisRunCommandsOn: QueryOption_RedisRunCommandsOn.ALL_NODES,
        explainFormat: QueryOption_ExplainFormat.JSON,
      },
    };
    const configuredUser = {
      ...user,
      metadata: { ...user.metadata, sqlEditor: configured },
    };
    const configuredAnswer = {
      ...completion,
      metadata: { ...completion.metadata, sqlEditor: configured },
    };
    const configuredQuery = queryWith({
      schema: "public",
      container: "partition",
      data_source_id: "reader",
      query_option: {
        redis_run_commands_on: "ALL_NODES",
        explain_format: "JSON",
      },
    });
    expect(
      bindAgentCompletedQueryActions(configuredAnswer, [
        configuredUser,
        configuredQuery,
        result,
        configuredAnswer,
      ])
    ).toHaveLength(1);
  });

  test("binds the original request body without freezing the click-time row limit", () => {
    const first = bindAgentCompletedQueryActions(completion, completedTurn)[0];
    const differentLimit = bindAgentCompletedQueryActions(completion, [
      user,
      queryWith({ limit: 5 }),
      result,
      completion,
    ]);
    expect(differentLimit).toHaveLength(1);
    expect(differentLimit[0].sourceHash).not.toBe(first.sourceHash);
    const explicitDefaults = bindAgentCompletedQueryActions(completion, [
      user,
      queryWith({ queryOption: {} }),
      result,
      completion,
    ]);
    expect(explicitDefaults).toHaveLength(1);
    expect(explicitDefaults[0].sourceHash).not.toBe(first.sourceHash);
  });

  test("binds actual successful SQL without a model fence or proposal marker", () => {
    expect(bindAgentCompletedQueryActions(completion, completedTurn)).toEqual([
      {
        chatId: "chat",
        messageId: "answer",
        blockIndex: 0,
        queryResultMessageId: "query-result",
        sql: " SELECT 20; ",
        snapshot,
        sourceHash: expect.stringMatching(/^[a-f0-9]{64}$/),
      },
    ]);
    expect(bindAgentSqlBlockActions(completion, completedTurn)).toEqual([]);
  });

  test("binds the evidence identity and answer bytes, not model-provided SQL", () => {
    const first = bindAgentCompletedQueryActions(completion, completedTurn)[0];
    const changed = { ...completion, content: "Another summary." };
    const renamedResult = { ...result, id: "another-result" };
    expect(
      bindAgentCompletedQueryActions(changed, [user, query, result, changed])[0]
        .sourceHash
    ).not.toBe(first.sourceHash);
    expect(
      bindAgentCompletedQueryActions(completion, [
        user,
        query,
        renamedResult,
        completion,
      ])[0]
    ).toMatchObject({ queryResultMessageId: "another-result" });
    expect(
      bindAgentCompletedQueryActions(completion, [
        user,
        query,
        renamedResult,
        completion,
      ])[0].sourceHash
    ).not.toBe(first.sourceHash);
  });

  test.each([
    { runId: undefined },
    { sqlActionProvenance: undefined },
    { pending: true },
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
    { sqlEditor: { ...snapshot, batchActive: true } },
  ])("rejects ineligible completed answer %j", (metadata) => {
    const changed = {
      ...completion,
      metadata: { ...completion.metadata, ...metadata },
    } as AgentMessage;
    expect(
      bindAgentCompletedQueryActions(changed, [user, query, result, changed])
    ).toEqual([]);
  });

  test("does not bind a result from another connection or run", () => {
    const other = {
      ...query,
      toolCalls: [
        {
          ...query.toolCalls![0],
          arguments: JSON.stringify({
            operationId: "SQLService/Query",
            body: {
              name: "instances/other/databases/other",
              statement: "SELECT 20;",
            },
          }),
        },
      ],
    };
    expect(
      bindAgentCompletedQueryActions(completion, [
        user,
        other,
        result,
        completion,
      ])
    ).toEqual([]);
    expect(
      bindAgentCompletedQueryActions(completion, [
        user,
        query,
        { ...result, metadata: { runId: "other" } },
        completion,
      ])
    ).toEqual([]);
  });

  test("revalidates completed-query evidence rather than requiring a proposal marker", () => {
    const source = bindAgentCompletedQueryActions(completion, completedTurn)[0];
    expect(source).toBeDefined();
    let liveMessages = completedTurn;
    const original = useAgentStore.getState();
    const state = {
      ...original,
      workspaceName: "workspaces/sample",
      currentChatId: "chat",
      getChat: () => ({
        id: "chat",
        title: "Query",
        context: "projects/sample",
        workspaceName: "workspaces/sample",
        status: "idle" as const,
        createdTs: 1,
        updatedTs: 4,
        archived: false,
        totalTokensUsed: 0,
      }),
      getMessages: () => liveMessages,
      remoteBusyChatIds: {},
      abortControllersByChatId: {},
      queuedUserMessageInFlightByChatId: {},
      queuedUserMessagesByChatId: {},
      followUpQueuesByChatId: {},
    };
    const spy = vi.spyOn(useAgentStore, "getState").mockReturnValue(state);
    try {
      expect(getCurrentAgentSqlActionSource(source)).toEqual(source);
      liveMessages = [
        user,
        query,
        {
          ...result,
          content: JSON.stringify({ error: "Result no longer verified" }),
        },
        completion,
      ];
      expect(getCurrentAgentSqlActionSource(source)).toBeUndefined();
      liveMessages = completedTurn;
      expect(
        getCurrentAgentSqlActionSource({
          ...source,
          queryResultMessageId: "other",
        })
      ).toBeUndefined();
      expect(
        getCurrentAgentSqlActionSource({
          ...source,
          sql: "SELECT private_value;",
        })
      ).toBeUndefined();
    } finally {
      spy.mockRestore();
    }
  });
});
