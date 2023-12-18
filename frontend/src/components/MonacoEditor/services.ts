import { languages } from "monaco-editor";
import { initialize as initializeServices } from "vscode/services";
import { SupportedLanguages } from "./types";

const state = {
  servicesInitialized: undefined as Promise<void> | undefined,
};

const initializeRunner = async () => {
  await initializeServices({});

  SupportedLanguages.forEach((lang) => languages.register(lang));
};

export const initializeMonacoServices = async () => {
  if (state.servicesInitialized) {
    return state.servicesInitialized;
  }

  const job = initializeRunner();
  state.servicesInitialized = job;
  return job;
};
