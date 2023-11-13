import { languages } from "monaco-editor";
import { initServices } from "monaco-languageclient";

const state = {
  servicesInitialized: undefined as Promise<void> | undefined,
};

export const SupportedLanguages: languages.ILanguageExtensionPoint[] = [
  {
    id: "sql",
    extensions: [".sql"],
    aliases: ["SQL", "sql"],
    mimetypes: ["application/x-sql"],
  },
  {
    id: "javascript",
    extensions: [".js"],
    aliases: ["JS", "js"],
    mimetypes: ["application/javascript"],
  },
  {
    id: "redis",
    extensions: [".redis"],
    aliases: ["REDIS", "redis"],
    mimetypes: ["application/redis"],
  },
];

const initializeRunner = async () => {
  await initServices({
    enableThemeService: false,
    enableTextmateService: false,
    enableModelService: true,
    configureEditorOrViewsService: {},
    // enableKeybindingsService: true,
    enableLanguagesService: true,
    enableOutputService: true,
    enableAccessibilityService: true,
    debugLogging: false,
  });

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
