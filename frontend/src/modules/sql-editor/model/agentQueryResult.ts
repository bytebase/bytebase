import type { ConnectError } from "@connectrpc/connect";
import { v4 as uuidv4 } from "uuid";
import { queryErrorResultSet, queryResultSet } from "@/stores/app/sql";
import type { SQLEditorQueryParams } from "@/types";
import type { QueryResponse } from "@/types/proto-es/v1/sql_service_pb";
import { useSQLEditorStore } from "../store";
import { getSQLEditorTabsState } from "../store/tab";
import { sqlEditorEvents } from "./events";

export type AgentSqlQueryObservation =
  | { kind: "response"; response: QueryResponse; beginTimestampMS: number }
  | { kind: "error"; error: ConnectError; beginTimestampMS: number }
  | { kind: "uncertain" };

export function captureAgentQueryResultTarget(
  tabId: string,
  database: string,
  project: string
) {
  const tab = getSQLEditorTabsState().tabsById.get(tabId);
  if (
    !tab ||
    tab.id !== getSQLEditorTabsState().currentTabId ||
    tab.mode !== "SAVED_QUERY" ||
    tab.connection.database !== database
  )
    return;
  const generation = tab.databaseQueryContexts;
  return (
    params: SQLEditorQueryParams,
    observation: AgentSqlQueryObservation
  ) => {
    const tabs = getSQLEditorTabsState();
    const current = tabs.tabsById.get(tabs.currentTabId);
    if (
      observation.kind === "uncertain" ||
      current?.id !== tabId ||
      current.mode !== "SAVED_QUERY" ||
      current.connection.database !== database ||
      current.databaseQueryContexts !== generation
    )
      return false;
    const previous = generation?.get(database) ?? [];
    if (
      previous.some(
        (entry) => entry.status === "EXECUTING" || entry.status === "PENDING"
      )
    )
      return false;
    const contexts = new Map(generation);
    contexts.set(database, [
      {
        id: uuidv4(),
        params,
        status: "DONE",
        beginTimestampMS: observation.beginTimestampMS,
        resultSet:
          observation.kind === "response"
            ? queryResultSet(observation.response)
            : queryErrorResultSet(observation.error),
      },
      ...previous.slice(0, 49),
    ]);
    tabs.updateTab(tabId, { databaseQueryContexts: contexts });
    void useSQLEditorStore
      .getState()
      .mergeLatest({ project, database })
      .catch(() => {})
      .finally(() => {
        void sqlEditorEvents.emit("query-executed", { project, database });
      });
    return true;
  };
}
