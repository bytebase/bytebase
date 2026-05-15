import { create, fromJson, toJson } from "@bufbuild/protobuf";
import { Code, ConnectError } from "@connectrpc/connect";
import Emittery from "emittery";
import { cloneDeep } from "lodash-es";
import type { Subscription } from "rxjs";
import { fromEventPattern, Observable } from "rxjs";
import { markRaw, ref } from "vue";
import { useCancelableTimeout } from "@/composables/useCancelableTimeout";
import { refreshTokens } from "@/connect/refreshToken";
import { useSQLEditorVueState } from "@/react/stores/sqlEditor/editor-vue-state";
import { pushNotification, useDatabaseV1Store } from "@/store";
import type {
  SQLEditorQueryParams,
  SQLEditorTab,
  SQLResultSetV1,
  StreamingQueryController,
} from "@/types";
import type {
  AdminExecuteRequest,
  AdminExecuteResponse,
  QueryResult,
} from "@/types/proto-es/v1/sql_service_pb";
import {
  AdminExecuteRequestSchema,
  AdminExecuteResponseSchema,
  QueryResult_Message_Level,
  QueryResultSchema,
} from "@/types/proto-es/v1/sql_service_pb";
import {
  extractGrpcErrorMessage,
  getErrorCode as extractGrpcStatusCode,
} from "@/utils/connect";
import { sqlEditorEvents } from "@/views/sql-editor/events";
import { useSQLEditorStore as useSQLEditorReactStore } from "./index";

const ENDPOINT = "/v1:adminExecute";
const SIG_ABORT = 3000 + Code.Aborted;
const QUERY_TIMEOUT_MS = 5000;

/**
 * Per-tab admin-mode streaming session. The `tab` + `timer` + `controller`
 * fields are framework-agnostic mutable services (Emittery / RxJS /
 * `useCancelableTimeout`); the query items themselves live in the zustand
 * `webTerminalQueryItemsByTabId` slice so React consumers re-render via
 * selectors instead of `useVueState` on a Vue ref.
 */
export interface WebTerminalQuerySession {
  tab: SQLEditorTab;
  timer: ReturnType<typeof useCancelableTimeout>;
  controller: StreamingQueryController;
}

const sessions = new Map<string, WebTerminalQuerySession>();

export const getWebTerminalQuerySession = (
  tab: SQLEditorTab
): WebTerminalQuerySession => {
  const existed = sessions.get(tab.id);
  if (existed) return existed;
  const session: WebTerminalQuerySession = {
    tab,
    timer: markRaw(useCancelableTimeout(QUERY_TIMEOUT_MS)),
    controller: createStreamingQueryController(),
  };
  sessions.set(tab.id, session);
  useSQLEditorReactStore.getState().ensureWebTerminalQueryState(tab.id);
  bindStreamingLogic(session);
  return session;
};

export const disposeWebTerminalQuerySession = (tabId: string): void => {
  sessions.delete(tabId);
  useSQLEditorReactStore.getState().clearWebTerminalQueryState(tabId);
};

const createStreamingQueryController = () => {
  const status: StreamingQueryController["status"] = ref("DISCONNECTED");
  const events: StreamingQueryController["events"] = markRaw(new Emittery());
  const input$ = fromEventPattern<SQLEditorQueryParams>(
    (handler) => events.on("query", handler),
    (handler) => events.off("query", handler)
  );

  const $ws = ref<WebSocket>();

  const controller: StreamingQueryController = {
    status,
    events,
    abort() {
      // noop here. will be overwritten after connected
    },
  };

  events.on("query", async (params) => {
    const request = mapRequest(params);
    console.debug("query", request);

    if (status.value === "DISCONNECTED") {
      // Refresh the access-token cookie before opening a new WebSocket.
      // The cookie may have expired since the last connection closed.
      await refreshTokens().catch(() => {});
      $ws.value = connect(request);
    }
  });

  const connect = (
    initialRequest: AdminExecuteRequest | undefined = undefined
  ) => {
    const abortController = new AbortController();

    const url = new URL(`${window.location.origin}${ENDPOINT}`);
    url.protocol = url.protocol.replace(/^http/, "ws");
    const ws = new WebSocket(url);
    status.value = "CONNECTED";

    const send = (request: AdminExecuteRequest) => {
      const payload = toJson(AdminExecuteRequestSchema, request);
      console.debug("will send", JSON.stringify(payload));
      ws.send(JSON.stringify(payload));
    };

    const response$ = new Observable<AdminExecuteResponse>((subscriber) => {
      let requestSubscription: Subscription;

      ws.addEventListener("open", () => {
        console.debug("ws open");
        if (initialRequest) {
          send(initialRequest);
        }
        requestSubscription = input$.subscribe({
          next(params) {
            send(mapRequest(params));
          },
        });
      });
      ws.addEventListener("message", (event) => {
        try {
          const data = JSON.parse(event.data);
          if (data.result) {
            const response = fromJson(AdminExecuteResponseSchema, data.result);
            subscriber.next(response);
          } else if (data.error) {
            const err = new ConnectError(data.error.message, data.error.code);
            subscriber.error(err);
          }
        } catch (err) {
          subscriber.error(err);
        }
      });
      ws.addEventListener("error", (event) => {
        console.debug("error", event);
        subscriber.error(
          new ConnectError("Internal server error", Code.Internal)
        );
      });
      ws.addEventListener("close", (event) => {
        console.debug("ws close", event.wasClean, event.reason, event.code);
        if (event.code === SIG_ABORT) {
          subscriber.error(new ConnectError(event.reason, Code.Aborted));
          return;
        }
        subscriber.error(
          new ConnectError(
            `Closed by server ${event.code}`,
            Code.DeadlineExceeded
          )
        );
      });

      return () => {
        console.debug("teardown");
        if (requestSubscription) {
          requestSubscription.unsubscribe();
        }
      };
    });

    abortController.signal.addEventListener("abort", (e) => {
      console.debug("abort", e);
      ws.close(SIG_ABORT, abortController.signal.reason);
    });
    controller.abort = abortController.abort.bind(abortController);

    response$.subscribe({
      next(response) {
        response.results.forEach((result) => {
          if (!result.error && result.columnNames.length === 0) {
            Object.assign(result, mockAffectedV1Rows0());
          }
        });
        events.emit("result", {
          error: "",
          ...response,
        });
      },
      error(error) {
        console.debug("subscribe error", error);

        const result: SQLResultSetV1 = {
          error: extractGrpcErrorMessage(error),
          status: extractGrpcStatusCode(error),
          results: [],
        };
        if (result.status === Code.Aborted && !result.error) {
          result.error = "Aborted";
        }

        events.emit("result", result);
        status.value = "DISCONNECTED";
      },
    });

    return ws;
  };

  return controller;
};

