import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { isEqual } from "lodash-es";
import { v4 as uuidv4 } from "uuid";
import { databaseServiceClientConnect } from "@/api";
import { silentContextKey } from "@/api/context-key";
import type {
  AgentSqlQueryObservation,
  captureAgentQueryResultTarget,
} from "@/modules/sql-editor/model/agentQueryResult";
import { State } from "@/types/proto-es/v1/common_pb";
import { GetDatabaseRequestSchema } from "@/types/proto-es/v1/database_service_pb";
import { QueryOptionSchema } from "@/types/proto-es/v1/sql_service_pb";
import { useAgentStore } from "../store/agent";
import { priorUncertainApiAttemptIds } from "./apiAttempt";
import { notSentApiResult } from "./callApiResult";
import { supportsAgentRunRecovery } from "./runOwnership";
import type { AgentSqlEditorSnapshot } from "./sqlEditorContext";
import {
  type AgentSqlActionSource,
  bindAgentSqlBlockActions,
  canBeginAgentSqlResponseAction,
  getCurrentAgentSqlActionSource,
  isAgentSqlActionChat,
  isAgentSqlExecutionTargetCurrent,
} from "./sqlResponse";
import { admitPreparedAgentApiCall } from "./tools/apiAdmission";
import { type CallApiArgs, prepareApiCall } from "./tools/callApi";
import type { AgentChatSnapshot, ToolExecutionResult } from "./types";

export interface AgentSqlExecutionGuard {
  isCurrent(): boolean;
  validateBeforeSend(): Promise<boolean>;
  onQueryObservation?(observation: AgentSqlQueryObservation): void;
}

export function buildAgentSqlRequest(
  snapshot: AgentSqlEditorSnapshot,
  sql: string
): CallApiArgs | undefined {
  const c = snapshot.connection;
  if (
    snapshot.tab?.mode !== "SAVED_QUERY" ||
    snapshot.batchActive ||
    !snapshot.project ||
    !c ||
    c.project !== snapshot.project ||
    snapshot.rowLimit === undefined
  )
    return;
  return {
    operationId: "bytebase.v1.SQLService.Query",
    body: {
      name: c.database,
      statement: sql,
      schema: c.querySchema,
      container: c.container,
      dataSourceId: c.dataSourceId,
      queryOption: snapshot.queryOptions,
      limit: snapshot.rowLimit,
    },
  };
}

