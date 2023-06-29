import Emittery from "emittery";
import { ComputedRef, Ref } from "vue";
import { ExecuteConfig, ExecuteOption, TabInfo } from "../tab";
import { SQLResultSetV1 } from "./sql";

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

export type WebTerminalQueryParamsV1 = {
  query: string;
  config: ExecuteConfig;
  option?: Partial<ExecuteOption>;
};

export type WebTerminalQueryItemV1 = {
  id: string;
  sql: string;
  params?: WebTerminalQueryParamsV1;
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
  query: WebTerminalQueryParamsV1;
  result: SQLResultSetV1;
}>;

export type StreamingQueryController = {
  status: Ref<"CONNECTED" | "DISCONNECTED">;
  events: QueryEvents;
  abort(reason?: any): void;
};

export type WebTerminalQueryState = {
  tab: TabInfo;
  queryItemList: Ref<WebTerminalQueryItemV1[]>;
  controller: StreamingQueryController;
  timer: QueryTimer;
};
