import { createLanguageServerWorker } from "@sql-lsp/server";
import type { ConnectionScope, Schema } from "@sql-lsp/types";
import type { MonacoLanguageClient } from "monaco-languageclient";
import type { ExecuteCommandParams } from "vscode-languageclient";
import type { SQLDialect } from "@/types";
import { createLanguageClient } from "./createLanguageClient";

type LocalStage = {
  worker: Promise<Worker>;
  client: Promise<MonacoLanguageClient>;
  stopped: boolean;

  // Store pending commands when the client is not connected yet.
  // Execute them at the first time when the connection starts.
  // Only keep the latest command for each type and drops the outdated ones.
  // So we don't need to know the perfect timing to call `executeCommand`
  // in <MonacoEditor> and wherever, just call it.
  pendingCommands: Map<string, ExecuteCommandParams>;
};

// Working as a singleton
const state: LocalStage = {
  worker: undefined as any,
  client: undefined as any,
  stopped: true,
  pendingCommands: new Map(),
};

const getWorker = (): Promise<Worker> => {
  // Won't initialize more than once.
  if (!state.worker) {
    state.worker = createLanguageServerWorker();
  }
  return state.worker;
};

const initializeLanguageClient = async () => {
  const worker = await getWorker();

  const { client } = createLanguageClient(worker);

  return client;
};

const getLanguageClient = () => {
  // Won't initialize more than once.
  if (!state.client) {
    state.client = initializeLanguageClient();
  }
  return state.client;
};

const executeCommand = (params: ExecuteCommandParams) => {
  // Don't go further if we are not connected.
  if (state.stopped) {
    state.pendingCommands.set(params.command, params);
    return;
  }
  getLanguageClient().then((client) => {
    // Double check the status since we are in an async callback
    if (state.stopped) {
      state.pendingCommands.set(params.command, params);
      return;
    }
    client.sendRequest("workspace/executeCommand", params);
  });
};

const changeSchema = (schema: Schema) => {
  executeCommand({
    command: "changeSchema",
    arguments: [schema],
  });
};

const changeDialect = (dialect: SQLDialect) => {
  executeCommand({
    command: "changeDialect",
    arguments: [dialect],
  });
};

const changeConnectionScope = (scope: ConnectionScope) => {
  executeCommand({
    command: "changeConnectionScope",
    arguments: [scope],
  });
};

const resolvePendingCommands = (client: MonacoLanguageClient) => {
  if (state.stopped) {
    return;
  }

  for (const params of state.pendingCommands.values()) {
    client.sendRequest("workspace/executeCommand", params);
  }

  state.pendingCommands.clear();
};

const start = () => {
  if (!state.stopped) {
    // Don't start twice
    return;
  }
  state.client.then((client) => {
    if (!state.stopped) {
      // Double check
      return;
    }
    try {
      client.start();
      state.stopped = false;
      resolvePendingCommands(client);
    } catch {
      // nothing todo
    }
  });
};
const stop = () => {
  state.stopped = true;

  if (!state.client) {
    // We don't need to stop if the client is not started yet
    return;
  }

  state.client.then((client) => {
    state.pendingCommands.clear();
    if (client.isRunning()) {
      client.stop();
    }
  });
};

export const useLanguageClient = () => {
  // Initialize if needed
  getLanguageClient();

  return { start, stop, changeSchema, changeDialect, changeConnectionScope };
};
