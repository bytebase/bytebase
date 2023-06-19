import Emittery from "emittery";
import { Observable } from "rxjs";
import { ComputedRef } from "vue";
import { AdminExecuteResponse } from "../proto/v1/sql_service";
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
 *   -(has a)> QueryStreaming
 *   -(has a)> QueryTimer
 * a StreamingQueryController
 *   -(has a)> QueryEvents
 *   -(has a)> input stream
 *   -(has a)> response stream
 *   -(has a)> streaming AdminExecute connection
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
  response: AdminExecuteResponse;
}>;

export type StreamingQueryController = {
  events: QueryEvents;
  input$: Observable<WebTerminalQueryParamsV1>;
  response$: Observable<AdminExecuteResponse>;
  abort(reason?: any): void;
};

export type WebTerminalQueryState = {
  tab: TabInfo;
  queryItemList: WebTerminalQueryItemV1[];
  controller: StreamingQueryController;
  timer: QueryTimer;
};
