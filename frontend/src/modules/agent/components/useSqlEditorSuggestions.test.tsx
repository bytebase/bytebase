import { create } from "@bufbuild/protobuf";
import { act, cleanup, renderHook, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, type MockInstance, test, vi } from "vitest";
import { aiServiceClientConnect } from "@/api";
import { silentContextKey } from "@/api/context-key";
import { useAppStore } from "@/stores/app";
import { AIChatResponseSchema } from "@/types/proto-es/v1/ai_service_pb";
import { Engine } from "@/types/proto-es/v1/common_pb";
import { DatabaseMetadataSchema } from "@/types/proto-es/v1/database_service_pb";
import { UserSchema } from "@/types/proto-es/v1/user_service_pb";
import { WorkspaceSchema } from "@/types/proto-es/v1/workspace_service_pb";
import type { AgentSqlConnectionSnapshot, AgentSqlEditorSnapshot } from "../logic/sqlEditorContext";
import { registerAgentSqlEditorHost } from "../logic/sqlEditorHost";
import { useAgentStore } from "../store/agent";
import { isAgentSqlSuggestionSourceCurrent, useSqlEditorSuggestions } from "./useSqlEditorSuggestions";

const workspaceName = "workspaces/default";
const userEmail = "member@example.com";
const connection: AgentSqlConnectionSnapshot = { database: "instances/main/databases/app", project: "projects/sample", engine: Engine.POSTGRES, querySchema: "public" };
const metadata = (database = connection.database) => create(DatabaseMetadataSchema, {
  name: `${database}/metadata`, schemas: [{ name: "public", tables: [{ name: "employees", columns: [{ name: "salary", type: "integer" }] }] }],
});
const response = (question = "What is the average employee salary?") => create(AIChatResponseSchema, { content: JSON.stringify({ questions: [question] }) });
const params = { enabled: true, workspaceName, userEmail, connection, locale: "en-US" };
let snapshot: AgentSqlEditorSnapshot;
let unregister: () => void;
let slot: HTMLDivElement;
let chat: MockInstance<typeof aiServiceClientConnect.chat>;
const originalFetch = useAppStore.getState().getOrFetchDatabaseMetadata;

beforeEach(() => {
  useAgentStore.setState({ userEmail: undefined });
  slot = document.createElement("div"); document.body.append(slot);
  snapshot = { version: 1, project: connection.project, connection: { ...connection } };
  useAgentStore.setState({ workspaceName, userEmail });
  useAppStore.setState({ workspace: create(WorkspaceSchema, { name: workspaceName }), currentUser: create(UserSchema, { email: userEmail, workspace: workspaceName }), serverInfo: { ...useAppStore.getState().serverInfo!, aiEnabled: true }, metadataByName: {} });
  unregister = registerAgentSqlEditorHost({ id: "sql", slot, mainColumn: slot, resizer: slot, paneRatio: 0.4, captureContext: () => ({ kind: "ready", snapshot }) });
  chat = vi.spyOn(aiServiceClientConnect, "chat").mockResolvedValue(response());
  useAppStore.setState({ getOrFetchDatabaseMetadata: vi.fn(async ({ database }) => {
    const value = metadata(database);
    useAppStore.setState(state => ({ metadataByName: { ...state.metadataByName, [`${database}/metadata::::0`]: value } }));
    return value;
  }) });
});

afterEach(() => { cleanup(); unregister(); slot.remove(); vi.restoreAllMocks(); vi.useRealTimers(); useAppStore.setState({ getOrFetchDatabaseMetadata: originalFetch }); });

test("generates schema-only questions using the existing AI service without tools", async () => {
  const { result } = renderHook(() => useSqlEditorSuggestions(params));
  expect(result.current?.status).toBe("loading");
  await waitFor(() => expect(result.current?.questions).toEqual(["What is the average employee salary?"]));
  expect(result.current?.source).toEqual({ workspaceName, userEmail, connection });
  expect(result.current?.status).toBe("ready");
  const [request, options] = chat.mock.calls[0];
  expect(request.messages?.map(message => message.content).join("\n")).toContain("employees");
  expect(request.toolDefinitions).toBeUndefined();
  expect(options?.contextValues?.get(silentContextKey)).toBe(true);
  expect(chat).toHaveBeenCalledTimes(1);
});