// The owner invokes this under its existing chat lease. The ephemeral target and
// permission below cannot be reconstructed by replaying the persisted transcript.
export async function beginSqlResponseAction(
  source: AgentSqlActionSource,
  snapshot: AgentSqlEditorSnapshot,
  page: AgentChatSnapshot,
  memberLabel: string,
  publish: ReturnType<typeof captureAgentQueryResultTarget>
): Promise<boolean> {
  const { chatId } = source;
  const conn = snapshot.connection;
  const target = snapshot.editorTarget;
  const request = buildAgentSqlRequest(snapshot, source.sql);
  if (!request || !conn || !target || !supportsAgentRunRecovery()) return false;
  const store = useAgentStore.getState();
  const scope = store.captureApiScope();
  const idle = () =>
    scope() &&
    canBeginAgentSqlResponseAction(source, snapshot) &&
    Boolean(getCurrentAgentSqlActionSource(source));
  store.syncState();
  if (!idle()) return false;
  if (!(await store.checkAgentStorageAdmission()) || !idle()) return false;
  const executionOrigin = {
    kind: "sql-response" as const,
    messageId: source.messageId,
    blockIndex: source.blockIndex,
    sourceHash: source.sourceHash,
  };
  const root = store.addMessage({
    chatId,
    role: "user",
    content: memberLabel,
    metadata: {
      route: page.path,
      sqlEditor: snapshot,
      sqlExecutionAction: executionOrigin,
    },
  });
  if (!root) return false;
  const runId = await store.beginChatRun(chatId, root.id, {
    context: conn.project,
    page: { ...page, sqlEditor: snapshot },
    runMode: "default",
    references: [],
    executionOrigin,
  });
  if (!runId) return false;
  const matchesRun = () => {
    const chat = useAgentStore.getState().getChat(chatId);
    return (
      scope() &&
      chat?.runId === runId &&
      isAgentSqlActionChat(chat, snapshot, store.workspaceName) &&
      chat.context === conn.project &&
      chat.runCheckpoint?.rootUserMessageId === root.id &&
      isEqual(
        useAgentStore
          .getState()
          .getMessages(chatId)
          .find((m) => m.id === root.id)?.metadata?.sqlExecutionAction,
        executionOrigin
      ) &&
      isEqual(chat.runCheckpoint.bindings.executionOrigin, executionOrigin)
    );
  };
  const isCurrent = () => {
    const state = useAgentStore.getState();
    const chat = state.getChat(chatId);
    if (
      !matchesRun() ||
      state.currentChatId !== chatId ||
      (chat?.status !== "running" && chat?.status !== "awaiting_user") ||
      !isAgentSqlExecutionTargetCurrent(source, snapshot)
    )
      return false;
    const messages = state.getMessages(chatId);
    const msg = messages.find((m) => m.id === source.messageId);
    return (
      !!msg &&
      bindAgentSqlBlockActions(msg, messages).some((x) => isEqual(source, x))
    );
  };
  const controller = new AbortController();
  const toolCallId = uuidv4();
  const settle = async (result?: string) => {
    if (!matchesRun()) return;
    if (result)
      store.addMessage({
        chatId,
        role: "tool",
        toolCallId,
        content: result,
        metadata: { runId, route: page.path },
      });
    store.setFollowUpQueuePaused(chatId, true);
    store.finishChatRun(chatId);
    await store.flushRunPersistence();
  };
  if (!isCurrent()) {
    await settle();
    return false;
  }
  store.setAbortController(chatId, controller);
  try {
    const prepared = await prepareApiCall(request);
    if (!matchesRun()) return false;
    let limit = snapshot.rowLimit;
    if (prepared.kind === "prepared")
      limit = JSON.parse(prepared.call.body).limit;
    const call = store.addMessage({
      chatId,
      role: "assistant",
      toolCalls: [
        {
          id: toolCallId,
          name: "call_api",
          arguments: JSON.stringify({
            ...request,
            body: { ...request.body, limit },
          }),
        },
      ],
      metadata: { route: page.path, runId },
    });
    if (!call) {
      await settle();
      return false;
    }
    if (controller.signal.aborted || !isCurrent()) {
      await settle(
        notSentApiResult(
          "The SQL source changed or the action stopped before admission.",
          "admission-rejected"
        )
      );
      return false;
    }
    if (prepared.kind === "error") {
      await settle(prepared.result);
      return false;
    }
    const sqlOrigin: AgentSqlExecutionGuard = {
      isCurrent,
      async validateBeforeSend() {
        if (!isCurrent()) return false;
        const db = await databaseServiceClientConnect.getDatabase(
          create(GetDatabaseRequestSchema, { name: conn.database }),
          { contextValues: createContextValues().set(silentContextKey, true) }
        );
        return (
          isCurrent() &&
          db.state === State.ACTIVE &&
          db.name === conn.database &&
          db.project === conn.project
        );
      },
      onQueryObservation(observation) {
        if (!isCurrent()) return;
        publish?.(
          {
            statement: source.sql,
            engine: conn.engine,
            explain: false,
            selection: null,
            limit,
            connection: {
              instance: conn.database.split("/databases/")[0],
              database: conn.database,
              schema: conn.querySchema,
              table: conn.container,
              dataSourceId: conn.dataSourceId,
            },
            queryOption: create(QueryOptionSchema, snapshot.queryOptions),
          },
          observation
        );
      },
    };
    const admitted: ToolExecutionResult = await admitPreparedAgentApiCall({
      call: prepared.call,
      toolCallId,
      chatId,
      userName: target.actor,
      signal: controller.signal,
      referencedProject: conn.project,
      diagnosticRequested: false,
      priorAttemptIds: (fp) =>
        priorUncertainApiAttemptIds(useAgentStore.getState().chats, fp),
      isScopeCurrent: scope,
      sqlOrigin,
    });
    if (
      admitted.kind !== "ask_user" ||
      !isCurrent() ||
      controller.signal.aborted
    ) {
      await settle(
        admitted.kind === "tool_result"
          ? admitted.result
          : notSentApiResult(
              "The SQL request could not be admitted.",
              "admission-rejected"
            )
      );
      return false;
    }
    store.awaitUser(chatId, admitted.ask);
    if (!(await store.flushRunPersistence())) {
      await settle(
        notSentApiResult(
          "The permission request could not be saved.",
          "admission-rejected"
        )
      );
      return false;
    }
    return true;
  } catch {
    await settle(
      notSentApiResult(
        "The SQL request could not be prepared.",
        "admission-rejected"
      )
    );
    return false;
  } finally {
    if (store.getAbortController(chatId) === controller)
      store.setAbortController(chatId, null);
  }
}
