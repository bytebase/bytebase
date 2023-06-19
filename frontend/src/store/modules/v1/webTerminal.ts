import { computed, markRaw, reactive, ref } from "vue";
import { defineStore } from "pinia";
import { uniqueId } from "lodash-es";
import { Metadata } from "nice-grpc-web";

import {
  StreamingQueryController,
  TabInfo,
  UNKNOWN_ID,
  WebTerminalQueryItemV1,
  WebTerminalQueryParamsV1,
  WebTerminalQueryState,
} from "@/types";
import Emittery from "emittery";
import { useCancelableTimeout } from "@/composables/useCancelableTimeout";
import { from, fromEventPattern, map, Observable } from "rxjs";
import { defer } from "@/utils";
import {
  AdminExecuteRequest,
  AdminExecuteResponse,
  QueryResult,
} from "@/types/proto/v1/sql_service";
import { useDatabaseV1Store } from "./database";
import { useInstanceV1Store } from "./instance";
import { sqlStreamingServiceClient } from "@/grpcweb";
import { extractGrpcErrorMessage } from "@/utils/grpcweb";
import { pushNotification } from "../notification";

const QUERY_TIMEOUT_MS = 5000;
const MAX_QUERY_ITEM_COUNT = 20;

export const useWebTerminalV1Store = defineStore("webTerminal_v1", () => {
  const map = ref(new Map<string, WebTerminalQueryState>());

  const getQueryStateByTab = (tab: TabInfo) => {
    const existed = map.value.get(tab.id);
    if (existed) return existed;

    const qs = createQueryState(tab);
    useQueryStateLogic(qs);
    map.value.set(tab.id, qs);
    return qs;
  };

  const clearQueryStateByTab = (tab: TabInfo) => {
    map.value.delete(tab.id);
  };

  return { getQueryStateByTab, clearQueryStateByTab };
});

const createQueryState = (tab: TabInfo): WebTerminalQueryState => {
  return reactive({
    tab,
    queryItemList: reactive([createInitialQueryItemByTab(tab)]),
    timer: markRaw(useCancelableTimeout(QUERY_TIMEOUT_MS)),
    controller: createStreamingQueryController(tab),
  });
};

const createInitialQueryItemByTab = (tab: TabInfo): WebTerminalQueryItemV1 => {
  return createQueryItemV1(tab.statement);
};

export const createQueryItemV1 = (
  sql = "",
  status: WebTerminalQueryItemV1["status"] = "IDLE"
): WebTerminalQueryItemV1 => ({
  id: uniqueId(),
  sql,
  status,
});

const createStreamingQueryController = (tab: TabInfo) => {
  const events: StreamingQueryController["events"] = markRaw(new Emittery());
  const input$ = fromEventPattern<WebTerminalQueryParamsV1>(
    (handler) => events.on("query", handler),
    (handler) => events.off("query", handler)
  );

  const controller: StreamingQueryController = {
    events,
    input$,
    response$: fromEventPattern<AdminExecuteResponse>(
      (handler) => events.on("response", handler),
      (handler) => events.off("response", handler)
    ),
    abort() {
      // noop here. will be overwritten after connected
    },
  };

  const connect = (retries = 0) => {
    const abortController = new AbortController();

    const requestParams$ = input$.pipe(
      map((params): AdminExecuteRequest => {
        const { instanceId, databaseId } = tab.connection;
        const instance = useInstanceV1Store().getInstanceByUID(instanceId);
        const database = useDatabaseV1Store().getDatabaseByUID(databaseId);
        return AdminExecuteRequest.fromJSON({
          name: instance.name,
          connectionDatabase:
            database.uid === String(UNKNOWN_ID) ? "" : database.databaseName,
          statement: params.query,
        });
      })
    );

    let response$: Observable<AdminExecuteResponse>;
    try {
      const requestParamsStream = toAsyncIterable(requestParams$);

      const responseStream = sqlStreamingServiceClient.adminExecute(
        requestParamsStream,
        {
          // metadata: new Metadata().set(
          //   "cookie",
          //   `access-token=${accessTokenCopiedFromCookie}; refresh-token=${refreshTokenCopiedFromCookie}`
          // ),
          signal: abortController.signal,
        }
      );
      controller.abort = abortController.abort.bind(abortController);
      response$ = from(responseStream);
    } catch (err) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: "Connection failed",
        description: extractGrpcErrorMessage(err),
      });
      response$ = new Observable<AdminExecuteResponse>();
    }

    response$.subscribe({
      next(response) {
        events.emit("response", response);
      },
      error(error) {
        events.emit(
          "response",
          AdminExecuteResponse.fromJSON({
            results: [
              {
                error: extractGrpcErrorMessage(error),
              },
            ],
          })
        );
        // if (retries < 5) {
        //   connect(retries + 1);
        // }
      },
    });
  };

  connect();
  return controller;
};

