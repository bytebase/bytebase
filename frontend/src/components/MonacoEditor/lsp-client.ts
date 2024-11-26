import { omit } from "lodash-es";
import {
  MonacoLanguageClient,
  type IConnectionProvider,
} from "monaco-languageclient";
import type { ExecuteCommandParams } from "vscode-languageclient";
import { CloseAction, ErrorAction, State } from "vscode-languageclient";
import {
  toSocket,
  WebSocketMessageReader,
  WebSocketMessageWriter,
} from "vscode-ws-jsonrpc";
import { shallowReactive, toRef } from "vue";
import { sleep } from "@/utils";
import {
  createUrl,
  errorNotification,
  MAX_RETRIES,
  messages,
  progressiveDelay,
  WEBSOCKET_HEARTBEAT_INTERVAL,
  WEBSOCKET_TIMEOUT,
} from "./utils";

export type ConnectionState = {
  url: string;
  state: "initial" | "ready" | "closed" | "reconnecting";
  ws: Promise<WebSocket> | undefined;
  lastCommand: ExecuteCommandParams | undefined;
  retries: number;
  heartbeat: {
    timer: NodeJS.Timeout | undefined;
    counter: number;
    timestamp: number;
  };
};

const conn = shallowReactive<ConnectionState>({
  url: createUrl(location.host, "/lsp").toString(),
  state: "initial",
  ws: undefined,
  lastCommand: undefined,
  retries: 0,
  heartbeat: shallowReactive({
    timer: undefined,
    counter: 0,
    timestamp: 0,
  }),
});

const connectWebSocket = () => {
  if (conn.ws) {
    return conn.ws;
  }

  const connect = (
    resolve: (value: WebSocket | PromiseLike<WebSocket>) => void,
    reject: (reason?: any) => void
  ) => {
    const ws = new WebSocket(conn.url);
    const retries = conn.retries++;

    switch (conn.state) {
      case "closed":
        return reject(`Connection is closed`);
      case "initial":
        break;
      case "ready":
      case "reconnecting":
        conn.state = "reconnecting";
        break;
    }

    const delay = progressiveDelay(retries);
    console.debug(
      `[LSP-Client] try connecting: state=${conn.state} retries=${retries} delay=${delay}`
    );

    sleep(delay).then(() => {
      const handleError = (code: number, reason: string) => {
        if (conn.state === "closed" || conn.state === "ready") {
          return;
        }

        if (conn.retries >= MAX_RETRIES) {
          conn.state = "closed";
          return reject(
            `${messages.disconnected()}: max retries exceeded (${MAX_RETRIES}). code=${code} reason="${reason}"`
          );
        }
        return connect(resolve, reject);
      };

      const timer = setTimeout(() => {
        handleError(-1, "timeout");
      }, WEBSOCKET_TIMEOUT);

      ws.addEventListener("open", () => {
        clearTimeout(timer);
        console.debug(`[LSP-Client] WebSocket open`);
        if (conn.state === "ready" || conn.state === "closed") {
          return;
        }
        conn.state = "ready";
        conn.retries = 0; // reset retry counter
        useHeartbeat(ws);
        resolve(ws);
      });
      ws.addEventListener("close", (e) => {
        clearTimeout(timer);
        console.debug(
          `[LSP-Client] WebSocket close state=${conn.state} code=${e.code} reason=${e.reason}`
        );
        handleError(e.code, e.reason);
      });
    });
  };

  const promise = new Promise<WebSocket>(connect);
  conn.ws = promise;
  return promise;
};

const state = {
  client: undefined as MonacoLanguageClient | undefined,
  clientInitialized: undefined as Promise<MonacoLanguageClient> | undefined,
};

export const createLanguageClient = (
  connectionProvider: IConnectionProvider
): MonacoLanguageClient => {
  return new MonacoLanguageClient({
    name: "Bytebase Language Client",
    clientOptions: {
      // use a language id as a document selector
      documentSelector: ["sql"],
      // disable the default error handler
      errorHandler: {
        error: (error, message, count) => {
          console.debug("[MonacoLanguageClient] error", error, message, count);
          return {
            action: ErrorAction.Continue,
          };
        },
        closed: async () => {
          console.debug("[MonacoLanguageClient] closed");
          conn.ws = undefined;
          try {
            await connectWebSocket();
            return {
              action: CloseAction.Restart,
            };
          } catch (err) {
            errorNotification(err);
            return {
              action: CloseAction.DoNotRestart,
            };
          }
        },
      },
    },
    // create a language client connection from the JSON RPC connection on demand
    connectionProvider,
  });
};

export const createWebSocketAndStartClient = (): {
  languageClient: Promise<MonacoLanguageClient>;
} => {
  const languageClient = new Promise<MonacoLanguageClient>((resolve) => {
    const languageClient = createLanguageClient({
      async get() {
        const ws = await connectWebSocket();
        const socket = toSocket(ws);
        const reader = new WebSocketMessageReader(socket);
        const writer = new WebSocketMessageWriter(socket);
        return { reader, writer };
      },
    });
    languageClient.onDidChangeState((e) => {
      if (e.newState === State.Running) {
        const { lastCommand } = conn;
        if (lastCommand) {
          // When LSP Client is reconnected, the LSP context (e.g. setMetadata)
          // will be cleared.
          // So we need to catch the last command, and re-send it to recover
          // the context
          executeCommand(
            languageClient,
            lastCommand.command,
            lastCommand.arguments
          );
        }
      }
    });

    languageClient.start().catch((err) => {
      // LSP Client startup failed.
      errorNotification(err);
    });

    resolve(languageClient);
  });

  return {
    languageClient,
  };
};

const initializeRunner = async () => {
  const client = await createWebSocketAndStartClient().languageClient;
  state.client = client;
  return client;
};

export const initializeLSPClient = () => {
  if (state.clientInitialized) {
    return state.clientInitialized;
  }

  const job = initializeRunner();
  state.clientInitialized = job;
  return job;
};

export const useLSPClient = () => {
  const { client } = state;
  if (!client) {
    throw new Error(
      "Unexpected `useLSPClient` call before lsp client initialized"
    );
  }
  return client;
};

export const executeCommand = async (
  client: MonacoLanguageClient,
  command: string,
  args: any[] | undefined
) => {
  const executeCommandParams: ExecuteCommandParams = {
    command,
    arguments: args,
  };
  conn.lastCommand = executeCommandParams;
  const result = await client.sendRequest(
    "workspace/executeCommand",
    executeCommandParams
  );
  return result;
};

const useHeartbeat = (ws: WebSocket) => {
  const cleanup = () => {
    clearTimeout(conn.heartbeat.timer);
    conn.heartbeat = {
      timer: undefined,
      counter: 0,
      timestamp: 0,
    };
  };

  ws.addEventListener("error", cleanup);
  ws.addEventListener("close", cleanup);

  const ping = () => {
    conn.heartbeat.counter++;
    conn.heartbeat.timestamp = Date.now();
    ws.send(
      JSON.stringify({
        jsonrpc: "2.0",
        method: "$ping",
        params: {
          state: omit(conn.heartbeat, "timer"),
        },
      })
    );
    conn.heartbeat.timer = setTimeout(ping, WEBSOCKET_HEARTBEAT_INTERVAL);
  };

  ping();
};

export const connectionState = toRef(conn, "state");
export const connectionHeartbeat = toRef(conn, "heartbeat");
