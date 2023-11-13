import { MonacoLanguageClient } from "monaco-languageclient";
import {
  CloseAction,
  ErrorAction,
  MessageTransports,
} from "vscode-languageclient";
import {
  WebSocketMessageReader,
  WebSocketMessageWriter,
  toSocket,
} from "vscode-ws-jsonrpc";

export const createLanguageClient = (
  transports: MessageTransports
): MonacoLanguageClient => {
  return new MonacoLanguageClient({
    name: "Bytebase Language Client",
    clientOptions: {
      // use a language id as a document selector
      documentSelector: ["sql"],
      // disable the default error handler
      errorHandler: {
        error: () => ({ action: ErrorAction.Continue }),
        closed: () => ({ action: CloseAction.DoNotRestart }),
      },
    },
    // create a language client connection from the JSON RPC connection on demand
    connectionProvider: {
      get: () => {
        return Promise.resolve(transports);
      },
    },
  });
};

export const createUrl = (
  hostname: string,
  port: number,
  path: string,
  searchParams: Record<string, any> = {},
  secure: boolean = location.protocol === "https:"
): string => {
  const protocol = secure ? "wss" : "ws";
  const url = new URL(`${protocol}://${hostname}:${port}${path}`);

  for (const [key, value] of Object.entries(searchParams)) {
    const v = value instanceof Array ? value.join(",") : value;
    if (v) {
      url.searchParams.set(key, v);
    }
  }

  return url.toString();
};

export const createWebSocketAndStartClient = (
  url: string
): {
  webSocket: WebSocket;
  languageClient: Promise<MonacoLanguageClient>;
} => {
  const webSocket = new WebSocket(url);
  const languageClient = new Promise<MonacoLanguageClient>((resolve) => {
    webSocket.onopen = () => {
      const socket = toSocket(webSocket);
      const reader = new WebSocketMessageReader(socket);
      const writer = new WebSocketMessageWriter(socket);
      const languageClient = createLanguageClient({
        reader,
        writer,
      });
      languageClient.start();
      reader.onClose(() => languageClient.stop());
      resolve(languageClient);
    };
  });

  return {
    webSocket,
    languageClient,
  };
};

const state = {
  clientInitialized: undefined as Promise<MonacoLanguageClient> | undefined,
};

const initializeRunner = async () => {
  const url = createUrl("localhost", 23333, "/helloServer");
  const client = await createWebSocketAndStartClient(url).languageClient;
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