const useQueryStateLogic = (qs: WebTerminalQueryState) => {
  const activeQuery = () => {
    return qs.queryItemList[qs.queryItemList.length - 1];
  };

  const pushQueryItem = () => {
    const list = qs.queryItemList;
    list.push(createQueryItemV1());

    if (list.length > MAX_QUERY_ITEM_COUNT) {
      list.shift();
    }
  };

  const cleanup = () => {
    activeQuery().status = "FINISHED";
    qs.timer.stop();

    pushQueryItem();
    // Clear the tab's statement and keep it sync with the latest query
    qs.tab.statement = "";
    qs.tab.selectedStatement = "";
  };

  qs.controller.input$.subscribe({
    next: (input) => {
      // Send request to stream connection
      qs.timer.start();
      activeQuery().params = input;
      activeQuery().status = "RUNNING";
    },
  });

  qs.controller.response$.subscribe({
    next: (response) => {
      const { results } = response;
      results.forEach((result) => {
        if (!result.error && result.columnNames.length === 0) {
          Object.assign(result, mockAffectedV1Rows0());
        }
      });
      activeQuery().resultSet = {
        error: "",
        advices: [],
        ...response,
      };
      cleanup();
    },
  });
};

async function* toAsyncIterable<T>(
  observable: Observable<T>
): AsyncIterable<T> {
  const state = {
    curr: defer<T>(),
    finished: false,
  };

  const subscription = observable.subscribe({
    next(value) {
      const d = state.curr;
      state.curr = defer<T>();
      d.resolve(value);
    },
    error(error: unknown) {
      const d = state.curr;
      state.curr = defer<T>();
      d.reject(error instanceof Error ? error : new Error(String(error)));
    },
    complete() {
      state.finished = true;
      state.curr.resolve(undefined as any);
    },
  });

  try {
    while (true) {
      const value = await state.curr.promise;
      if (state.finished) break;
      yield value;
    }
  } finally {
    subscription.unsubscribe();
  }
}

const accessTokenCopiedFromCookie =
  "eyJhbGciOiJIUzI1NiIsImtpZCI6InYxIiwidHlwIjoiSldUIn0.eyJuYW1lIjoiSmltIExpdSIsImlzcyI6ImJ5dGViYXNlIiwic3ViIjoiMTAxIiwiYXVkIjpbImJiLnVzZXIuYWNjZXNzLmRldiJdLCJleHAiOjE2ODcyMzcyOTQsImlhdCI6MTY4NzE1MDg5NH0.Zs5uinQhwY82W6kjMozTHyQykV52sw3baCAvX-kqtnI";
const refreshTokenCopiedFromCookie =
  "eyJhbGciOiJIUzI1NiIsImtpZCI6InYxIiwidHlwIjoiSldUIn0.eyJuYW1lIjoiSmltIExpdSIsImlzcyI6ImJ5dGViYXNlIiwic3ViIjoiMTAxIiwiYXVkIjpbImJiLnVzZXIucmVmcmVzaC5kZXYiXSwiZXhwIjoxNjg3NzU1Njk0LCJpYXQiOjE2ODcxNTA4OTR9.2pRgUrBpM0AoPzXeD6Da4SU6hOBzpO5U2vccRTyJ0qI";

export const mockAffectedV1Rows0 = (): QueryResult => {
  return {
    columnNames: ["Affected Rows"],
    columnTypeNames: ["BIGINT"],
    masked: [false],
    error: "",
    rows: [
      {
        values: [
          {
            int64Value: 0,
          },
        ],
      },
    ],
  };
};
