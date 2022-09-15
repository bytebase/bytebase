export const createLanguageServerWorker = async () => {
  const { default: LSPWorker } = await import("./server.ts?worker");

  const worker = new LSPWorker();
  return worker;
};
