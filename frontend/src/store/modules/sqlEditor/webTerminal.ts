import Emittery from "emittery";
import { uniqueId } from "lodash-es";
import { ClientError, Status } from "nice-grpc-common";
import { defineStore } from "pinia";
import type { Subscription } from "rxjs";
import { fromEventPattern, map, Observable } from "rxjs";
import { markRaw, ref, shallowRef } from "vue";
import { useCancelableTimeout } from "@/composables/useCancelableTimeout";
import type {
  SQLResultSetV1,
  StreamingQueryController,
  SQLEditorTab,
  WebTerminalQueryItemV1,
  WebTerminalQueryState,
  SQLEditorQueryParams,
} from "@/types";
import { Duration } from "@/types/proto/google/protobuf/duration";
import {
  AdminExecuteRequest,
  AdminExecuteResponse,
  QueryResult,
} from "@/types/proto/v1/sql_service";
import {
  extractGrpcErrorMessage,
  getErrorCode as extractGrpcStatusCode,
} from "@/utils/grpcweb";
import { useDatabaseV1Store } from "../v1";

const ENDPOINT = "/v1:adminExecute";
const SIG_ABORT = 3000 + Status.ABORTED;
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
    controller: createStreamingQueryController(tab),
  };
};

const createInitialQueryItemByTab = (
  _tab: SQLEditorTab
): WebTerminalQueryItemV1 => {
  return createQueryItemV1("");
};

export const createQueryItemV1 = (
  sql = "",
  status: WebTerminalQueryItemV1["status"] = "IDLE"
): WebTerminalQueryItemV1 => ({
  id: uniqueId(),
  sql,
  status,
});

const createStreamingQueryController = (tab: SQLEditorTab) => {
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
    const request = mapRequest(tab, params);
    console.debug("query", request);

    if (status.value === "DISCONNECTED") {
      $ws.value = connect(request);
    }
  });

  const connect = (
    initialRequest: AdminExecuteRequest | undefined = undefined
  ) => {
    const request$ = input$.pipe(map((params) => mapRequest(tab, params)));
    const abortController = new AbortController();

    const url = new URL(`${window.location.origin}${ENDPOINT}`);
    url.protocol = url.protocol.replace(/^http/, "ws");
    const ws = new WebSocket(url);
    status.value = "CONNECTED";

    const send = (request: AdminExecuteRequest) => {
      const payload: any = {
        ...request,
      };
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
            const { results } = data.result;
            if (Array.isArray(results)) {
              results.forEach((result) => {
                result.latency = parseDuration(result.latency);
              });
            }
            const response = AdminExecuteResponse.fromJSON(data.result);
            subscriber.next(response);
          } else if (data.error) {
            const err = new ClientError(
              ENDPOINT,
              data.error.code,
              data.error.message
            );
            subscriber.error(err);
          }
        } catch (err) {
          subscriber.error(err);
        }
      });
      ws.addEventListener("error", (event) => {
        console.debug("error", event);
        subscriber.error(
          new ClientError(ENDPOINT, Status.INTERNAL, "Internal server error")
        );
      });
      ws.addEventListener("close", (event) => {
        console.debug("ws close", event.wasClean, event.reason, event.code);
        if (event.code === SIG_ABORT) {
          subscriber.error(
            new ClientError(ENDPOINT, Status.ABORTED, event.reason)
          );
          return;
        }
        subscriber.error(
          new ClientError(
            ENDPOINT,
            Status.DEADLINE_EXCEEDED,
            `Closed by server ${event.code}`
          )
        );
      });

      return () => {
        console.debug("teardown");
        requestSubscription.unsubscribe();
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
          advices: [],
          allowExport: false,
          ...response,
        });
      },
      error(error) {
        console.debug("subscribe error", error);

        const result: SQLResultSetV1 = {
          error: extractGrpcErrorMessage(error),
          status: extractGrpcStatusCode(error),
          advices: [],
          results: [],
          allowExport: false,
        };
        if (result.status === Status.ABORTED && !result.error) {
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
    activeQuery().params = input;
    activeQuery().status = "RUNNING";
  });

  qs.controller.events.on("result", (resultSet) => {
    console.debug("event resultSet", resultSet);
    activeQuery().resultSet = resultSet;
    cleanup();
  });
};

export const mockAffectedV1Rows0 = (): QueryResult => {
  return QueryResult.fromPartial({
    columnNames: ["Affected Rows"],
    columnTypeNames: ["BIGINT"],
    masked: [false],
    error: "",
    statement: "",
    rows: [
      {
        values: [
          {
            int64Value: 0,
          },
        ],
      },
    ],
  });
};

const mapRequest = (
  tab: SQLEditorTab,
  params: SQLEditorQueryParams
): AdminExecuteRequest => {
  const { connection, statement, explain } = params;

  const database = useDatabaseV1Store().getDatabaseByName(connection.database);
  const request = AdminExecuteRequest.fromPartial({
    name: database.name,
    statement: explain ? `EXPLAIN ${statement}` : statement,
    schema: connection.schema,
  });
  return request;
};

export const parseDuration = (str: string): Duration | undefined => {
  if (typeof str !== "string") return undefined;

  const matches = str.match(/^([0-9.]+)s$/);
  if (!matches) return undefined;
  const totalSeconds = parseFloat(matches[0]);
  if (Number.isNaN(totalSeconds) || totalSeconds < 0) return undefined;
  const seconds = Math.floor(totalSeconds);
  const nanos = (totalSeconds - seconds) * 1e9;
  return Duration.fromPartial({
    seconds,
    nanos,
  });
};
