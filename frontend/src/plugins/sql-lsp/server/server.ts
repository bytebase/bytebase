import {
  InitializeResult,
  CompletionParams,
  CompletionTriggerKind,
  CompletionItem,
} from "vscode-languageserver/browser";
import { initializeConnection } from "./initializeConnection";
import { EngineTypesUsingSQL, Schema, SQLDialect } from "@sql-lsp/types";
import { complete } from "./complete";

declare const self: DedicatedWorkerGlobalScope;

const TRIGGER_CHARACTERS = [".", " "];

const { connection, documents } = initializeConnection(self);

type LocalState = {
  schema: Schema;
  dialect: SQLDialect;
};

const state: LocalState = {
  schema: { databases: [] } as Schema,
  dialect: "MYSQL",
};

connection.onInitialize((params): InitializeResult => {
  console.debug(`onInitialize`, params);

  return {
    capabilities: {
      textDocumentSync: 1,
      completionProvider: {
        resolveProvider: true,
        triggerCharacters: TRIGGER_CHARACTERS,
      },
      renameProvider: true,
      executeCommandProvider: {
        commands: ["changeSchema", "changeDialect"],
      },
    },
  };
});

connection.onCompletion(
  async (params: CompletionParams): Promise<CompletionItem[]> => {
    console.debug("onCompletion", params);
    // Make sure the client does not send use completion request for characters
    // other than the dot which we asked for.
    if (
      params.context?.triggerKind === CompletionTriggerKind.TriggerCharacter
    ) {
      const triggerCharacter = params.context?.triggerCharacter;
      if (!triggerCharacter || !TRIGGER_CHARACTERS.includes(triggerCharacter)) {
        return [];
      }
    }
    const document = documents.get(params.textDocument.uri);
    if (!document) {
      return [];
    }
    const text = documents.get(params.textDocument.uri)?.getText();
    if (!text) {
      return [];
    }
    const candidates = await complete(
      params,
      document,
      state.schema,
      state.dialect
    );
    console.debug("onCompletion returns: " + JSON.stringify(candidates));
    return candidates;
  }
);

connection.onCompletionResolve((item: CompletionItem): CompletionItem => {
  console.debug("onCompletionResolve", item);
  return item;
});

connection.onExecuteCommand((request) => {
  console.debug(
    `received executeCommand request: ${request.command}`,
    JSON.stringify(request.arguments)
  );
  const args = request.arguments ?? [];
  if (request.command === "changeSchema") {
    const schema = args[0];
    if (!schema) {
      connection.sendNotification("error", {
        message: "schema required",
      });
      return;
    }
    state.schema = schema;
  } else if (request.command === "changeDialect") {
    const dialect = args[0];
    if (EngineTypesUsingSQL.includes(dialect)) {
      state.dialect = dialect;
    } else {
      state.dialect = "MYSQL";
    }
  } else {
    connection.sendNotification("error", {
      message: "unknown command requested",
      request,
    });
  }
});

connection.listen();
console.debug("language server start listening");
