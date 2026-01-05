import { create, fromJson, toJson } from "@bufbuild/protobuf";
import { Code, ConnectError } from "@connectrpc/connect";
import Emittery from "emittery";
import { cloneDeep, uniqueId } from "lodash-es";
import { defineStore } from "pinia";
import type { Subscription } from "rxjs";
import { fromEventPattern, map, Observable } from "rxjs";
import { markRaw, ref, shallowRef } from "vue";
import { useCancelableTimeout } from "@/composables/useCancelableTimeout";
import { pushNotification, useDatabaseV1Store } from "@/store";
import type {
  SQLEditorQueryParams,
  SQLEditorTab,
  SQLResultSetV1,
  StreamingQueryController,
  WebTerminalQueryItemV1,
  WebTerminalQueryState,
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

const ENDPOINT = "/v1:adminExecute";
const SIG_ABORT = 3000 + Code.Aborted;
const QUERY_TIMEOUT_MS = 5000;
const MAX_QUERY_ITEM_COUNT = 20;

export const useWebTerminalStore = defineStore("webTerminal", () => {
  const map = shallowRef(new Map<string, WebTerminalQueryState>());

  const getQueryStateByTab = (tab: SQLEditorTab) => {
    const existed = map.value.get(tab.id);
    if (existed) return existed;

    const qs = createQueryState(tab);
    map.value.set(tab.id, qs);
    useQueryStateLogic(qs);
    return qs;
  };

  const clearQueryStateByTab = (id: string) => {
    map.value.delete(id);
  };

  return { getQueryStateByTab, clearQueryStateByTab };
});

const createQueryState = (tab: SQLEditorTab): WebTerminalQueryState => {
  return {
    tab,
    queryItemList: ref([createInitialQueryItemByTab(tab)]),
    timer: markRaw(useCancelableTimeout(QUERY_TIMEOUT_MS)),
    controller: createStreamingQueryController(),
  };
};

const createInitialQueryItemByTab = (
  _tab: SQLEditorTab
): WebTerminalQueryItemV1 => {
  return createQueryItemV1("");
};

export const createQueryItemV1 = (
  statement = "",
  status: WebTerminalQueryItemV1["status"] = "IDLE"
): WebTerminalQueryItemV1 => ({
  id: uniqueId(),
  statement,
  status,
});

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

  events.on("query", (params) => {
    const request = mapRequest(params);
    console.debug("query", request);

    if (status.value === "DISCONNECTED") {
      $ws.value = connect(request);
    }
  });

  const connect = (
    initialRequest: AdminExecuteRequest | undefined = undefined
  ) => {
    const request$ = input$.pipe(map((params) => mapRequest(params)));
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
        requestSubscription = request$.subscribe({
          next(request) {
            send(request);
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

const useQueryStateLogic = (qs: WebTerminalQueryState) => {
  const activeQuery = () => {
    return qs.queryItemList.value[qs.queryItemList.value.length - 1];
  };

  const pushQueryItem = () => {
    const list = qs.queryItemList.value;
    list.push(createQueryItemV1());

    if (list.length > MAX_QUERY_ITEM_COUNT) {
      list.shift();
    }
  };

  const cleanup = () => {
    activeQuery().status = "FINISHED";
    qs.timer.stop();

    pushQueryItem();
  };

  qs.controller.events.on("query", (input) => {
    qs.timer.start();
    activeQuery().params = cloneDeep(input);
    activeQuery().status = "RUNNING";
  });

  qs.controller.events.on("result", (resultSet) => {
    console.debug("event resultSet", resultSet);
    activeQuery().resultSet = resultSet;
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
    cleanup();
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
