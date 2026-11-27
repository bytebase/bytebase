import { createContextValues } from "@connectrpc/connect";
import { isEqual } from "lodash-es";
import { useEffect, useMemo, useState } from "react";
import { aiServiceClientConnect } from "@/api";
import { silentContextKey } from "@/api/context-key";
import { useAppStore } from "@/stores/app";
import { extractDatabaseResourceName } from "@/utils/v1/database";
import type { AgentSqlConnectionSnapshot } from "../logic/sqlEditorContext";
import { getCurrentAgentSqlEditorContext } from "../logic/sqlEditorHost";
import {
  buildSqlEditorSuggestionMessages,
  parseSqlEditorSuggestions,
} from "../logic/sqlEditorSuggestions";
import { useAgentStore } from "../store/agent";

export interface AgentSqlSuggestionSource {
  workspaceName: string;
  userEmail: string;
  connection: AgentSqlConnectionSnapshot;
}

export interface AgentSqlDatabaseSuggestions {
  source: AgentSqlSuggestionSource;
  questions: string[];
}

export type AgentSqlSuggestionsState =
  | {
      status: "idle" | "loading" | "fallback";
      source?: never;
      questions?: never;
    }
  | ({ status: "ready" } & AgentSqlDatabaseSuggestions);

const LOADING_TIMEOUT_MS = 15_000;
const requests = new Map<
  string,
  {
    scope: string;
    pending: Promise<string[] | undefined>;
    questions?: string[];
  }
>();
let cacheRevision = 0;

const appIdentity = (state: ReturnType<typeof useAppStore.getState>) =>
  JSON.stringify([
    state.workspace?.name,
    state.currentUser?.name,
    state.currentUser?.workspace,
    state.currentUser?.email,
  ]);
const agentIdentity = (state: ReturnType<typeof useAgentStore.getState>) =>
  JSON.stringify([state.workspaceName, state.userEmail]);
const invalidateRequests = () => {
  requests.clear();
  cacheRevision++;
};
let observingIdentity = false;
function observeIdentity() {
  if (observingIdentity) return;
  observingIdentity = true;
  // Keep invalidating after the landing unmounts, including a switch back to
  // the same account after its old requests were dispatched.
  useAppStore.subscribe((next, previous) => {
    if (appIdentity(next) !== appIdentity(previous)) invalidateRequests();
  });
  useAgentStore.subscribe((next, previous) => {
    if (agentIdentity(next) !== agentIdentity(previous)) invalidateRequests();
  });
}

export function isAgentSqlSuggestionSourceCurrent(
  source: AgentSqlSuggestionSource
): boolean {
  const app = useAppStore.getState();
  const agent = useAgentStore.getState();
  if (
    app.workspace?.name !== source.workspaceName ||
    app.currentUser?.workspace !== source.workspaceName ||
    app.currentUser?.email !== source.userEmail ||
    agent.workspaceName !== source.workspaceName ||
    agent.userEmail !== source.userEmail
  )
    return false;
  try {
    const capture = getCurrentAgentSqlEditorContext();
    return (
      capture?.kind === "ready" &&
      capture.snapshot.project === source.connection.project &&
      isEqual(capture.snapshot.connection, source.connection)
    );
  } catch {
    return false;
  }
}