test.each(["disabled", "disconnected", "AI unavailable", "wrong owner"])("does not fetch or generate for %s", async reason => {
  if (reason === "AI unavailable") useAppStore.setState({ serverInfo: { ...useAppStore.getState().serverInfo!, aiEnabled: false } });
  if (reason === "wrong owner") useAgentStore.setState({ userEmail: "other@example.com" });
  const { result } = renderHook(() => useSqlEditorSuggestions({ ...params, enabled: reason !== "disabled", connection: reason === "disconnected" ? undefined : connection }));
  await act(async () => {});
  expect(result.current?.status).toBe("idle");
  expect(useAppStore.getState().getOrFetchDatabaseMetadata).not.toHaveBeenCalled();
  expect(chat).not.toHaveBeenCalled();
});

test.each(["metadata denied", "empty schema", "provider failure", "invalid response"])("silently retains fallback for %s", async reason => {
  if (reason === "metadata denied") useAppStore.setState({ getOrFetchDatabaseMetadata: vi.fn().mockRejectedValue(new Error("Denied")) });
  if (reason === "empty schema") useAppStore.setState({ metadataByName: { [`${connection.database}/metadata::::0`]: create(DatabaseMetadataSchema, { name: `${connection.database}/metadata` }) } });
  if (reason === "provider failure") chat.mockRejectedValue(new Error("Unavailable"));
  if (reason === "invalid response") chat.mockResolvedValue(create(AIChatResponseSchema, { content: "not JSON" }));
  const { result, rerender } = renderHook(() => useSqlEditorSuggestions(params));
  await act(async () => {});
  expect(result.current?.status).toBe("fallback");
  rerender();
  await act(async () => {});
  expect(chat.mock.calls.length).toBe(reason === "metadata denied" || reason === "empty schema" ? 0 : 1);
});

test("deduplicates generation across rerenders and clearing selection", async () => {
  const { result, rerender } = renderHook(({ enabled }) => useSqlEditorSuggestions({ ...params, enabled }), { initialProps: { enabled: true } });
  await waitFor(() => expect(result.current?.questions?.length).toBe(1));
  rerender({ enabled: false }); expect(result.current?.status).toBe("idle");
  rerender({ enabled: true });
  await waitFor(() => expect(result.current?.questions?.length).toBe(1));
  expect(chat).toHaveBeenCalledTimes(1);
});

test.each(["database", "schema", "project", "owner", "workspace", "disconnect"] as const)("discards a late completion after %s changes", async change => {
  let finish!: (value: ReturnType<typeof response>) => void;
  chat.mockImplementation(() => new Promise(resolve => { finish = resolve; }));
  const { result, rerender } = renderHook(value => useSqlEditorSuggestions(value), { initialProps: params as Parameters<typeof useSqlEditorSuggestions>[0] });
  await waitFor(() => expect(chat).toHaveBeenCalledTimes(1));
  const completeOld = finish;
  const next = { ...params, connection: { ...connection } as AgentSqlConnectionSnapshot | undefined };
  if (change === "owner") { next.userEmail = "other@example.com"; useAgentStore.setState({ userEmail: next.userEmail }); }
  else if (change === "workspace") { next.workspaceName = "workspaces/other"; useAgentStore.setState({ workspaceName: next.workspaceName }); }
  else if (change === "disconnect") { next.connection = undefined; snapshot = { version: 1, project: connection.project }; }
  else { next.connection = { ...connection, ...(change === "database" ? { database: "instances/main/databases/other" } : change === "schema" ? { querySchema: "private" } : { project: "projects/other" }) }; snapshot = { version: 1, project: next.connection.project, connection: next.connection }; }
  rerender(next);
  expect(result.current?.questions).toBeUndefined();
  await act(async () => completeOld(response("Old database question?")));
  expect(result.current?.questions ?? []).not.toContain("Old database question?");
});

