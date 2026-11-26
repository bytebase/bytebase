import { describe, expect, test } from "vitest";
import { Engine } from "@/types/proto-es/v1/common_pb";
import {
  getAgentSourceSqlEditorContext,
  normalizeAgentChatSnapshot,
  normalizeAgentSqlEditorSnapshot,
} from "./sqlEditorContext";
import type { AgentMessage } from "./types";

const project = "projects/example";
const tab = { id: "tab-1", mode: "SAVED_QUERY" as const };
const connection = {
  database: "instances/demo/databases/example",
  project,
  engine: Engine.POSTGRES,
  querySchema: "public",
  container: "items",
  dataSourceId: "read",
};
const page = { path: "/sql-editor", title: "SQL Editor" };

describe("SQL context from a source answer", () => {
  const user: AgentMessage = {
    id: "u",
    chatId: "chat",
    role: "user",
    createdTs: 1,
    content: "What is a join?",
  };
  const answer: AgentMessage = {
    ...user,
    id: "a",
    role: "assistant",
    content: "A join combines rows.",
  };
  const snapshot = { version: 1 as const, project, connection };
  const metadata = { sqlEditor: snapshot };

  test("distinguishes a valid legacy answer from a shared normalized SQL snapshot", () => {
    expect(
      getAgentSourceSqlEditorContext([user, answer], answer.id)
    ).toBeUndefined();
    expect(
      getAgentSourceSqlEditorContext(
        [
          { ...user, metadata },
          { ...answer, metadata },
        ],
        answer.id
      )
    ).toEqual({ kind: "ready", snapshot });
  });

  test.each([
    [user, { ...answer, metadata }],
    [{ ...user, metadata }, answer],
    [user, answer, answer],
    [answer],
    [user, { ...answer, chatId: "foreign" }],
    [user, { ...answer, metadata: { pending: true } }],
    [
      user,
      {
        ...answer,
        content: undefined,
        toolCalls: [{ id: "c", name: "get_skill", arguments: "{}" }],
      },
    ],
  ])(
    "never turns an inconsistent or incomplete source into ordinary Workspace context: %j",
    (...messages) => {
      expect(getAgentSourceSqlEditorContext(messages, answer.id)).toEqual({
        kind: "pending",
      });
    }
  );
});