export function useSqlEditorSuggestions({
  enabled,
  workspaceName,
  userEmail,
  connection,
  locale,
}: {
  enabled: boolean;
  workspaceName?: string;
  userEmail?: string;
  connection?: AgentSqlConnectionSnapshot;
  locale: string;
}): AgentSqlSuggestionsState {
  useAppStore(appIdentity);
  useAgentStore(agentIdentity);
  const sourceKey =
    workspaceName && userEmail && connection
      ? JSON.stringify({ workspaceName, userEmail, connection })
      : undefined;
  const source = useMemo(
    () =>
      sourceKey
        ? (JSON.parse(sourceKey) as AgentSqlSuggestionSource)
        : undefined,
    [sourceKey]
  );
  const aiEnabled = useAppStore((state) => state.serverInfo?.aiEnabled);
  const sourceCurrent = !!source && isAgentSqlSuggestionSourceCurrent(source);
  const active = enabled && aiEnabled === true && sourceCurrent;
  const scopeKey = JSON.stringify([cacheRevision, sourceKey, locale]);
  const metadata = useAppStore((state) =>
    active && source
      ? state.getCachedDatabaseMetadata(source.connection.database)
      : undefined
  );
  const getMetadata = useAppStore((state) => state.getOrFetchDatabaseMetadata);
  const messages = useMemo(() => {
    if (!active || !source || !metadata) return undefined;
    const canonical = extractDatabaseResourceName(
      source.connection.database
    ).database;
    if (metadata.name !== `${canonical}/metadata`) return undefined;
    return buildSqlEditorSuggestionMessages(
      metadata,
      source.connection,
      locale
    );
  }, [active, source, metadata, locale]);
  const requestKey = messages
    ? JSON.stringify([scopeKey, messages])
    : undefined;
  const [failedScope, setFailedScope] = useState<string>();
  const [result, setResult] = useState<{
    key: string;
    value: AgentSqlDatabaseSuggestions;
  }>();
  const questions = requestKey
    ? (requests.get(requestKey)?.questions ??
      (result?.key === requestKey ? result.value.questions : undefined))
    : undefined;
  const state: AgentSqlSuggestionsState =
    !enabled || !sourceCurrent || aiEnabled === false
      ? { status: "idle" }
      : failedScope === scopeKey ||
          (metadata && !messages) ||
          questions?.length === 0
        ? { status: "fallback" }
        : questions && source
          ? { status: "ready", source, questions }
          : { status: "loading" };
  const status = state.status;

  useEffect(() => {
    if (active) observeIdentity();
  }, [active]);

  useEffect(() => {
    if (
      !active ||
      !source ||
      metadata ||
      failedScope === scopeKey ||
      !isAgentSqlSuggestionSourceCurrent(source)
    )
      return;
    let current = true;
    void getMetadata({
      database: source.connection.database,
      silent: true,
    }).catch(() => {
      if (current && isAgentSqlSuggestionSourceCurrent(source))
        setFailedScope(scopeKey);
    });
    return () => {
      current = false;
    };
  }, [active, source, metadata, getMetadata, failedScope, scopeKey]);

  useEffect(() => {
    if (
      !requestKey ||
      !source ||
      !messages ||
      failedScope === scopeKey ||
      !isAgentSqlSuggestionSourceCurrent(source)
    )
      return;
    let current = true;
    let entry = requests.get(requestKey);
    if (!entry) {
      entry = { scope: scopeKey, pending: Promise.resolve(undefined) };
      const request = entry;
      entry.pending = aiServiceClientConnect
        .chat(
          { messages },
          {
            timeoutMs: 15_000,
            contextValues: createContextValues().set(silentContextKey, true),
          }
        )
        .then((response) => parseSqlEditorSuggestions(response.content ?? ""))
        .catch(() => [])
        .then((questions) => {
          if (
            requests.get(requestKey) !== request ||
            !isAgentSqlSuggestionSourceCurrent(source)
          )
            return undefined;
          if (questions.length) request.questions = questions;
          else requests.delete(requestKey);
          return questions;
        });
      if (requests.size >= 8) {
        const oldest = requests.keys().next().value;
        if (oldest !== undefined) requests.delete(oldest);
      }
      requests.set(requestKey, entry);
    }
    void entry.pending.then((questions) => {
      if (
        current &&
        questions !== undefined &&
        isAgentSqlSuggestionSourceCurrent(source)
      ) {
        if (!questions.length) setFailedScope(scopeKey);
        setResult({ key: requestKey, value: { source, questions } });
      }
    });
    return () => {
      current = false;
    };
  }, [requestKey, source, messages, failedScope, scopeKey]);

  useEffect(() => {
    if (status !== "loading") return;
    const timer = setTimeout(() => {
      setFailedScope(scopeKey);
      for (const [key, entry] of requests) {
        if (entry.scope === scopeKey && !entry.questions) requests.delete(key);
      }
    }, LOADING_TIMEOUT_MS);
    return () => clearTimeout(timer);
  }, [status, scopeKey]);

  return state;
}
