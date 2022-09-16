import type { ExecuteCommandParams } from "monaco-languageclient";
import { MonacoLanguageClient, MonacoServices } from "monaco-languageclient";
import { StandaloneServices } from "vscode/services";
import getMessageServiceOverride from "vscode/service-override/messages";
import { createLanguageServerWorker } from "@sql-lsp/server";
import { Schema, SQLDialect } from "@sql-lsp/types";
import { createLanguageClient } from "./createLanguageClient";

type LocalStage = {
  worker: Promise<Worker>;
  client: Promise<MonacoLanguageClient>;
  stopped: boolean;
};

// Working as a singleton
const state: LocalStage = {
  worker: undefined as any,
  client: undefined as any,
  stopped: true,
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

  StandaloneServices.initialize({
    ...getMessageServiceOverride(document.body),
  });
  // install Monaco language client services
  MonacoServices.install();

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
    return;
  }
  getLanguageClient().then((client) => {
    // Double check the status since we are in an async callback
    if (state.stopped) {
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
    client.stop();
  });
};

export const useLanguageClient = () => {
  // Initialize if needed
  getLanguageClient();

  return { start, stop, changeSchema, changeDialect };
};