describe("SQL Editor snapshots", () => {
  test("retains a selected Project without claiming a database", () => {
    const snapshot = { version: 1, project, tab };
    expect(normalizeAgentSqlEditorSnapshot(snapshot)).toEqual(snapshot);
  });

  test("retains verified Workspace context without selecting a Project or database", () => {
    expect(normalizeAgentSqlEditorSnapshot({ version: 1 })).toEqual({
      version: 1,
    });
  });

  test("does not restore a database connection as an unscoped Workspace snapshot", () => {
    const malformed = { version: 1 as const, connection };
    expect(normalizeAgentSqlEditorSnapshot(malformed)).toBeUndefined();
    expect(
      normalizeAgentChatSnapshot({ ...page, sqlEditor: malformed })
    ).toBeUndefined();
    expect(
      getAgentSourceSqlEditorContext(
        [
          {
            id: "user",
            chatId: "chat",
            role: "user",
            createdTs: 1,
            metadata: { sqlEditor: malformed },
          },
          {
            id: "answer",
            chatId: "chat",
            role: "assistant",
            createdTs: 2,
            content: "An answer",
            metadata: { sqlEditor: malformed },
          },
        ],
        "answer"
      )
    ).toEqual({ kind: "pending" });
  });

  test.each([
    "instances/demo/databases/example",
    "projects/example/instances/demo/databases/example",
  ])("preserves the full database name %s", (database) => {
    const snapshot = {
      version: 1,
      project,
      tab,
      connection: { ...connection, database },
      viewerObject: { kind: "table", name: "members", schema: "tenant" },
      batchActive: true,
      rowLimit: 100,
      queryOptions: { explainFormat: 0 },
    };
    expect(normalizeAgentSqlEditorSnapshot(snapshot)).toEqual(snapshot);
  });

  test.each([
    { version: 2 },
    { version: 1, project: "" },
    { version: 1, project: "projects/-" },
    { version: 1, project: "projects/example/extra" },
    { version: 1, tab: { id: "tab", mode: "UNKNOWN" } },
    { version: 1, rowLimit: -1 },
    { version: 1, batchActive: "false" },
    { version: 1, queryOptions: { explainFormat: 999 } },
    { version: 1, connection: { ...connection, engine: 9999 } },
    { version: 1, connection: { ...connection, engine: "POSTGRES" } },
    { version: 1, connection: { ...connection, engine: 0 } },
    { version: 1, connection: { ...connection, database: "example" } },
    {
      version: 1,
      connection: { ...connection, database: "instances/demo/databases/-" },
    },
    { version: 1, project: "projects/other", connection },
    {
      version: 1,
      project,
      connection: {
        ...connection,
        database: "projects/other/instances/demo/databases/example",
      },
    },
  ])("rejects malformed or inconsistent SQL context: %j", (snapshot) => {
    expect(normalizeAgentSqlEditorSnapshot(snapshot)).toBeUndefined();
    expect(
      normalizeAgentChatSnapshot({ ...page, sqlEditor: snapshot })
    ).toBeUndefined();
  });

  test("binds editor target provenance to the exact tab and connection", () => {
    const editorTarget = {
      actor: "users/member@example.com",
      workspace: "workspaces/example",
      hostId: "mount-1",
      tabId: tab.id,
      modelUri: "inmemory://sql/example",
      modelVersion: 2,
      connection,
      selection: { start: 10, end: 13 },
    };
    const snapshot = { version: 1, project, tab, connection, editorTarget };
    expect(normalizeAgentSqlEditorSnapshot(snapshot)).toEqual(snapshot);
    for (const patch of [
      { tabId: "tab-2" },
      { modelVersion: 0 },
      { selection: { start: 0.25, end: 13 } },
      { selection: { start: 13, end: 10 } },
      { connection: { ...connection, querySchema: "other" } },
      { connection: { ...connection, container: "other" } },
      {
        connection: {
          ...connection,
          database: "projects/example/instances/demo/databases/example",
        },
      },
    ])
      expect(
        normalizeAgentSqlEditorSnapshot({
          ...snapshot,
          editorTarget: { ...editorTarget, ...patch },
        })
      ).toBeUndefined();
  });

  test("allows a nonexecuting editor target for a disconnected Saved Query", () => {
    const snapshot = {
      version: 1,
      project,
      tab,
      editorTarget: {
        actor: "users/member@example.com",
        workspace: "workspaces/example",
        hostId: "mount-1",
        tabId: tab.id,
        modelUri: "inmemory://sql/example",
        modelVersion: 2,
        selection: { start: 0, end: 0 },
      },
    };
    expect(normalizeAgentSqlEditorSnapshot(snapshot)).toEqual(snapshot);
    expect(
      normalizeAgentSqlEditorSnapshot({ ...snapshot, connection })
    ).toBeUndefined();
  });

  test("preserves the exact SQL at the 32,768-byte boundary and rejects larger attachments", () => {
    const sql = " ".concat("\u00e9".repeat(16_382), "\r\n ");
    expect(new TextEncoder().encode(sql)).toHaveLength(32_768);
    const snapshot = {
      version: 1,
      project,
      attachment: { kind: "selection", sql },
    };
    expect(normalizeAgentSqlEditorSnapshot(snapshot)).toEqual(snapshot);
    expect(
      normalizeAgentSqlEditorSnapshot({
        ...snapshot,
        attachment: { ...snapshot.attachment, sql: `${sql} ` },
      })
    ).toBeUndefined();
  });

  test("rejects SQL text with no lossless UTF-8 representation", () => {
    expect(
      normalizeAgentSqlEditorSnapshot({
        version: 1,
        attachment: { kind: "selection", sql: "'\ud800'" },
      })
    ).toBeUndefined();
  });

  test("accepts an old page record without converting a malformed SQL field into one", () => {
    expect(normalizeAgentChatSnapshot({ path: "/", title: "Home" })).toEqual({
      path: "/",
      title: "Home",
    });
    expect(
      normalizeAgentChatSnapshot({ ...page, sqlEditor: null })
    ).toBeUndefined();
  });
});
