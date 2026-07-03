import { AISetting_Provider } from "@/types/proto-es/v1/setting_service_pb";

export const PROVIDER_DEFAULTS: Record<
  AISetting_Provider,
  { apiKeyDoc: string; endpoint: string; model: string }
> = {
  [AISetting_Provider.OPEN_AI]: {
    apiKeyDoc: "https://platform.openai.com/account/api-keys",
    endpoint: "https://api.openai.com/v1/chat/completions",
    model: "gpt-5.5",
  },
  [AISetting_Provider.AZURE_OPENAI]: {
    apiKeyDoc: "https://ai.azure.com/",
    endpoint:
      "https://{resource name}.openai.azure.com/openai/deployments/{deployment id}/chat/completions?api-version=2024-06-01",
    model: "gpt-5.5",
  },
  [AISetting_Provider.GEMINI]: {
    apiKeyDoc: "https://ai.google.dev/gemini-api/docs",
    endpoint: "https://generativelanguage.googleapis.com/v1beta",
    model: "gemini-3.5-flash",
  },
  [AISetting_Provider.CLAUDE]: {
    apiKeyDoc: "https://docs.anthropic.com/en/api/getting-started",
    endpoint: "https://api.anthropic.com/v1/messages",
    model: "claude-sonnet-5",
  },
  [AISetting_Provider.PROVIDER_UNSPECIFIED]: {
    apiKeyDoc: "",
    endpoint: "",
    model: "",
  },
};
