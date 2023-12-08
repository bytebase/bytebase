import { MonacoLanguageClient } from "monaco-languageclient";
import {
  CloseAction,
  ErrorAction,
  ExecuteCommandParams,
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
        closed: () => ({ action: CloseAction.Restart }),
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
  host: string,
  path: string,
  searchParams: Record<string, any> = {},
  secure: boolean = location.protocol === "https:"
): string => {
  const protocol = secure ? "wss" : "ws";
  const url = new URL(`${protocol}://${host}${path}`);

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
  client: undefined as MonacoLanguageClient | undefined,
  clientInitialized: undefined as Promise<MonacoLanguageClient> | undefined,
};

const initializeRunner = async () => {
  const url = createUrl(location.host, "/lsp");
  const client = await createWebSocketAndStartClient(url).languageClient;
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
  const result = await client.sendRequest(
    "workspace/executeCommand",
    executeCommandParams
  );
  return result;
};
