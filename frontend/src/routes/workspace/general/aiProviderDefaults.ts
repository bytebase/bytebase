import { AISetting_Provider } from "@/types/proto-es/v1/setting_service_pb";

export interface ProviderModel {
  endpoint: string;
  value: string;
}

const OPENAI_RESPONSES_ENDPOINT = "https://api.openai.com/v1/responses";
const OPENAI_CHAT_COMPLETIONS_ENDPOINT =
  "https://api.openai.com/v1/chat/completions";
const AZURE_OPENAI_RESPONSES_ENDPOINT =
  "https://{resource name}.openai.azure.com/openai/v1/responses";
const AZURE_OPENAI_CHAT_COMPLETIONS_ENDPOINT =
  "https://{resource name}.openai.azure.com/openai/v1/chat/completions";

export const PROVIDER_MODELS: Record<AISetting_Provider, ProviderModel[]> = {
  [AISetting_Provider.OPEN_AI]: [
    { value: "gpt-5.5", endpoint: OPENAI_RESPONSES_ENDPOINT },
    { value: "gpt-5", endpoint: OPENAI_RESPONSES_ENDPOINT },
    { value: "gpt-5-mini", endpoint: OPENAI_RESPONSES_ENDPOINT },
    { value: "gpt-4o", endpoint: OPENAI_CHAT_COMPLETIONS_ENDPOINT },
  ],
  [AISetting_Provider.AZURE_OPENAI]: [
    { value: "gpt-5.5", endpoint: AZURE_OPENAI_RESPONSES_ENDPOINT },
    { value: "gpt-5", endpoint: AZURE_OPENAI_RESPONSES_ENDPOINT },
    { value: "gpt-5-mini", endpoint: AZURE_OPENAI_RESPONSES_ENDPOINT },
    { value: "gpt-4o", endpoint: AZURE_OPENAI_CHAT_COMPLETIONS_ENDPOINT },
  ],
  [AISetting_Provider.GEMINI]: [
    {
      value: "gemini-3.5-flash",
      endpoint: "https://generativelanguage.googleapis.com/v1beta",
    },
  ],
  [AISetting_Provider.CLAUDE]: [
    {
      value: "claude-sonnet-5",
      endpoint: "https://api.anthropic.com/v1/messages",
    },
  ],
  [AISetting_Provider.PROVIDER_UNSPECIFIED]: [],
};

export function getModelEndpoint(
  provider: AISetting_Provider,
  currentEndpoint: string,
  model: ProviderModel
): string {
  if (provider !== AISetting_Provider.AZURE_OPENAI) {
    return model.endpoint;
  }
  if (
    currentEndpoint.includes("{resource name}") ||
    currentEndpoint.includes("{resource%20name}")
  ) {
    return model.endpoint;
  }
  try {
    const endpoint = new URL(currentEndpoint);
    const modelEndpoint = new URL(
      model.endpoint.replace("{resource name}", "resource")
    );
    endpoint.pathname = modelEndpoint.pathname;
    endpoint.search = modelEndpoint.search;
    return endpoint.toString();
  } catch {
    return model.endpoint;
  }
}

export const PROVIDER_DEFAULTS: Record<
  AISetting_Provider,
  { apiKeyDoc: string; endpoint: string; model: string }
> = {
  [AISetting_Provider.OPEN_AI]: {
    apiKeyDoc: "https://platform.openai.com/account/api-keys",
    endpoint: OPENAI_RESPONSES_ENDPOINT,
    model: "gpt-5.5",
  },
  [AISetting_Provider.AZURE_OPENAI]: {
    apiKeyDoc: "https://ai.azure.com/",
    endpoint: AZURE_OPENAI_RESPONSES_ENDPOINT,
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
