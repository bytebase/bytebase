import "@codingame/monaco-vscode-javascript-default-extension";
import getLanguagesServiceOverride from "@codingame/monaco-vscode-languages-service-override";
import "@codingame/monaco-vscode-sql-default-extension";
import getTextMateServiceOverride from "@codingame/monaco-vscode-textmate-service-override";
import "@codingame/monaco-vscode-theme-defaults-default-extension";
import getThemeServiceOverride from "@codingame/monaco-vscode-theme-service-override";
import "vscode/localExtensionHost";

type WorkerLoader = () => Worker;

const workerLoaders: Partial<Record<string, WorkerLoader>> = {
  editorWorkerService: () =>
    new Worker(
      new URL("monaco-editor/esm/vs/editor/editor.worker.js", import.meta.url),
      { type: "module" }
    ),
  TextEditorWorker: () =>
    new Worker(
      new URL("monaco-editor/esm/vs/editor/editor.worker.js", import.meta.url),
      { type: "module" }
    ),
  TextMateWorker: () =>
    new Worker(
      new URL(
        "@codingame/monaco-vscode-textmate-service-override/worker",
        import.meta.url
      ),
      { type: "module" }
    ),
  textMateWorker: () =>
    new Worker(
      new URL(
        "@codingame/monaco-vscode-textmate-service-override/worker",
        import.meta.url
      ),
      { type: "module" }
    ),
};

const previousEnvironment = window.MonacoEnvironment;
const previousGetWorker = previousEnvironment?.getWorker;

window.MonacoEnvironment = {
  ...previousEnvironment,
  getWorker: function (moduleId, label) {
    const workerFactory = workerLoaders[label];
    if (workerFactory) {
      return workerFactory();
    }
    if (previousGetWorker) {
      return previousGetWorker(moduleId, label);
    }
    throw new Error(`Worker ${label} not found`);
  },
};

// `@codingame/monaco-vscode-api` `initialize` is a *global* one-shot: a second
// call rejects and leaves the editor spinner stuck forever until a hard
// refresh (BYT-9242), so every caller must await the same promise.
const GLOBAL_INIT_KEY = "__bytebaseMonacoServicesInitPromise__";

type GlobalWithMonacoInit = typeof globalThis & {
  [GLOBAL_INIT_KEY]?: Promise<void>;
};

const initializeRunner = async () => {
  const { initialize: initializeServices } = await import(
    "@codingame/monaco-vscode-api"
  );
  await initializeServices({
    ...getTextMateServiceOverride(),
    ...getThemeServiceOverride(),
    ...getLanguagesServiceOverride(),
  });
};

export const initializeMonacoServices = async (): Promise<void> => {
  const g = globalThis as GlobalWithMonacoInit;
  if (g[GLOBAL_INIT_KEY]) {
    return g[GLOBAL_INIT_KEY];
  }

  const job = initializeRunner().catch((err) => {
    // Allow a retry on the next call instead of caching a rejected promise.
    if (g[GLOBAL_INIT_KEY] === job) {
      g[GLOBAL_INIT_KEY] = undefined;
    }
    throw err;
  });
  g[GLOBAL_INIT_KEY] = job;
  return job;
};
