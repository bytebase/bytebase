import { omit, throttle } from "lodash-es";
import type {
  ExecuteCommandParams,
  LanguageClientOptions,
  MessageTransports,
} from "vscode-languageclient";
import {
  BaseLanguageClient,
  CloseAction,
  ErrorAction,
  State,
} from "vscode-languageclient";
import {
  toSocket,
  WebSocketMessageReader,
  WebSocketMessageWriter,
} from "vscode-ws-jsonrpc";
import { refreshTokens } from "@/api/refreshToken";
import { sleep } from "@/utils";
import { initializeMonacoServices } from "./services";
import {
  createUrl,
  errorNotification,
  MAX_RETRIES,
  messages,
  progressiveDelay,
  WEBSOCKET_HEARTBEAT_INTERVAL,
  WEBSOCKET_HEARTBEAT_TIMEOUT,
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

type LanguageClientConnection = {
  client: MonacoLanguageClient;
  ws: WebSocket;
};

// Inline replacement for `monaco-languageclient`'s `MonacoLanguageClient`,
// which was a ~15-line subclass and the only thing we consumed from that
// package. All the LSP machinery lives in `vscode-languageclient`, whose
// undeclared `import "vscode"` resolves to the top-level `vscode` alias
// (`@codingame/monaco-vscode-extension-api`) in package.json — that alias
// must stay for language features to register against the same API
// instance that `initializeMonacoServices` initializes.
class MonacoLanguageClient extends BaseLanguageClient {
  private readonly messageTransports: MessageTransports;

  constructor({
    name,
    clientOptions,
    messageTransports,
  }: {
    name: string;
    clientOptions: LanguageClientOptions;
    messageTransports: MessageTransports;
  }) {
    super(name.toLowerCase(), name, clientOptions);
    this.messageTransports = messageTransports;
  }

  protected createMessageTransports(
    _encoding: string
  ): Promise<MessageTransports> {
    return Promise.resolve(this.messageTransports);
  }
}

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
let reconnectHeartbeatTimer: ReturnType<typeof setTimeout> | undefined;
let reconnectJob: Promise<MonacoLanguageClient> | undefined;

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

const clearHeartbeat = () => {
  clearTimeout(conn.heartbeat.timer);
  conn.heartbeat = {
    timer: undefined,
    counter: 0,
    timestamp: 0,
  };
  emit();
};

const clearReconnectHeartbeat = () => {
  clearTimeout(reconnectHeartbeatTimer);
  reconnectHeartbeatTimer = undefined;
};

const scheduleReconnectHeartbeat = () => {
  if (reconnectHeartbeatTimer || conn.state === "ready") {
    return;
  }
  reconnectHeartbeatTimer = setTimeout(() => {
    reconnectHeartbeatTimer = undefined;
    void reconnect().catch(() => undefined);
  }, WEBSOCKET_HEARTBEAT_INTERVAL);
};

const startHeartbeat = (client: MonacoLanguageClient, ws: WebSocket) => {
  let active = true;
  let heartbeatResponseTimer: ReturnType<typeof setTimeout> | undefined;

  const cleanup = () => {
    active = false;
    clearTimeout(heartbeatResponseTimer);
    clearHeartbeat();
  };

  ws.addEventListener("error", cleanup);
  ws.addEventListener("close", cleanup);

  const ping = () => {
    clearTimeout(heartbeatResponseTimer);
    const counter = conn.heartbeat.counter + 1;
    const timestamp = Date.now();
    setHeartbeat({
      counter,
      timestamp,
    });

    heartbeatResponseTimer = setTimeout(() => {
      if (active && ws.readyState === WebSocket.OPEN) {
        ws.close();
      }
    }, WEBSOCKET_HEARTBEAT_TIMEOUT);

    void Promise.resolve(
      client.sendRequest("$ping", {
        state: omit(conn.heartbeat, "timer"),
      })
    )
      .then(() => {
        if (!active) {
          return;
        }
        clearTimeout(heartbeatResponseTimer);
        heartbeatResponseTimer = undefined;
        if (ws.readyState === WebSocket.OPEN) {
          setHeartbeat({
            timer: setTimeout(ping, WEBSOCKET_HEARTBEAT_INTERVAL),
          });
        }
      })
      .catch(() => {
        if (active && ws.readyState === WebSocket.OPEN) {
          ws.close();
        }
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
        clearReconnectHeartbeat();
        setConnState({ state: "ready", retries: 0 });
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

const disposeClient = () => {
  if (state.client) {
    try {
      state.client.dispose();
    } catch {
      // ignore
    }
    state.client = undefined;
  }
  state.clientInitialized = undefined;
};

const reconnectRunner = async () => {
  setConnState({
    ws: undefined,
    state: "initial",
    retries: 0,
  });

  clearReconnectHeartbeat();
  disposeClient();
  try {
    await refreshTokens();
  } catch (err) {
    setConnState({
      ws: undefined,
      state: "closed",
      retries: 0,
    });
    scheduleReconnectHeartbeat();
    throw err;
  }

  return initializeLSPClient();
};

const reconnect = () => {
  if (reconnectJob) {
    return reconnectJob;
  }

  reconnectJob = reconnectRunner().finally(() => {
    reconnectJob = undefined;
  });
  return reconnectJob;
};

const createLanguageClient = async (): Promise<LanguageClientConnection> => {
  // `vscode-languageclient` touches the `vscode` API at construction time,
  // which requires `@codingame/monaco-vscode-api`'s `initialize()` to have
  // completed before `new MonacoLanguageClient(...)`,
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
  return {
    client: new MonacoLanguageClient({
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
    }),
    ws,
  };
};

const initializeRunner = async () => {
  const { client, ws } = await createLanguageClient();
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
  startHeartbeat(client, ws);
  return client;
};

export const initializeLSPClient = () => {
  if (state.clientInitialized) {
    return state.clientInitialized;
  }
  const job = initializeRunner().catch((err) => {
    disposeClient();
    if (conn.state !== "ready") {
      setConnState({
        ws: undefined,
        state: "closed",
        retries: 0,
      });
      scheduleReconnectHeartbeat();
    }
    throw err;
  });
  state.clientInitialized = job;
  return job;
};

export const ensureLSPConnection = () => {
  if (conn.state === "ready" || conn.state === "reconnecting") {
    return state.clientInitialized ?? initializeLSPClient();
  }
  if (conn.state === "initial" && conn.ws) {
    return state.clientInitialized ?? initializeLSPClient();
  }
  return reconnect();
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
