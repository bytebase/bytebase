import { omit, throttle } from "lodash-es";
import { MonacoLanguageClient } from "monaco-languageclient";
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
    reject: (reason?: unknown) => void
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

const createLanguageClient = async (): Promise<MonacoLanguageClient> => {
  const ws = await connectWebSocket();
  const socket = toSocket(ws);
  const reader = new WebSocketMessageReader(socket);
  const writer = new WebSocketMessageWriter(socket);
  // NOTE: We cannot debounce textDocument/didChange as it breaks LSP incremental sync
  const client = new MonacoLanguageClient({
    name: "Bytebase Language Client",
    clientOptions: {
      // use a language id as a document selector
      documentSelector: ["sql"],
      // Optimize initialization options
      initializationOptions: {
        // Request server to batch/throttle expensive operations
        performanceMode: true,
        // Reduce diagnostic frequency
        diagnosticDelay: 500,
        // Disable expensive features during typing
        disableFeaturesWhileTyping: true,
      },
      // Configure which capabilities to enable
      middleware: {
        // Throttle hover requests
        provideHover: throttle(async (document, position, token, next) => {
          return next(document, position, token);
        }, 300),
        // Throttle completion requests
        provideCompletionItem: throttle(
          async (document, position, context, token, next) => {
            // Only trigger completion on specific characters
            const triggerCharacters = [".", ",", "(", " "];
            if (
              context.triggerKind === 1 &&
              !triggerCharacters.includes(context.triggerCharacter || "")
            ) {
              return { items: [] }; // Return empty for non-trigger characters
            }
            return next(document, position, context, token);
          },
          200
        ),
      },
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
    messageTransports: {
      reader,
      writer,
    },
  });

  return client;
};

const createWebSocketAndStartClient = (): {
  languageClient: Promise<MonacoLanguageClient>;
} => {
  const languageClient = (async () => {
    const languageClient = await createLanguageClient();
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

    try {
      await languageClient.start();
    } catch (err) {
      // LSP Client startup failed.
      errorNotification(err);
    }

    return languageClient;
  })();

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

export const executeCommand = async (
  client: MonacoLanguageClient,
  command: string,
  args: unknown[] | undefined
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
export const connectionWebSocket = toRef(conn, "ws");
