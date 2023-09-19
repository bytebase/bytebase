import { TextDocument } from "vscode-languageserver-textdocument";
import {
  createConnection,
  BrowserMessageReader,
  BrowserMessageWriter,
  TextDocuments,
} from "vscode-languageserver/browser";

export const initializeConnection = (
  context: DedicatedWorkerGlobalScope | Worker
) => {
  const messageReader = new BrowserMessageReader(context);
  const messageWriter = new BrowserMessageWriter(context);

  const connection = createConnection(messageReader, messageWriter);

  const documents = new TextDocuments(TextDocument);
  documents.listen(connection);

  return { connection, documents };
};
