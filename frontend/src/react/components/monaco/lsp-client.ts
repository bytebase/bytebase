import { omit, throttle } from "lodash-es";
import { MonacoLanguageClient } from "monaco-languageclient";
import type { ExecuteCommandParams } from "vscode-languageclient";
import { CloseAction, ErrorAction, State } from "vscode-languageclient";
import {
  toSocket,
  WebSocketMessageReader,
  WebSocketMessageWriter,
} from "vscode-ws-jsonrpc";
import { refreshTokens } from "@/connect/refreshToken";
import { sleep } from "@/utils";
import { initializeMonacoServices } from "./services";
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
  heartbeat: {
    counter: number;
    timer: ReturnType<typeof setTimeout> | undefined;
    timestamp: number;
  };
  lastCommand: ExecuteCommandParams | undefined;
  retries: number;
  state: "initial" | "ready" | "closed" | "reconnecting";
  url: string;
  ws: Promise<WebSocket> | undefined;
};

const listeners = new Set<() => void>();

const emit = () => {
  listeners.forEach((listener) => listener());
};

const lspHost = (() => {
  const grpcLocal = import.meta.env.BB_GRPC_LOCAL;
  if (grpcLocal) {
    try {
      return new URL(grpcLocal).host;
    } catch {
      // ignore
    }
  }
  return location.host;
})();

const conn: ConnectionState = {
  url: createUrl(lspHost, "/lsp").toString(),
  state: "initial",
  ws: undefined,
  lastCommand: undefined,
  retries: 0,
  heartbeat: {
    timer: undefined,
    counter: 0,
    timestamp: 0,
  },
};
let connectionSnapshot = {
  state: conn.state,
  heartbeat: conn.heartbeat,
};

const state = {
  client: undefined as MonacoLanguageClient | undefined,
  clientInitialized: undefined as Promise<MonacoLanguageClient> | undefined,
};

const setConnState = (patch: Partial<ConnectionState>) => {
  Object.assign(conn, patch);
  connectionSnapshot = {
    state: conn.state,
    heartbeat: conn.heartbeat,
  };
  emit();
};

const setHeartbeat = (patch: Partial<ConnectionState["heartbeat"]>) => {
  Object.assign(conn.heartbeat, patch);
  connectionSnapshot = {
    state: conn.state,
    heartbeat: conn.heartbeat,
  };
  emit();
};

const useHeartbeat = (ws: WebSocket) => {
  const cleanup = () => {
    clearTimeout(conn.heartbeat.timer);
    conn.heartbeat = {
      timer: undefined,
      counter: 0,
      timestamp: 0,
    };
    emit();
  };

  ws.addEventListener("error", cleanup);
  ws.addEventListener("close", cleanup);

  const ping = () => {
    setHeartbeat({
      counter: conn.heartbeat.counter + 1,
      timestamp: Date.now(),
    });
    ws.send(
      JSON.stringify({
        jsonrpc: "2.0",
        method: "$ping",
        params: {
          state: omit(conn.heartbeat, "timer"),
        },
      })
    );
    setHeartbeat({
      timer: setTimeout(ping, WEBSOCKET_HEARTBEAT_INTERVAL),
    });
  };

  ping();
};

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
    emit();

    switch (conn.state) {
      case "closed":
        reject("Connection is closed");
        return;
      case "ready":
      case "reconnecting":
        setConnState({ state: "reconnecting" });
        break;
      case "initial":
        break;
    }

    sleep(progressiveDelay(retries)).then(() => {
      const handleError = (code: number, reason: string) => {
        if (conn.state === "closed" || conn.state === "ready") {
          return;
        }
        if (conn.retries >= MAX_RETRIES) {
          setConnState({ state: "closed" });
          reject(
            `${messages.disconnected()}: max retries exceeded (${MAX_RETRIES}). code=${code} reason="${reason}"`
          );
          return;
        }
        connect(resolve, reject);
      };

      const timer = setTimeout(() => {
        handleError(-1, "timeout");
      }, WEBSOCKET_TIMEOUT);

      ws.addEventListener("open", () => {
        clearTimeout(timer);
        if (conn.state === "ready" || conn.state === "closed") {
          return;
        }
        setConnState({ state: "ready", retries: 0 });
        useHeartbeat(ws);
        resolve(ws);
      });

      ws.addEventListener("close", (event) => {
        clearTimeout(timer);
        handleError(event.code, event.reason);
      });
    });
  };

  const promise = new Promise<WebSocket>(connect);
  setConnState({ ws: promise });
  return promise;
};

const reconnect = async () => {
  setConnState({
    ws: undefined,
    state: "initial",
    retries: 0,
  });

  await refreshTokens();

  if (state.client) {
    try {
      state.client.dispose();
    } catch {
      // ignore
    }
    state.client = undefined;
  }
  state.clientInitialized = undefined;
  await initializeLSPClient();
};

const createLanguageClient = async (): Promise<MonacoLanguageClient> => {
  // `monaco-languageclient` v9+ requires `@codingame/monaco-vscode-api`'s
  // `initialize()` to have completed before `new MonacoLanguageClient(...)`,
  // otherwise the constructor throws "Default api is not ready yet, do not
  // forget to import 'vscode/localExtensionHost' and wait for services
  // initialization". `MonacoEditor.tsx` kicks off `initializeLSPClient()`
  // *before* awaiting `createMonacoEditor()` so the LSP WebSocket connects
  // in parallel with Monaco loading — that means we can't rely on the
  // caller to have initialized services first. Await it here so callers
  // can fire-and-forget safely.
  await initializeMonacoServices();
  const ws = await connectWebSocket();
  const socket = toSocket(ws);
  const reader = new WebSocketMessageReader(socket);
  const writer = new WebSocketMessageWriter(socket);
  return new MonacoLanguageClient({
    name: "Bytebase Language Client",
    clientOptions: {
      documentSelector: ["sql", "javascript"],
      initializationOptions: {
        performanceMode: true,
        diagnosticDelay: 500,
        disableFeaturesWhileTyping: true,
      },
      middleware: {
        provideHover: throttle(async (document, position, token, next) => {
          return next(document, position, token);
        }, 300),
        provideCompletionItem: throttle(
          async (document, position, context, token, next) => {
            const triggerCharacters = [".", ",", "(", " "];
            if (
              context.triggerKind === 1 &&
              !triggerCharacters.includes(context.triggerCharacter || "")
            ) {
              return { items: [] };
            }
            return next(document, position, context, token);
          },
          200
        ),
      },
      errorHandler: {
        error: () => ({ action: ErrorAction.Continue }),
        closed: () => {
          reconnect().catch((err) => errorNotification(err));
          return { action: CloseAction.DoNotRestart };
        },
      },
    },
    messageTransports: { reader, writer },
  });
};

const initializeRunner = async () => {
  const client = await createLanguageClient();
  client.onDidChangeState((event) => {
    if (event.newState === State.Running) {
      const { lastCommand } = conn;
      if (lastCommand) {
        void executeCommand(client, lastCommand.command, lastCommand.arguments);
      }
    }
  });

  try {
    await client.start();
  } catch (err) {
    errorNotification(err);
  }

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
  setConnState({ lastCommand: executeCommandParams });
  return client.sendRequest("workspace/executeCommand", executeCommandParams);
};

export const subscribeConnectionState = (listener: () => void) => {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
};

export const getConnectionStateSnapshot = () => connectionSnapshot;

export const getConnectionWebSocket = () => conn.ws;
