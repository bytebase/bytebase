import type * as monaco from "monaco-editor";
import { defer } from "@/utils";

// Lazy load monaco-editor to reduce initial bundle size
let monacoModule: typeof monaco | undefined;
const monacoLoadDefer = defer<typeof monaco>();

export const loadMonacoEditor = async (): Promise<typeof monaco> => {
  if (monacoModule) {
    return monacoModule;
  }

  // Dynamic import of monaco-editor (will create a separate chunk)
  monacoModule = await import("monaco-editor");
  monacoLoadDefer.resolve(monacoModule);

  return monacoModule;
};

export const getMonacoEditor = async (): Promise<typeof monaco> => {
  return monacoLoadDefer.promise;
};

// Helper to check if monaco is already loaded
export const isMonacoLoaded = (): boolean => {
  return monacoModule !== undefined;
};