test("invalidates immediately and regenerates on metadata and locale changes", async () => {
  const { result, rerender } = renderHook(({ locale }) => useSqlEditorSuggestions({ ...params, locale }), { initialProps: { locale: "en-US" } });
  await waitFor(() => expect(result.current?.questions?.length).toBe(1));
  chat.mockResolvedValue(response("How many departments are there?"));
  act(() => useAppStore.setState({ metadataByName: { [`${connection.database}/metadata::::0`]: create(DatabaseMetadataSchema, { name: `${connection.database}/metadata`, schemas: [{ name: "public", tables: [{ name: "departments", columns: [{ name: "id", type: "int" }] }] }] }) } }));
  await waitFor(() => expect(result.current?.questions).toEqual(["How many departments are there?"]));
  chat.mockResolvedValue(response("How many departments in another language?"));
  rerender({ locale: "ja-JP" }); expect(result.current?.status).toBe("loading");
  await waitFor(() => expect(result.current?.questions).toEqual(["How many departments in another language?"]));
  expect(chat).toHaveBeenCalledTimes(3);
});

describe("suggestion source freshness", () => {
  test("rejects an old source even before React rerenders", () => {
    const source = { workspaceName, userEmail, connection: { ...connection } };
    expect(isAgentSqlSuggestionSourceCurrent(source)).toBe(true);
    snapshot.connection = { ...connection, database: "instances/main/databases/other" };
    expect(isAgentSqlSuggestionSourceCurrent(source)).toBe(false);
    snapshot.connection = connection;
    useAppStore.setState({ currentUser: create(UserSchema, { email: "other@example.com", workspace: workspaceName }) });
    expect(isAgentSqlSuggestionSourceCurrent(source)).toBe(false);
  });
});

test("reuses ready suggestions synchronously across remount without another provider call", async () => {
  const first = renderHook(() => useSqlEditorSuggestions(params));
  await waitFor(() => expect(first.result.current?.status).toBe("ready"));
  first.unmount();
  const reopened = renderHook(() => useSqlEditorSuggestions(params));
  expect(reopened.result.current).toMatchObject({ status: "ready", questions: ["What is the average employee salary?"] });
  expect(chat).toHaveBeenCalledTimes(1);
});

test("deduplicates in-flight generation across remount and stays loading until ready", async () => {
  let finish!: (value: ReturnType<typeof response>) => void;
  chat.mockImplementation(() => new Promise(resolve => { finish = resolve; }));
  const first = renderHook(() => useSqlEditorSuggestions(params));
  await waitFor(() => expect(chat).toHaveBeenCalledTimes(1));
  expect(first.result.current?.status).toBe("loading");
  first.unmount();
  const reopened = renderHook(() => useSqlEditorSuggestions(params));
  expect(reopened.result.current?.status).toBe("loading");
  await act(async () => finish(response()));
  expect(reopened.result.current?.status).toBe("ready");
  expect(chat).toHaveBeenCalledTimes(1);
});

test.each(["database", "schema", "project", "locale", "metadata"] as const)("does not reuse ready suggestions after %s changes on remount", async change => {
  const first = renderHook(() => useSqlEditorSuggestions(params));
  await waitFor(() => expect(first.result.current?.status).toBe("ready"));
  first.unmount();
  const next = { ...params, connection: { ...connection } };
  if (change === "locale") next.locale = "ja-JP";
  else if (change === "metadata") useAppStore.setState({ metadataByName: { [`${connection.database}/metadata::::0`]: create(DatabaseMetadataSchema, { name: `${connection.database}/metadata`, schemas: [{ name: "public", tables: [{ name: "departments" }] }] }) } });
  else next.connection = { ...connection, ...(change === "database" ? { database: "instances/main/databases/other" } : change === "schema" ? { querySchema: "other" } : { project: "projects/other" }) };
  if (change === "schema") useAppStore.setState({ metadataByName: { [`${connection.database}/metadata::::0`]: create(DatabaseMetadataSchema, { name: `${connection.database}/metadata`, schemas: [{ name: "other", tables: [{ name: "departments" }] }] }) } });
  snapshot = { version: 1, project: next.connection.project, connection: next.connection };
  let finish!: (value: ReturnType<typeof response>) => void;
  chat.mockImplementation(() => new Promise(resolve => { finish = resolve; }));
  const reopened = renderHook(() => useSqlEditorSuggestions(next));
  expect(reopened.result.current?.status).toBe("loading");
  expect(reopened.result.current?.questions).toBeUndefined();
  await waitFor(() => expect(chat).toHaveBeenCalledTimes(2));
  await act(async () => finish(response("A new scoped question?")));
  expect(reopened.result.current?.questions).toEqual(["A new scoped question?"]);
});

