import { MonacoLanguageClient } from "monaco-languageclient";
import { CloseAction, ErrorAction } from "vscode-languageclient";
import {
  BrowserMessageReader,
  BrowserMessageWriter,
} from "vscode-languageserver-protocol/browser";

export const createLanguageClient = (worker: Worker) => {
  const reader = new BrowserMessageReader(worker);
  const writer = new BrowserMessageWriter(worker);

  const client = new MonacoLanguageClient({
    name: "SQL Language Client",
    clientOptions: {
      // use a language id as a document selector
      documentSelector: [{ language: "sql" }],
      // disable the default error handler
      errorHandler: {
        error: () => ({ action: ErrorAction.Continue }),
        closed: () => ({ action: CloseAction.DoNotRestart }),
      },
    },
    // create a language client connection to the server running in the web worker
    connectionProvider: {
      get: () => {
        return Promise.resolve({ reader, writer });
      },
    },
  });

  return { reader, writer, client };
};
