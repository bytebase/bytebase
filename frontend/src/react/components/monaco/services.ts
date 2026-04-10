import "@codingame/monaco-vscode-javascript-default-extension";
import getLanguagesServiceOverride from "@codingame/monaco-vscode-languages-service-override";
import "@codingame/monaco-vscode-sql-default-extension";
import getTextMateServiceOverride from "@codingame/monaco-vscode-textmate-service-override";
import "@codingame/monaco-vscode-theme-defaults-default-extension";
import getThemeServiceOverride from "@codingame/monaco-vscode-theme-service-override";
import "vscode/localExtensionHost";

type WorkerLoader = () => Worker;

const workerLoaders: Partial<Record<string, WorkerLoader>> = {
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
};

window.MonacoEnvironment = {
  getWorker: function (_moduleId, label) {
    const workerFactory = workerLoaders[label];
    if (workerFactory != null) {
      return workerFactory();
    }
    throw new Error(`Worker ${label} not found`);
  },
};

const state = {
  servicesInitialized: undefined as Promise<void> | undefined,
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

export const initializeMonacoServices = async () => {
  if (state.servicesInitialized) {
    return state.servicesInitialized;
  }

  const job = initializeRunner();
  state.servicesInitialized = job;
  return job;
};