test.each(["agent", "app", "workspace"])("invalidates successful cache when %s identity changes while the landing is unmounted", async identity => {
  const first = renderHook(() => useSqlEditorSuggestions(params));
  await waitFor(() => expect(first.result.current?.status).toBe("ready"));
  first.unmount();
  if (identity === "agent") {
    useAgentStore.setState({ userEmail: "other@example.com" });
    useAgentStore.setState({ userEmail });
  } else if (identity === "app") {
    const currentUser = useAppStore.getState().currentUser;
    useAppStore.setState({ currentUser: create(UserSchema, { name: "users/other", email: "other@example.com", workspace: workspaceName }) });
    useAppStore.setState({ currentUser });
  } else {
    const workspace = useAppStore.getState().workspace;
    useAppStore.setState({ workspace: create(WorkspaceSchema, { name: "workspaces/other" }) });
    useAppStore.setState({ workspace });
  }
  chat.mockImplementation(() => new Promise(() => {}));
  const reopened = renderHook(() => useSqlEditorSuggestions(params));
  expect(reopened.result.current?.status).toBe("loading");
  expect(chat).toHaveBeenCalledTimes(2);
});

test("uses one loading deadline across metadata and AI, ignoring late completions", async () => {
  vi.useFakeTimers();
  let finishMetadata!: () => void;
  useAppStore.setState({ getOrFetchDatabaseMetadata: vi.fn(() => new Promise<ReturnType<typeof metadata>>(resolve => {
    finishMetadata = () => {
      const value = metadata();
      useAppStore.setState({ metadataByName: { [`${connection.database}/metadata::::0`]: value } });
      resolve(value);
    };
  })) });
  let finishAI!: (value: ReturnType<typeof response>) => void;
  chat.mockImplementation(() => new Promise(resolve => { finishAI = resolve; }));
  const { result } = renderHook(() => useSqlEditorSuggestions(params));
  expect(result.current?.status).toBe("loading");
  await act(async () => { await vi.advanceTimersByTimeAsync(14_000); finishMetadata(); });
  expect(chat).toHaveBeenCalledTimes(1);
  expect(result.current?.status).toBe("loading");
  await act(async () => { await vi.advanceTimersByTimeAsync(1_000); });
  expect(result.current?.status).toBe("fallback");
  await act(async () => finishAI(response("A late question?")));
  expect(result.current?.status).toBe("fallback");
  expect(result.current?.questions).toBeUndefined();
});

test("hung metadata falls back and a late cache update does not start AI", async () => {
  vi.useFakeTimers();
  useAppStore.setState({ getOrFetchDatabaseMetadata: vi.fn(() => new Promise<ReturnType<typeof metadata>>(() => {})) });
  const { result } = renderHook(() => useSqlEditorSuggestions(params));
  await act(async () => { await vi.advanceTimersByTimeAsync(15_000); });
  expect(result.current?.status).toBe("fallback");
  act(() => useAppStore.setState({ metadataByName: { [`${connection.database}/metadata::::0`]: metadata() } }));
  expect(result.current?.status).toBe("fallback");
  expect(chat).not.toHaveBeenCalled();
});

test("provider failure is retried on a later opening, not cached as a successful result", async () => {
  chat.mockRejectedValueOnce(new Error("Unavailable"));
  const first = renderHook(() => useSqlEditorSuggestions(params));
  await waitFor(() => expect(first.result.current?.status).toBe("fallback"));
  first.unmount();
  const reopened = renderHook(() => useSqlEditorSuggestions(params));
  await waitFor(() => expect(reopened.result.current?.status).toBe("ready"));
  expect(chat).toHaveBeenCalledTimes(2);
});

test("terminal provider fallback does not refresh after unchanged metadata is reloaded", async () => {
  chat.mockRejectedValueOnce(new Error("Unavailable"));
  const { result } = renderHook(() => useSqlEditorSuggestions(params));
  await waitFor(() => expect(result.current.status).toBe("fallback"));
  await act(async () => useAppStore.setState({ metadataByName: { [`${connection.database}/metadata::::0`]: metadata() } }));
  expect(chat).toHaveBeenCalledTimes(1);
  expect(result.current.status).toBe("fallback");
});