const bindStreamingLogic = (session: WebTerminalQuerySession) => {
  const tabId = session.tab.id;

  const activeItem = () => {
    const list =
      useSQLEditorReactStore.getState().webTerminalQueryItemsByTabId[tabId] ??
      [];
    return list[list.length - 1];
  };

  session.controller.events.on("query", (input) => {
    session.timer.start();
    const tail = activeItem();
    if (!tail) return;
    useSQLEditorReactStore
      .getState()
      .updateWebTerminalQueryItem(tabId, tail.id, {
        params: cloneDeep(input),
        status: "RUNNING",
      });
  });

  session.controller.events.on("result", (resultSet) => {
    // The tab may have been closed mid-flight; disposeWebTerminalQuerySession
    // dropped our session entry, but the WebSocket is still bound and this
    // handler is still subscribed. Bail before pushing a new query item,
    // which would resurrect store state for a tab that no longer exists.
    if (!sessions.has(tabId)) return;
    console.debug("event resultSet", resultSet);
    const tail = activeItem();
    if (tail) {
      useSQLEditorReactStore
        .getState()
        .updateWebTerminalQueryItem(tabId, tail.id, { resultSet });
    }
    for (const result of resultSet.results) {
      for (const message of result.messages) {
        pushNotification({
          module: "bytebase",
          style: "INFO",
          title: QueryResult_Message_Level[message.level],
          description: message.content,
        });
      }
    }
    // Admin-mode queries don't go through `useExecuteSQL`, so mirror its
    // post-exec history refresh here. `mergeLatest` prepends the
    // just-run statement without resetting the user's pagination.
    const database = tail?.params?.connection.database;
    if (database) {
      useSQLEditorReactStore
        .getState()
        .mergeLatest({
          project: useSQLEditorVueState().project,
          database,
        })
        .catch(() => {
          /* nothing */
        })
        .finally(() => {
          void sqlEditorEvents.emit("query-executed");
        });
    } else {
      void sqlEditorEvents.emit("query-executed");
    }
    // Finish the current item and append a fresh one for the next prompt.
    const finishedTail = activeItem();
    if (finishedTail) {
      useSQLEditorReactStore
        .getState()
        .updateWebTerminalQueryItem(tabId, finishedTail.id, {
          status: "FINISHED",
        });
    }
    session.timer.stop();
    useSQLEditorReactStore.getState().pushWebTerminalQueryItem(tabId);
  });
};

export const mockAffectedV1Rows0 = (): QueryResult => {
  return create(QueryResultSchema, {
    columnNames: ["Affected Rows"],
    columnTypeNames: ["BIGINT"],
    masked: [],
    error: "",
    statement: "",
    rows: [
      {
        values: [
          {
            kind: {
              case: "int64Value",
              value: BigInt(0),
            },
          },
        ],
      },
    ],
  });
};

const mapRequest = (params: SQLEditorQueryParams): AdminExecuteRequest => {
  const { connection, statement, explain } = params;

  const database = useDatabaseV1Store().getDatabaseByName(connection.database);
  const request = create(AdminExecuteRequestSchema, {
    name: database.name,
    statement: explain ? `EXPLAIN ${statement}` : statement,
    schema: connection.schema,
  });
  return request;
};
