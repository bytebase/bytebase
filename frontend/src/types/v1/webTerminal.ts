import type Emittery from "emittery";
import type { ComputedRef, Ref } from "vue";
import type { SQLEditorQueryParams, SQLEditorTab } from "../sqlEditor";
import type { SQLResultSetV1 } from "./sql";

/**
 * Model
 *
 * a Tab -(has a)> WebTerminalQueryState
 *   according to tab' id
 * a WebTerminalQueryState
 *   -(belongs to a)> Tab
 *   -(has a)> QueryItem[]
 *   -(has a)> QueryTimer
 * a StreamingQueryController
 *   -(has a)> QueryEvents
 *   -(has a)> status
 *   -(has a)> websocket streaming connection
 */

export type WebTerminalQueryItemV1 = {
  id: string;
  statement: string;
  params?: SQLEditorQueryParams;
  resultSet?: SQLResultSetV1;
  status: "IDLE" | "RUNNING" | "FINISHED";
};

export type QueryTimer = {
  start(): void;
  stop(): void;
  elapsedMS: ComputedRef<number>;
  expired: ComputedRef<boolean>;
};

export type QueryEvents = Emittery<{
  query: SQLEditorQueryParams;
  result: SQLResultSetV1;
}>;

export type StreamingQueryController = {
  status: Ref<"CONNECTED" | "DISCONNECTED">;
  events: QueryEvents;
  abort(reason?: unknown): void;
};

export type WebTerminalQueryState = {
  tab: SQLEditorTab;
  queryItemList: Ref<WebTerminalQueryItemV1[]>;
  controller: StreamingQueryController;
  timer: QueryTimer;
};
